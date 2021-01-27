package protocol

import (
	"bytes"
	"errors"
	"github.com/gptankit/serviceq/algorithm"
	"github.com/gptankit/serviceq/errorlog"
	"github.com/gptankit/serviceq/model"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
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

const (
	SERVICEQ_NO_ERR      = 600
	SERVICEQ_FLOODED_ERR = 601
	UPSTREAM_NO_ERR      = 700
	UPSTREAM_TCP_ERR     = 701
	UPSTREAM_HTTP_ERR    = 702

	RESPONSE_FLOODED      = "SERVICEQ_FLOODED"
	RESPONSE_TIMED_OUT    = "UPSTREAM_TIMED_OUT"
	RESPONSE_SERVICE_DOWN = "UPSTREAM_DOWN"
	RESPONSE_NO_RESPONSE  = "UPSTREAM_NO_RESPONSE"
)

// HandleHttpConnection reads from incoming http connection and attempts to forward it to upstream nodes by calling
// dialAndSend(). It temporarily saves the request before forwarding, if needed for subsequent retries. This saved
// request can be buffered if dialAndSend() is unable to forward to any upstream nodes.
func HandleHttpConnection(conn *net.Conn, creq chan interface{}, cwork chan int, sqp *model.ServiceQProperties) {

	httpConn := model.HTTPConnection{}
	setTCPDeadline(conn, sqp.KeepAliveTimeout)
	httpConn.Enclose(conn)

	for {

		var resParam model.ResponseParam
		var reqParam model.RequestParam
		var toBuffer bool

		// read from and write to conn
		req, err := httpConn.ReadFrom()

		if err == nil {
			// add work
			cwork <- 1

			reqParam = saveReqParam(req)
			toBuffer = sqp.EnableUpfrontQ && canBeBuffered(reqParam, sqp)
			if !toBuffer {
				resParam, toBuffer, err = dialAndSend(reqParam, sqp)
				if err == nil {
					err = httpConn.WriteTo(resParam, sqp.CustomResponseHeaders)
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
			if optCloseConn(conn, reqParam, sqp.KeepAliveServe) {
				break
			}
		} else {
			//fmt.Fprintf(os.Stderr, "Error on reading from client conn\n")
			go errorlog.LogGenericError("Error on reading from client conn")
			forceCloseConn(conn)
			break
		}
	}

	return
}

// DiscardHttpConnection sets error response and discards client http connection.
func DiscardHttpConnection(conn *net.Conn, sqp *model.ServiceQProperties) {

	var resParam model.ResponseParam
	httpConn := model.HTTPConnection{}
	httpConn.Enclose(conn)
	req, err := httpConn.ReadFrom()

	if err == nil {
		resParam = getCustomResponse(req.Proto, http.StatusTooManyRequests, "Request Discarded")
		clientErr := errors.New(RESPONSE_FLOODED)
		go errorlog.IncrementErrorCount(sqp, "SQ_PROXY", SERVICEQ_FLOODED_ERR, clientErr.Error())
		err = httpConn.WriteTo(resParam, sqp.CustomResponseHeaders)
		if err != nil {
			//fmt.Fprintf(os.Stderr, "Error on writing to client conn\n")
			go errorlog.LogGenericError("Error on writing to client conn")
		}
	}

	go errorlog.LogGenericError("Error on reading from client conn")
	forceCloseConn(conn)

	return
}

// HandleHttpBufferedReader retries buffered requests by calling dialAndSend().
func HandleHttpBufferedReader(reqParam model.RequestParam, creq chan interface{}, cwork chan int, sqp *model.ServiceQProperties) {

	// send from buffer
	_, toBuffer, _ := dialAndSend(reqParam, sqp)

	// to buffer?
	if toBuffer {
		creq <- reqParam
		cwork <- 1
	}

	// remove work
	<-cwork

	return
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
		downstrService := sqp.ServiceList[choice]

		// fmt.Printf("%s] Connecting to %s\n", time.Now().UTC().Format("2006-01-02 15:04:05"), downstrService.Host)

		// ping ip -- response/error flow below will take care of tcp connect
		/*
			if !isTCPAlive(downstrService.Host) {
				clientErr = errors.New(RESPONSE_SERVICE_DOWN)
				go errorlog.IncrementErrorCount(sqp, downstrService.QualifiedUrl, UPSTREAM_TCP_ERR, clientErr.Error())
				time.Sleep(time.Duration(sqp.RetryGap) * time.Second) // wait on error
				continue
			}*/

		//fmt.Printf("->Forwarding to %s\n", downstrService.QualifiedUrl)

		body := ioutil.NopCloser(bytes.NewReader(reqParam.BodyBuff))
		downstrReq, _ := http.NewRequest(reqParam.Method, downstrService.QualifiedUrl+reqParam.RequestURI, body)
		downstrReq.Header = reqParam.Headers

		once.Do(func() {
			client.Timeout = time.Duration(sqp.OutRequestTimeout) * time.Second
		})
		resp, err := client.Do(downstrReq)

		// handle response
		if resp == nil || err != nil {
			nodeErr = evalError(err)
			go errorlog.IncrementErrorCount(sqp, downstrService.QualifiedUrl, UPSTREAM_HTTP_ERR, nodeErr.Error())

			time.Sleep(time.Duration(sqp.RetryGap) * time.Second) // wait on error
			continue
		} else {
			nodeErr = nil
			go errorlog.ResetErrorCount(sqp, downstrService.QualifiedUrl)

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
			nodeErr = errors.New(RESPONSE_TIMED_OUT)
		} else {
			nodeErr = errors.New(RESPONSE_NO_RESPONSE)
		}
	} else {
		nodeErr = errors.New(RESPONSE_NO_RESPONSE)
	}

	return nodeErr
}

// checkErrorAndRespond sets error and buffer flag based on buffer config and type of error from upstream node.
func checkErrorAndRespond(clientErr error, reqParam model.RequestParam, sqp *model.ServiceQProperties) (model.ResponseParam, bool, error) {

	if clientErr.Error() == RESPONSE_NO_RESPONSE || clientErr.Error() == RESPONSE_TIMED_OUT {
		if sqp.EnableDeferredQ && canBeBuffered(reqParam, sqp) {
			return getCustomResponse(reqParam.Protocol, http.StatusServiceUnavailable, "Request Buffered"), true, nil
		} else {
			return getCustomResponse(reqParam.Protocol, http.StatusServiceUnavailable, ""), false, nil
		}
	} else if clientErr.Error() == RESPONSE_SERVICE_DOWN {
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
