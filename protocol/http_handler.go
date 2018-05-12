package protocol

import (
	"algorithm"
	"bytes"
	"errorlog"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"model"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var client *http.Client

const (

	SERVICEQ_NO_ERR = 600
	SERVICEQ_FLOODED_ERR = 601
	UPSTREAM_NO_ERR = 700
	UPSTREAM_TCP_ERR = 701
	UPSTREAM_HTTP_ERR = 702

	RESPONSE_FLOODED      = "SERVICEQ_FLOODED"
	RESPONSE_TIMED_OUT    = "UPSTREAM_TIMED_OUT"
	RESPONSE_SERVICE_DOWN = "UPSTREAM_DOWN"
	RESPONSE_NO_RESPONSE  = "UPSTREAM_NO_RESPONSE"
)

func HandleHttpConnection(conn *net.Conn, creq chan interface{}, cwork chan int, sqp *model.ServiceQProperties) {

	httpConn := model.HTTPConnection{}
	setTCPDeadline(conn, (*sqp).KeepAliveTimeout)
	httpConn.Enclose(conn)

	for {

		var res *http.Response
		var reqParam model.RequestParam
		var toBuffer bool

		// read from and write to conn
		req, err := httpConn.ReadFrom()

		if err == nil {
			// add work
			cwork <- 1

			reqParam = saveReqParam(req)
			res, toBuffer, err = dialAndSend(reqParam, sqp)

			if err == nil {
				err = httpConn.WriteTo(res, (*sqp).CustomResponseHeaders)
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

func DiscardHttpConnection(conn *net.Conn, sqp *model.ServiceQProperties) {

	var res *http.Response
	httpConn := model.HTTPConnection{}
	httpConn.Enclose(conn)
	req, err := httpConn.ReadFrom()

	if err == nil {
		res = getCustomResponse(req.Proto, http.StatusTooManyRequests, "Request Discarded")
		clientErr := errors.New(RESPONSE_FLOODED)
		errorlog.IncrementErrorCount(sqp, "SQ_PROXY", SERVICEQ_FLOODED_ERR, clientErr.Error())
		err = httpConn.WriteTo(res, (*sqp).CustomResponseHeaders)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error on writing to client conn\n")
		}
	}
	forceCloseConn(conn)

	return
}

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

func dialAndSend(reqParam model.RequestParam, sqp *model.ServiceQProperties) (*http.Response, bool, error) {

	choice := -1
	var clientErr error

	for retry := 0; retry < (*sqp).MaxRetries; retry++ {

		choice = algorithm.ChooseServiceIndex(sqp, choice, retry)
		upstrService := (*sqp).ServiceList[choice]

		//fmt.Printf("%s] Connecting to %s\n", time.Now().UTC().Format("2006-01-02 15:04:05"), upstrService.Host)
		// ping ip -- response/error flow below will take care of tcp connect

		//if !isTCPAlive(upstrService.Host) {
		//	clientErr = errors.New(RESPONSE_SERVICE_DOWN)
		//	errorlog.IncrementErrorCount(sqp, upstrService.QualifiedUrl, UPSTREAM_TCP_ERR, clientErr.Error())
		//	time.Sleep(time.Duration((*sqp).RetryGap) * time.Millisecond) // wait on error
		//	continue
		//}

		//fmt.Printf("->Forwarding to %s\n", upstrService.QualifiedUrl)

		body := ioutil.NopCloser(bytes.NewReader(reqParam.BodyBuff))
		upstrReq, _ := http.NewRequest(reqParam.Method, upstrService.QualifiedUrl+reqParam.RequestURI, body)
		upstrReq.Header = reqParam.Headers

		// do http call
		if client == nil {
			client = &http.Client{Timeout: time.Duration((*sqp).OutRequestTimeout) * time.Millisecond}
		}

		resp, err := client.Do(upstrReq)

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
			go errorlog.IncrementErrorCount(sqp, upstrService.QualifiedUrl, UPSTREAM_HTTP_ERR, clientErr.Error())
			time.Sleep(time.Duration((*sqp).RetryGap) * time.Millisecond) // wait on error
			continue
		} else {
			go errorlog.ResetErrorCount(sqp, upstrService.QualifiedUrl)
			clientErr = nil
			return resp, false, nil
		}
	}

	// error based response
	if clientErr != nil {
		return checkErrorAndRespond(clientErr, reqParam, sqp)
	}

	return nil, true, errors.New("send-fail")
}

func checkErrorAndRespond(clientErr error, reqParam model.RequestParam, sqp *model.ServiceQProperties) (*http.Response, bool, error) {

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

func getCustomResponse(protocol string, statusCode int, resMsg string) *http.Response {

	var body io.ReadCloser
	var json string
	if resMsg != "" {
		json = `{"sq_msg":"` + resMsg +`"}`
		body = ioutil.NopCloser(bytes.NewReader([]byte(json)))
	}
	jsonLen := strconv.Itoa(len(json))

	return &http.Response{
		Proto:      protocol,
		Status:     strconv.Itoa(statusCode) + " " + http.StatusText(statusCode),
		StatusCode: statusCode, Header: http.Header{"Content-Type": []string{"application/json"}, "Content-Length": []string{jsonLen},},
		Body :	    body,
	}
}

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

func optCloseConn(conn *net.Conn, reqParam model.RequestParam, keepAliveServe bool) bool {

	if reqParam.Protocol == "HTTP/1.0" || reqParam.Protocol == "HTTP/1.1" { // Connection and keep-alive are ignored for http/2
		if v, ok := reqParam.Headers["Connection"]; ok {
			if v[0] == "keep-alive" && keepAliveServe {
			        return false// do not close conn
			} else if v[0] == "close" || !keepAliveServe { // close conn if Connection: close or keep-alive is not part of response
				return forceCloseConn(conn)
			}
		} else if reqParam.Protocol == "HTTP/1.1" && keepAliveServe {
			return false// do not close conn if Connection header not found for protocol http/1.1 -- follow default behaviour
		} else {
			return forceCloseConn(conn)
		}
	}

	return false
}

func forceCloseConn(conn *net.Conn) bool {

	(*conn).Close()

	return true
}
