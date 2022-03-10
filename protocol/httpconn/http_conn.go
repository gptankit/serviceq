package httpconn

import (
	"bufio"
	"bytes"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gptankit/serviceq/algorithm"
	"github.com/gptankit/serviceq/errorlog"
	"github.com/gptankit/serviceq/model"
	"github.com/gptankit/serviceq/tcputils"
)

var client *http.Client
var once sync.Once

func init() {

	client = &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:    200,
			IdleConnTimeout: 30 * time.Second},
	}
}

// HTTPConnection is a http connection object that holds underlying
// tcp connection with reader and writer to that connection.
type HTTPConnection struct {
	tcpConn *net.Conn
	reader  *bufio.Reader
	writer  *bufio.Writer
}

// Enclose initializes new http reader/writer to underlying tcp connection.
func New(tcpConn *net.Conn) *HTTPConnection {

	httpConn := new(HTTPConnection)
	httpConn.tcpConn = tcpConn
	httpConn.reader = bufio.NewReader(*httpConn.tcpConn)
	httpConn.writer = bufio.NewWriter(*httpConn.tcpConn)

	return httpConn
}

func NewNop() *HTTPConnection {

	httpConn := &HTTPConnection{
		tcpConn: nil,
		reader:  nil,
		writer:  nil,
	}

	return httpConn
}

// Read reads http request from reader.
func (httpConn *HTTPConnection) Read() (*http.Request, error) {

	req, err := http.ReadRequest(httpConn.reader)

	if err == nil {
		return req, nil
	}

	return nil, errors.New("read-fail")
}

// Write writes http response to writer (in http format).
func (httpConn *HTTPConnection) Write(res model.ResponseParam, customHeaders []string) error {

	responseHeaders := ""

	// add original response headers
	if res.Headers != nil {
		for k, v := range res.Headers {
			responseHeaders += k + ": " + strings.Join(v, ",") + "\n"
		}
	}

	// add user custom headers
	for _, h := range customHeaders {
		responseHeaders += h + "\n"
	}

	if responseHeaders != "" {
		responseHeaders = responseHeaders[:len(responseHeaders)-1]
		res.Status = res.Status + "\n"
	}

	clientResStr := res.Protocol + " " + res.Status + responseHeaders + "\n\n" + string(res.BodyBuff)

	clientRes := []byte(clientResStr)

	_, err := httpConn.writer.Write(clientRes) // tunneling onto tcp conn writer
	if err == nil {
		httpConn.writer.Flush()
		return nil
	}

	return errors.New("write-fail")
}

// ExecuteRealTime reads from incoming http connection and attempts to forward it to upstream nodes by calling
// dialAndSend(). It temporarily saves the request before forwarding, if needed for subsequent retries. This saved
// request can be buffered if dialAndSend() is unable to forward to any upstream nodes.
func (httpConn *HTTPConnection) ExecuteRealTime(creq chan interface{}, cwork chan int, sqp *model.ServiceQProperties) {

	tcputils.SetTCPDeadline(httpConn.tcpConn, sqp.KeepAliveTimeout)

	for {

		var resParam model.ResponseParam
		var reqParam model.RequestParam
		var toBuffer bool

		// read from and write to conn
		req, err := httpConn.Read()

		if err == nil {
			// add work
			cwork <- 1

			reqParam = saveReqParam(req)
			toBuffer = sqp.EnableUpfrontQ && canBeBuffered(reqParam, sqp)
			if !toBuffer {
				resParam, toBuffer, err = dialAndSend(reqParam, sqp)
				if err == nil {
					err = httpConn.Write(resParam, sqp.CustomResponseHeaders)
					if err != nil {
						//fmt.Fprintf(os.Stderr, "Error on writing to client conn\n")
						go errorlog.LogGenericError("Error on writing to client conn")
					}
				}
			}

			// to buffer?
			if toBuffer {
				creq <- reqParam
				cwork <- 1
			}

			// remove work
			<-cwork

			// check if a conn is to be closed
			if optCloseConn(httpConn.tcpConn, reqParam, sqp.KeepAliveServe) {
				break
			}
		} else {
			//fmt.Fprintf(os.Stderr, "Error on reading from client conn\n")
			go errorlog.LogGenericError("Error on reading from client conn")
			forceCloseConn(httpConn.tcpConn)
			break
		}
	}
}

// ExecuteBuffered retries buffered requests by calling dialAndSend().
func (httpConn *HTTPConnection) ExecuteBuffered(creq chan interface{}, cwork chan int, sqp *model.ServiceQProperties) {

	for {
		if len(cwork) > 0 && len(creq) > 0 {

			reqParam := (<-creq).(model.RequestParam)
			// send from buffer
			_, toBuffer, _ := dialAndSend(reqParam, sqp)

			// to buffer?
			if toBuffer {
				creq <- reqParam
				cwork <- 1
			}

			// remove work
			<-cwork

		} else {
			time.Sleep(time.Duration(sqp.IdleGap) * time.Millisecond) // wait for more work
		}
	}

}

// Discard sets error response and discards client http connection.
func (httpConn *HTTPConnection) Discard(sqp *model.ServiceQProperties) {

	var resParam model.ResponseParam
	req, err := httpConn.Read()

	if err == nil {
		resParam = getCustomResponse(req.Proto, http.StatusTooManyRequests, "Request Discarded")
		clientErr := errors.New(tcputils.RESPONSE_FLOODED)
		go errorlog.IncrementErrorCount(sqp, "SQ_PROXY", tcputils.SERVICEQ_FLOODED_ERR, clientErr.Error())
		err = httpConn.Write(resParam, sqp.CustomResponseHeaders)
		if err != nil {
			//fmt.Fprintf(os.Stderr, "Error on writing to client conn\n")
			go errorlog.LogGenericError("Error on writing to client conn")
		}
	}

	go errorlog.LogGenericError("Error on reading from client conn")
	forceCloseConn(httpConn.tcpConn)
}

// saveReqParam parses and temporarily saves http request.
func saveReqParam(req *http.Request) model.RequestParam {

	var reqParam model.RequestParam

	reqParam.Protocol = req.Proto
	reqParam.Method = req.Method

	if req.URL.RawQuery != "" {
		reqParam.RequestURI = req.URL.Path + "?" + req.URL.RawQuery
	} else {
		reqParam.RequestURI = req.URL.Path
	}

	if req.Body != nil {
		if bodyBuff, err := ioutil.ReadAll(req.Body); err == nil {
			reqParam.BodyBuff = bodyBuff
		}
	}

	if req.Header != nil {
		reqParam.Headers = make(map[string][]string, len(req.Header))
		for k, v := range req.Header {
			reqParam.Headers[k] = v
		}
	}

	return reqParam
}

// dialAndSend forwards request to upstream node selected by algorithm.ChooseServiceIndex() and in case of
// error, increments the error count, and retries for a maximum sqp.MaxRetries times. If the request succeedes,
// the coresponding node error count is reset. If the request fails on all nodes, it can be set to buffer.
func dialAndSend(reqParam model.RequestParam, sqp *model.ServiceQProperties) (model.ResponseParam, bool, error) {

	choice := -1
	var nodeErr error

	for retry := 0; retry < sqp.MaxRetries; retry++ {

		choice = algorithm.ChooseServiceIndex(sqp, choice, retry)
		upstrService := sqp.ServiceList[choice]

		// fmt.Printf("%s] Connecting to %s\n", time.Now().UTC().Format("2006-01-02 15:04:05"), upstrService.Host)

		// ping ip -- response/error flow below will take care of tcp connect
		/*
			if !isTCPAlive(upstrService.Host) {
				clientErr = errors.New(RESPONSE_SERVICE_DOWN)
				go errorlog.IncrementErrorCount(sqp, upstrService.QualifiedUrl, UPSTREAM_TCP_ERR, clientErr.Error())
				time.Sleep(time.Duration(sqp.RetryGap) * time.Second) // wait on error
				continue
			}*/

		//fmt.Printf("->Forwarding to %s\n", upstrService.QualifiedUrl)

		body := ioutil.NopCloser(bytes.NewReader(reqParam.BodyBuff))
		upstrReq, _ := http.NewRequest(reqParam.Method, upstrService.QualifiedUrl+reqParam.RequestURI, body)
		upstrReq.Header = reqParam.Headers

		once.Do(func() {
			client.Timeout = time.Duration(sqp.OutRequestTimeout) * time.Second
		})
		resp, err := client.Do(upstrReq)

		// handle response
		if resp == nil || err != nil {
			nodeErr = evalError(err)
			go errorlog.IncrementErrorCount(sqp, upstrService.QualifiedUrl, tcputils.UPSTREAM_HTTP_ERR, nodeErr.Error())

			time.Sleep(time.Duration(sqp.RetryGap) * time.Second) // wait on error
			continue
		} else {
			nodeErr = nil
			go errorlog.ResetErrorCount(sqp, upstrService.QualifiedUrl)

			// prepare response
			responseParam := model.ResponseParam{}
			responseParam.Protocol = resp.Proto
			responseParam.Status = resp.Status
			responseParam.Headers = resp.Header
			if resp.Body != nil {
				responseParam.BodyBuff, _ = ioutil.ReadAll(resp.Body)
				resp.Body.Close()
			}
			return responseParam, false, nil
		}
	}

	// error based response
	if nodeErr != nil {
		return checkErrorAndRespond(nodeErr, reqParam, sqp)
	}

	return model.ResponseParam{}, true, errors.New("send-fail")
}

// evalError evaluates the type of errors from upstream node.
func evalError(err error) error {

	nodeErr := err
	if nodeErr != nil {
		if e, ok := nodeErr.(net.Error); ok && e.Timeout() {
			nodeErr = errors.New(tcputils.RESPONSE_TIMED_OUT)
		} else {
			nodeErr = errors.New(tcputils.RESPONSE_NO_RESPONSE)
		}
	} else {
		nodeErr = errors.New(tcputils.RESPONSE_NO_RESPONSE)
	}

	return nodeErr
}

// checkErrorAndRespond sets error and buffer flag based on buffer config and type of error from upstream node.
func checkErrorAndRespond(clientErr error, reqParam model.RequestParam, sqp *model.ServiceQProperties) (model.ResponseParam, bool, error) {

	if clientErr.Error() == tcputils.RESPONSE_NO_RESPONSE || clientErr.Error() == tcputils.RESPONSE_TIMED_OUT {
		if sqp.EnableDeferredQ && canBeBuffered(reqParam, sqp) {
			return getCustomResponse(reqParam.Protocol, http.StatusServiceUnavailable, "Request Buffered"), true, nil
		} else {
			return getCustomResponse(reqParam.Protocol, http.StatusServiceUnavailable, ""), false, nil
		}
	} else if clientErr.Error() == tcputils.RESPONSE_SERVICE_DOWN {
		return getCustomResponse(reqParam.Protocol, http.StatusServiceUnavailable, ""), false, nil
	} else {
		return getCustomResponse(reqParam.Protocol, http.StatusBadGateway, ""), false, nil
	}
}

// getCustomResponse creates a new http response with appropriates status code and response message.
func getCustomResponse(protocol string, statusCode int, resMsg string) model.ResponseParam {

	var body []byte
	var json string
	if resMsg != "" {
		json = `{"sq_msg":"` + resMsg + `"}`
		body = []byte(json)
	}
	jsonLen := strconv.Itoa(len(json))

	return model.ResponseParam{
		Protocol: protocol,
		Status:   strconv.Itoa(statusCode) + " " + http.StatusText(statusCode),
		Headers:  http.Header{"Content-Type": []string{"application/json"}, "Content-Length": []string{jsonLen}},
		BodyBuff: body,
	}
}

// canBeBuffered determines whether a request is qualified for buffering based on buffer
// config in sq.properties. Http method and uri are matched against buffer config.
func canBeBuffered(reqParam model.RequestParam, sqp *model.ServiceQProperties) bool {

	reqFormats := sqp.QRequestFormats

	if reqFormats == nil || reqFormats[0] == "ALL" {
		return true
	}

	for _, rf := range reqFormats {
		satisfy := false
		rfBrkUp := strings.Split(rf, " ")
		if (0 < len(rfBrkUp) && reqParam.Method == rfBrkUp[0]) || (0 >= len(rfBrkUp)) {
			satisfy = true
			if (1 < len(rfBrkUp) && reqParam.RequestURI == rfBrkUp[1]) || (1 >= len(rfBrkUp)) {
				satisfy = true
				if 2 < len(rfBrkUp) && rfBrkUp[2] == "!" {
					satisfy = false
				}
			} else {
				satisfy = false
			}
		}
		if satisfy {
			return satisfy
		}
	}

	return false
}

// optCloseConn determines to optionally close a net.Conn object.
func optCloseConn(conn *net.Conn, reqParam model.RequestParam, keepAliveServe bool) bool {

	if reqParam.Protocol == "HTTP/1.0" || reqParam.Protocol == "HTTP/1.1" { // Connection and keep-alive are ignored for http/2
		if v, ok := reqParam.Headers["Connection"]; ok {
			if v[0] == "keep-alive" && keepAliveServe {
				return false // do not close conn
			} else if v[0] == "close" || !keepAliveServe { // close conn if Connection: close or keep-alive is not part of response
				return forceCloseConn(conn)
			}
		} else if reqParam.Protocol == "HTTP/1.1" && keepAliveServe {
			return false // do not close conn if Connection header not found for protocol http/1.1 -- follow default behaviour
		} else {
			return forceCloseConn(conn)
		}
	}

	return false
}

// forceCloseConn force closes a net.Conn object.
func forceCloseConn(conn *net.Conn) bool {

	(*conn).Close()
	return true
}
