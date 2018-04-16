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
	RESPONSE_TIMED_OUT    = "TIMED_OUT"
	RESPONSE_SERVICE_DOWN = "SERVICE_DOWN"
	RESPONSE_NO_RESPONSE  = "NO_RESPONSE"
)

func HandleConnection(conn *net.Conn, creq chan interface{}, cwork chan int, sqprops *model.ServiceQProperties) {

	var response *http.Response
	var reqParam model.RequestParam
	var toBuffer bool

	httpConn := model.HTTPConnection{}
	httpConn.Enclose(conn)
	request, err := httpConn.ReadFrom()

	if err == nil {
		reqParam = saveReqParam(request)
		response, toBuffer, err = dialAndSend(reqParam, sqprops)
	}

	if err == nil {
		err = httpConn.WriteTo(response, (*sqprops).CustomResponseHeaders)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error on writing to client conn\n")
		}
	}

	if toBuffer && canBeBuffered(reqParam, sqprops) {
		creq <- reqParam
		cwork <- 1
		fmt.Printf("Request bufferred\n")
	}

	(*conn).Close()
	<-cwork

	return
}

func HandleBufferedReader(reqParam model.RequestParam, creq chan interface{}, cwork chan int, sqprops *model.ServiceQProperties) {

	_, toBuffer, _ := dialAndSend(reqParam, sqprops)

	if toBuffer {
		creq <- reqParam
		cwork <- 1
		fmt.Printf("Request bufferred\n")
	}

	<-cwork

	return
}

func saveReqParam(request *http.Request) model.RequestParam {

	var reqParam model.RequestParam

	reqParam.Protocol = request.Proto
	reqParam.Method = request.Method
	reqParam.RequestURI = request.RequestURI

	if request.Body != nil {
		if bodyBuff, err := ioutil.ReadAll(request.Body); err == nil {
			reqParam.BodyBuff = bodyBuff
		}
	}

	if request.Header != nil {
		reqParam.Headers = make(map[string][]string, len(request.Header))
		for k, v := range request.Header {
			reqParam.Headers[k] = v
		}
	}

	return reqParam
}

func canBeBuffered(reqParam model.RequestParam, sqprops *model.ServiceQProperties) bool {

	if (*sqprops).EnableDeferredQ {

		reqFormats := (*sqprops).DeferredQRequestFormats

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

func dialAndSend(reqParam model.RequestParam, sqprops *model.ServiceQProperties) (*http.Response, bool, error) {

	choice := -1
	noOfServices := len((*sqprops).ServiceList)
	var clientErr error

	for retry := 0; retry < (*sqprops).MaxRetries; retry++ {

		choice = netserve.ChooseServiceIndex(noOfServices, choice, retry)
		dstService := (*sqprops).ServiceList[choice]

		fmt.Printf("%s] Connecting to %s\n", time.Now().UTC().Format("2006-01-02 15:04:05"), dstService.Host)
		// ping ip
		if !netserve.IsTCPAlive(dstService.Host) {
			errorlog.IncrementErrorCount(sqprops, dstService.QualifiedUrl)
			time.Sleep(time.Duration((*sqprops).RetryGap) * time.Millisecond) // wait on error
			clientErr = errors.New(RESPONSE_SERVICE_DOWN)
			continue
		}

		fmt.Printf("->Forwarding to %s\n", dstService.QualifiedUrl)

		body := ioutil.NopCloser(bytes.NewReader(reqParam.BodyBuff))
		newRequest, _ := http.NewRequest(reqParam.Method, dstService.QualifiedUrl+reqParam.RequestURI, body)
		newRequest.Header = reqParam.Headers

		// do http call
		client := &http.Client{Timeout: time.Duration((*sqprops).OutRequestTimeout) * time.Millisecond}
		resp, err := client.Do(newRequest)

		// handle response
		if resp == nil || err != nil {
			go errorlog.IncrementErrorCount(sqprops, dstService.QualifiedUrl)
			time.Sleep(time.Duration((*sqprops).RetryGap) * time.Millisecond) // wait on error
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
			break
		} else {
			go errorlog.ResetErrorCount(sqprops, dstService.QualifiedUrl)
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
