package model

import (
	"bufio"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
)

type HTTPConnection struct {
	tcpConn *net.Conn
}

func (httpConn *HTTPConnection) Enclose(tcpConn *net.Conn) {

	httpConn.tcpConn = tcpConn
}

func (httpConn *HTTPConnection) ReadFrom() (*http.Request, error) {

	reader := bufio.NewReader(*httpConn.tcpConn)
	req, err := http.ReadRequest(reader)

	if err == nil {
		return req, nil
	}

	return nil, errors.New("read-fail")
}

func (httpConn *HTTPConnection) WriteTo(res *http.Response, customHeaders []string) error {

	writer := bufio.NewWriter(*httpConn.tcpConn)

	var responseBody []byte
	if res.Body != nil {
		responseBody, _ = ioutil.ReadAll(res.Body)
		res.Body.Close()
	}

	responseProtocol := res.Proto
	responseHeaders := ""
	responseStatus := res.Status

	// add original response headers
	if res.Header != nil {
		for k, v := range res.Header {
			responseHeaders += k + ": " + strings.Join(v, ",") + "\n"
		}
	}

	// add user custom headers
	if customHeaders != nil {
		for _, h := range customHeaders {
			responseHeaders += h + "\n"
		}
	}

	if responseHeaders != "" {
		responseHeaders = responseHeaders[:len(responseHeaders)-1]
		responseStatus = responseStatus + "\n"
	}

	clientResStr := responseProtocol + " " + responseStatus + responseHeaders + "\n\n" + string(responseBody)

	clientRes := []byte(clientResStr)

	_, err := writer.Write(clientRes) // tunneling onto tcp conn writer
	if err == nil {
		writer.Flush()
		return nil
	}

	return errors.New("write-fail")
}
