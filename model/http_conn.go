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
	reader *bufio.Reader
	writer *bufio.Writer
}

func (httpConn *HTTPConnection) Enclose(tcpConn *net.Conn) {

	httpConn.tcpConn = tcpConn
	httpConn.reader = bufio.NewReader(*httpConn.tcpConn)
	httpConn.writer = bufio.NewWriter(*httpConn.tcpConn)
}

func (httpConn *HTTPConnection) ReadFrom() (*http.Request, error) {

	req, err := http.ReadRequest(httpConn.reader)

	if err == nil {
		return req, nil
	}

	return nil, errors.New("read-fail")
}

func (httpConn *HTTPConnection) WriteTo(res *http.Response, customHeaders []string) error {

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

	_, err := httpConn.writer.Write(clientRes) // tunneling onto tcp conn writer
	if err == nil {
		httpConn.writer.Flush()
		return nil
	}

	return errors.New("write-fail")
}
