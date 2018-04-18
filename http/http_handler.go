package http

import (
	"bytes"
	"errorlog"
	"errors"
	"fmt"
	"io/ioutil"
	"model"
	"net"
	"net/http"
	"netserve"
	"os"
	"strconv"
	"strings"
	"time"
)

const (

	UPSTREAM_NO_ERR = 0
	UPSTREAM_TCP_ERR = 1
	UPSTREAM_HTTP_ERR = 2

	RESPONSE_TIMED_OUT    = "TIMED_OUT"
	RESPONSE_SERVICE_DOWN = "SERVICE_DOWN"
	RESPONSE_NO_RESPONSE  = "NO_RESPONSE"
)

func HandleConnection(conn *net.Conn, creq chan interface{}, cwork chan int, sqp *model.ServiceQProperties) {

	var res *http.Response
	var reqParam model.RequestParam
	var toBuffer bool

	httpConn := model.HTTPConnection{}
	httpConn.Enclose(conn)
	req, err := httpConn.ReadFrom()

	if err == nil {
		reqParam = saveReqParam(req)
		res, toBuffer, err = dialAndSend(reqParam, sqp)
	}

	if err == nil {
		err = httpConn.WriteTo(res, (*sqp).CustomResponseHeaders)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error on writing to client conn\n")
		}
	}

	if toBuffer && canBeBuffered(reqParam, sqp) {
		creq <- reqParam
		cwork <- 1
		fmt.Printf("Request bufferred\n")
	}

	(*conn).Close()
	<-cwork

	return
}

func HandleBufferedReader(reqParam model.RequestParam, creq chan interface{}, cwork chan int, sqp *model.ServiceQProperties) {

	_, toBuffer, _ := dialAndSend(reqParam, sqp)

	if toBuffer {
		creq <- reqParam
		cwork <- 1
		fmt.Printf("Request bufferred\n")
	}

	<-cwork

	return
}

func saveReqParam(req *http.Request) model.RequestParam {

	var reqParam model.RequestParam

	reqParam.Protocol = req.Proto
	reqParam.Method = req.Method
	reqParam.RequestURI = req.RequestURI

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

func dialAndSend(reqParam model.RequestParam, sqp *model.ServiceQProperties) (*http.Response, bool, error) {

	choice := -1
	noOfServices := len((*sqp).ServiceList)
	var clientErr error

	for retry := 0; retry < (*sqp).MaxRetries; retry++ {

		choice = netserve.ChooseServiceIndex(noOfServices, choice, retry)
		upstrService := (*sqp).ServiceList[choice]

		fmt.Printf("%s] Connecting to %s\n", time.Now().UTC().Format("2006-01-02 15:04:05"), upstrService.Host)
		// ping ip
		if !netserve.IsTCPAlive(upstrService.Host) {
			clientErr = errors.New(RESPONSE_SERVICE_DOWN)
			errorlog.IncrementErrorCount(sqp, upstrService.QualifiedUrl, UPSTREAM_TCP_ERR, clientErr.Error())
			time.Sleep(time.Duration((*sqp).RetryGap) * time.Millisecond) // wait on error
			continue
		}

		fmt.Printf("->Forwarding to %s\n", upstrService.QualifiedUrl)

		body := ioutil.NopCloser(bytes.NewReader(reqParam.BodyBuff))
		upstrReq, _ := http.NewRequest(reqParam.Method, upstrService.QualifiedUrl+reqParam.RequestURI, body)
		upstrReq.Header = reqParam.Headers

		// do http call
		client := &http.Client{Timeout: time.Duration((*sqp).OutRequestTimeout) * time.Millisecond}
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
			break
		} else {
			go errorlog.ResetErrorCount(sqp, upstrService.QualifiedUrl)
			clientErr = nil
			return resp, false, nil
		}
	}

	// error based response
	if clientErr != nil {
		return checkErrorAndRespond(clientErr, reqParam.Protocol)
	}

	return nil, true, errors.New("send-fail")
}

func checkErrorAndRespond(clientErr error, protocol string) (*http.Response, bool, error) {

	if clientErr.Error() == RESPONSE_TIMED_OUT {
		return getCustomResponse(protocol, http.StatusGatewayTimeout), false, nil
	} else if clientErr.Error() == RESPONSE_SERVICE_DOWN {
		return getCustomResponse(protocol, http.StatusServiceUnavailable), true, nil
	} else if clientErr.Error() == RESPONSE_NO_RESPONSE {
		return getCustomResponse(protocol, http.StatusBadRequest), false, nil
	} else {
		return getCustomResponse(protocol, http.StatusBadGateway), false, nil
	}
}

func getCustomResponse(protocol string, statusCode int) *http.Response {

	return &http.Response{
		Proto:      protocol,
		Status:     strconv.Itoa(statusCode) + " " + http.StatusText(statusCode),
		StatusCode: statusCode, Header: http.Header{"Content-Type": []string{"application/json"}},
	}
}
