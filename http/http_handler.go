package http

import (
	"bytes"
	"errorlog"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"model"
	"net"
	"net/http"
	"netserve"
	"os"
	"time"
)

func HandleConnection(conn *net.Conn, creq chan interface{}, cwork chan int, sqprops *model.ServiceQProperties) {

	httpConn := model.HTTPConnection{}
	httpConn.Enclose(conn)
	request, err := httpConn.ReadFrom()
	var response *http.Response
	if err == nil {
		response, err = dialAndSendRequest(request, sqprops)
	}

	if err == nil {
		err = httpConn.WriteTo(response, (*sqprops).CustomResponseHeaders)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error on writing to client conn\n")
		}
	} else {
		// requeue on no response
		creq <- request
		cwork <- 1
		fmt.Printf("Request bufferred\n")
	}

	(*conn).Close()
	<-cwork

	return
}

func HandleBufferedReader(request *http.Request, creq chan interface{}, cwork chan int, sqprops *model.ServiceQProperties) {

	_, err := dialAndSendRequest(request, sqprops)
	if err == nil {
		// throw away response
	} else {
		// requeue on no response
		creq <- request
		cwork <- 1
		fmt.Printf("Request bufferred\n")
	}

	<-cwork
}

func dialAndSendRequest(request *http.Request, sqprops *model.ServiceQProperties) (*http.Response, error) {

	choice := -1
	noOfServices := len((*sqprops).ServiceList)
	method := request.Method
	requestURI := request.RequestURI

	// saving body
	var body io.ReadCloser
	if request.Body != nil {
		bodyBuff, _ := ioutil.ReadAll(request.Body)
		if len(bodyBuff) > 0 {
			body = ioutil.NopCloser(bytes.NewReader(bodyBuff))
		}
	}

	// saving headers, if available
	var headers map[string][]string
	if request.Header != nil {
		headers = make(map[string][]string, len(request.Header))
		for k, v := range request.Header {
			headers[k] = v
		}
	}

	for retry := 0; retry < (*sqprops).MaxRetries; retry++ {

		choice = netserve.ChooseServiceIndex(noOfServices, choice, retry)
		dstService := (*sqprops).ServiceList[choice]

		fmt.Printf("%s] Connecting to %s\n", time.Now().UTC().Format("2006-01-02 15:04:05"), dstService)
		// ping ip
		if !netserve.IsTCPAlive(dstService) {
			errorlog.IncrementErrorCount(sqprops, dstService)
			time.Sleep(time.Duration((*sqprops).RetryGap) * time.Millisecond) // wait on error
			continue
		}

		fmt.Printf("->Forwarding to %s\n", dstService)
		newRequest, _ := http.NewRequest(method, dstService+requestURI, body)
		newRequest.Header = headers
		// do http call
		client := &http.Client{Timeout: time.Duration((*sqprops).OutReqTimeout) * time.Millisecond}
		resp, err := client.Do(newRequest)

		// handle response
		if resp == nil || err != nil {
			go errorlog.IncrementErrorCount(sqprops, dstService)
			time.Sleep(time.Duration((*sqprops).RetryGap) * time.Millisecond) // wait on error
			continue
		} else {
			go errorlog.ResetErrorCount(sqprops, dstService)
			return resp, nil
		}
	}

	return nil, errors.New("send-fail")
}
