package httpservice

import (
	"bufio"
	"bytes"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gptankit/serviceq/algorithm"
	"github.com/gptankit/serviceq/errorlog"
	"github.com/gptankit/serviceq/model"
	"github.com/gptankit/serviceq/tcputils"
)

var _ model.NetService = &HTTPService{}

// HTTPService is the core http flow handler
type HTTPService struct {
	inTCPConn     *net.Conn
	inTCPReader   *bufio.Reader
	inTCPWriter   *bufio.Writer
	outHTTPClient *http.Client
	properties    *model.ServiceQProperties
}

type HTTPServiceOption func(*HTTPService) error

// New initializes new HTTPService and sets up the upstream client
func New(sqp *model.ServiceQProperties, httpSrvOptions ...HTTPServiceOption) *HTTPService {

	httpSrv := new(HTTPService)
	httpSrv.outHTTPClient = &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:    200,
			IdleConnTimeout: 30 * time.Second,
		},
		Timeout: time.Duration(sqp.OutRequestTimeout) * time.Second,
	}
	httpSrv.properties = sqp

	for _, httpSrvOption := range httpSrvOptions {
		if err := httpSrvOption(httpSrv); err != nil {
			httpSrv = nil
			return nil
		}
	}

	return httpSrv
}

// WithIncomingTCPConn assigns a tcp conn and binds reader/writer to http service
func WithIncomingTCPConn(tcpConn *net.Conn) HTTPServiceOption {

	return func(httpSrv *HTTPService) error {

		httpSrv.inTCPConn = tcpConn
		httpSrv.inTCPReader = bufio.NewReader(*httpSrv.inTCPConn)
		httpSrv.inTCPWriter = bufio.NewWriter(*httpSrv.inTCPConn)

		return nil
	}
}

// NewNop returns a HTTPService that does nothing
func NewNop(sqp *model.ServiceQProperties) *HTTPService {

	httpSrv := &HTTPService{
		inTCPConn:     nil,
		inTCPReader:   nil,
		inTCPWriter:   nil,
		outHTTPClient: nil,
		properties:    sqp,
	}

	return httpSrv
}

// Read reads http request from reader
func (httpSrv *HTTPService) Read() (interface{}, error) {

	req, err := http.ReadRequest(httpSrv.inTCPReader)

	if err == nil {
		return req, nil
	}

	return nil, errors.New("read-fail")
}

// Write writes http response to writer (in http format)
func (httpSrv *HTTPService) Write(resp interface{}) error {

	res, ok := resp.(model.ResponseParam)
	if !ok {
		return errors.New("invalid-restype")
	}

	responseHeaders := ""

	// add original response headers
	if res.Headers != nil {
		for k, v := range res.Headers {
			responseHeaders += k + ": " + strings.Join(v, ",") + "\n"
		}
	}

	// add user custom headers
	for _, h := range httpSrv.properties.CustomResponseHeaders {
		responseHeaders += h + "\n"
	}

	if responseHeaders != "" {
		responseHeaders = responseHeaders[:len(responseHeaders)-1]
		res.Status = res.Status + "\n"
	}

	clientResStr := res.Protocol + " " + res.Status + responseHeaders + "\n\n" + string(res.BodyBuff)

	clientRes := []byte(clientResStr)

	_, err := httpSrv.inTCPWriter.Write(clientRes) // tunneling onto tcp conn writer
	if err == nil {
		httpSrv.inTCPWriter.Flush()
		return nil
	}

	return errors.New("write-fail")
}

// ExecuteRealTime reads from incoming http connection and attempts to forward it to upstream nodes by calling
// dialAndSend(). It temporarily saves the request before forwarding, if needed for subsequent retries. This saved
// request can be buffered if dialAndSend() is unable to forward to any upstream nodes.
func (httpSrv *HTTPService) ExecuteRealTime(creq chan interface{}, cwork chan int) {

	tcputils.SetTCPDeadline(httpSrv.inTCPConn, httpSrv.properties.KeepAliveTimeout)

	for {

		var resParam model.ResponseParam
		var reqParam model.RequestParam
		var toBuffer bool

		// read from and write to conn
		reqp, err := httpSrv.Read()
		req, ok := reqp.(*http.Request)
		if !ok {
			break
		}

		if err == nil {
			// add work
			cwork <- 1

			reqParam = httpSrv.saveReqParam(req)
			toBuffer = httpSrv.properties.EnableUpfrontQ && httpSrv.canBeBuffered(reqParam)
			if !toBuffer {
				resParam, toBuffer, err = httpSrv.dialAndSend(reqParam)
				if err == nil {
					err = httpSrv.Write(resParam)
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
			if httpSrv.optCloseConn(reqParam) {
				break
			}
		} else {
			//fmt.Fprintf(os.Stderr, "Error on reading from client conn\n")
			go errorlog.LogGenericError("Error on reading from client conn")
			httpSrv.forceCloseConn()
			break
		}
	}
}

// ExecuteBuffered retries buffered requests by calling dialAndSend()
func (httpSrv *HTTPService) ExecuteBuffered(creq chan interface{}, cwork chan int) {

	for {
		if len(cwork) > 0 && len(creq) > 0 {

			reqParam := (<-creq).(model.RequestParam)
			// send from buffer
			_, toBuffer, _ := httpSrv.dialAndSend(reqParam)

			// to buffer?
			if toBuffer {
				creq <- reqParam
				cwork <- 1
			}

			// remove work
			<-cwork

		} else {
			time.Sleep(time.Duration(httpSrv.properties.IdleGap) * time.Millisecond) // wait for more work
		}
	}

}

// Discard sets error response and discards client http connection
func (httpSrv *HTTPService) Discard() {

	var resParam model.ResponseParam
	reqp, err := httpSrv.Read()
	req, ok := reqp.(*http.Request)
	if !ok {
		return
	}

	if err == nil {
		resParam = httpSrv.getCustomResponse(req.Proto, http.StatusTooManyRequests, "Request Discarded")
		clientErr := errors.New(tcputils.RESPONSE_FLOODED)
		go errorlog.IncrementErrorCount(httpSrv.properties, "SQ_PROXY", tcputils.SERVICEQ_FLOODED_ERR, clientErr.Error())
		err = httpSrv.Write(resParam)
		if err != nil {
			//fmt.Fprintf(os.Stderr, "Error on writing to client conn\n")
			go errorlog.LogGenericError("Error on writing to client conn")
		}
	}

	go errorlog.LogGenericError("Error on reading from client conn")
	httpSrv.forceCloseConn()
}

// saveReqParam parses and temporarily saves http request
func (httpSrv *HTTPService) saveReqParam(req *http.Request) model.RequestParam {

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

// dialAndSend forwards request to upstream node selected by ChooseServiceIndex() and in case of
// error, increments the error count, and retries for a maximum MaxRetries times. If the request succeedes,
// the coresponding node error count is reset. If the request fails on all nodes, it can be set to buffer.
func (httpSrv *HTTPService) dialAndSend(reqParam model.RequestParam) (model.ResponseParam, bool, error) {

	choice := -1
	var nodeErr error

	for retry := 0; retry < httpSrv.properties.MaxRetries; retry++ {

		choice = algorithm.ChooseServiceIndex(httpSrv.properties, choice, retry)
		upstrService := httpSrv.properties.ServiceList[choice]

		body := ioutil.NopCloser(bytes.NewReader(reqParam.BodyBuff))
		upstrReq, _ := http.NewRequest(reqParam.Method, upstrService.QualifiedUrl+reqParam.RequestURI, body)
		upstrReq.Header = reqParam.Headers

		resp, err := httpSrv.outHTTPClient.Do(upstrReq)

		// handle response
		if resp == nil || err != nil {
			nodeErr = tcputils.EvalError(err)
			go errorlog.IncrementErrorCount(httpSrv.properties, upstrService.QualifiedUrl, tcputils.UPSTREAM_HTTP_ERR, nodeErr.Error())

			time.Sleep(time.Duration(httpSrv.properties.RetryGap) * time.Second) // wait on error
			continue
		} else {
			nodeErr = nil
			go errorlog.ResetErrorCount(httpSrv.properties, upstrService.QualifiedUrl)

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
		return httpSrv.checkErrorAndRespond(nodeErr, reqParam)
	}

	return model.ResponseParam{}, true, errors.New("send-fail")
}

// checkErrorAndRespond sets error and buffer flag based on buffer config and type of error from upstream node
func (httpSrv *HTTPService) checkErrorAndRespond(clientErr error, reqParam model.RequestParam) (model.ResponseParam, bool, error) {

	if clientErr.Error() == tcputils.RESPONSE_NO_RESPONSE || clientErr.Error() == tcputils.RESPONSE_TIMED_OUT {
		if httpSrv.properties.EnableDeferredQ && httpSrv.canBeBuffered(reqParam) {
			return httpSrv.getCustomResponse(reqParam.Protocol, http.StatusServiceUnavailable, "Request Buffered"), true, nil
		} else {
			return httpSrv.getCustomResponse(reqParam.Protocol, http.StatusServiceUnavailable, ""), false, nil
		}
	} else if clientErr.Error() == tcputils.RESPONSE_SERVICE_DOWN {
		return httpSrv.getCustomResponse(reqParam.Protocol, http.StatusServiceUnavailable, ""), false, nil
	} else {
		return httpSrv.getCustomResponse(reqParam.Protocol, http.StatusBadGateway, ""), false, nil
	}
}

// getCustomResponse creates a new http response with appropriates status code and response message
func (httpSrv *HTTPService) getCustomResponse(protocol string, statusCode int, resMsg string) model.ResponseParam {

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
func (httpSrv *HTTPService) canBeBuffered(reqParam model.RequestParam) bool {

	reqFormats := httpSrv.properties.QRequestFormats

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

// optCloseConn determines to optionally close a net.Conn object
func (httpSrv *HTTPService) optCloseConn(reqParam model.RequestParam) bool {

	if reqParam.Protocol == "HTTP/1.0" || reqParam.Protocol == "HTTP/1.1" { // Connection and keep-alive are ignored for http/2
		if v, ok := reqParam.Headers["Connection"]; ok {
			if v[0] == "keep-alive" && httpSrv.properties.KeepAliveServe {
				return false // do not close conn
			} else if v[0] == "close" || httpSrv.properties.KeepAliveServe { // close conn if Connection: close or keep-alive is not part of response
				return httpSrv.forceCloseConn()
			}
		} else if reqParam.Protocol == "HTTP/1.1" && httpSrv.properties.KeepAliveServe {
			return false // do not close conn if Connection header not found for protocol http/1.1 -- follow default behaviour
		} else {
			return httpSrv.forceCloseConn()
		}
	}

	return false
}

// forceCloseConn force closes a net.Conn object
func (httpSrv *HTTPService) forceCloseConn() bool {

	(*httpSrv.inTCPConn).Close()
	return true
}
