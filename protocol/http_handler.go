package protocol

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/gptankit/serviceq/algorithm"
	"github.com/gptankit/serviceq/errorlog"
	"github.com/gptankit/serviceq/model"
	"io/ioutil"
	"net"
	"net/http"
	"os"
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
	DOWNSTREAM_NO_ERR    = 700
	DOWNSTREAM_TCP_ERR   = 701
	DOWNSTREAM_HTTP_ERR  = 702

	RESPONSE_FLOODED      = "SERVICEQ_FLOODED"
	RESPONSE_TIMED_OUT    = "DOWNSTREAM_TIMED_OUT"
	RESPONSE_SERVICE_DOWN = "DOWNSTREAM_DOWN"
	RESPONSE_NO_RESPONSE  = "DOWNSTREAM_NO_RESPONSE"
)

// HandleHttpConnection reads from incoming http connection and attempts to forward it to downstream nodes by calling
// dialAndSend(). It temporarily saves the request before forwarding, if needed for subsequent retries. This saved
// request can be buffered if dialAndSend() is unable to forward to any downstream nodes.
func HandleHttpConnection(conn *net.Conn, creq chan interface{}, cwork chan int, sqp *model.ServiceQProperties) {

	httpConn := model.HTTPConnection{}
	setTCPDeadline(conn, (*sqp).KeepAliveTimeout)
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
			resParam, toBuffer, err = dialAndSend(reqParam, sqp)
			if err == nil {
				err = httpConn.WriteTo(resParam, (*sqp).CustomResponseHeaders)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error on writing to client conn\n")
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
			forceCloseConn(conn)
			break
		}
	}

	return
}

// DiscardHttpConnection sets error response and discards upstream http connection.
func DiscardHttpConnection(conn *net.Conn, sqp *model.ServiceQProperties) {

	var resParam model.ResponseParam
	httpConn := model.HTTPConnection{}
	httpConn.Enclose(conn)
	req, err := httpConn.ReadFrom()

	if err == nil {
		resParam = getCustomResponse(req.Proto, http.StatusTooManyRequests, "Request Discarded")
		clientErr := errors.New(RESPONSE_FLOODED)
		errorlog.IncrementErrorCount(sqp, "SQ_PROXY", SERVICEQ_FLOODED_ERR, clientErr.Error())
		err = httpConn.WriteTo(resParam, (*sqp).CustomResponseHeaders)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error on writing to client conn\n")
		}
	}
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

// dialAndSend forwards request to downstream node selected by algorithm.ChooseServiceIndex() and in case of
// error, increments the error count, and retries for a maximum (*sqp).MaxRetries times. If the request succeedes,
// the coresponding node error count is reset. If the request fails on all nodes, it can be set to buffer.
func dialAndSend(reqParam model.RequestParam, sqp *model.ServiceQProperties) (model.ResponseParam, bool, error) {

	choice := -1
	var clientErr error

	for retry := 0; retry < (*sqp).MaxRetries; retry++ {

		choice = algorithm.ChooseServiceIndex(sqp, choice, retry)
		downstrService := (*sqp).ServiceList[choice]

		// fmt.Printf("%s] Connecting to %s\n", time.Now().UTC().Format("2006-01-02 15:04:05"), downstrService.Host)

		// ping ip -- response/error flow below will take care of tcp connect
		/*
			if !isTCPAlive(downstrService.Host) {
				clientErr = errors.New(RESPONSE_SERVICE_DOWN)
				errorlog.IncrementErrorCount(sqp, downstrService.QualifiedUrl, UPSTREAM_TCP_ERR, clientErr.Error())
				time.Sleep(time.Duration((*sqp).RetryGap) * time.Second) // wait on error
				continue
			}*/

		//fmt.Printf("->Forwarding to %s\n", downstrService.QualifiedUrl)

		body := ioutil.NopCloser(bytes.NewReader(reqParam.BodyBuff))
		downstrReq, _ := http.NewRequest(reqParam.Method, downstrService.QualifiedUrl+reqParam.RequestURI, body)
		downstrReq.Header = reqParam.Headers

		once.Do(func() {
			client.Timeout = time.Duration((*sqp).OutRequestTimeout) * time.Second
		})
		resp, err := client.Do(downstrReq)

		// handle response
		if resp == nil || err != nil {
			clientErr = err
			if clientErr != nil {
				if e, ok := clientErr.(net.Error); ok && e.Timeout() {
					clientErr = errors.New(RESPONSE_TIMED_OUT)
				} else {
					clientErr = errors.New(RESPONSE_NO_RESPONSE)
				}
			} else {
				clientErr = errors.New(RESPONSE_NO_RESPONSE)
			}
			go errorlog.IncrementErrorCount(sqp, downstrService.QualifiedUrl, DOWNSTREAM_HTTP_ERR, clientErr.Error())
			time.Sleep(time.Duration((*sqp).RetryGap) * time.Second) // wait on error
			continue
		} else {
			go errorlog.ResetErrorCount(sqp, downstrService.QualifiedUrl)
			clientErr = nil

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
	if clientErr != nil {
		return checkErrorAndRespond(clientErr, reqParam, sqp)
	}

	return model.ResponseParam{}, true, errors.New("send-fail")
}

// checkErrorAndRespond sets error and buffer flag based on buffer config and type of error from downstream node.
func checkErrorAndRespond(clientErr error, reqParam model.RequestParam, sqp *model.ServiceQProperties) (model.ResponseParam, bool, error) {

	if clientErr.Error() == RESPONSE_NO_RESPONSE || clientErr.Error() == RESPONSE_TIMED_OUT {
		if canBeBuffered(reqParam, sqp) {
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

	if (*sqp).EnableDeferredQ {

		reqFormats := (*sqp).DeferredQRequestFormats

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
