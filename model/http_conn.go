package model

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
)

// HTTPConnection is a http connection object that holds underlying
// tcp connection with reader and writer to that connection.
type HTTPConnection struct {
	tcpConn *net.Conn
	reader  *bufio.Reader
	writer  *bufio.Writer
}

// Enclose initializes new http reader/writer to underlying tcp connection.
func (httpConn *HTTPConnection) Enclose(tcpConn *net.Conn) {

	httpConn.tcpConn = tcpConn
	httpConn.reader = bufio.NewReader(*httpConn.tcpConn)
	httpConn.writer = bufio.NewWriter(*httpConn.tcpConn)
}

// ReadFrom reads http request from reader.
func (httpConn *HTTPConnection) ReadFrom() (*http.Request, error) {

	req, err := http.ReadRequest(httpConn.reader)

	if err == nil {
		return req, nil
	}

	fmt.Println("read-failed")
	fmt.Println(err.Error())

	return nil, errors.New("read-fail")
}

// WriteTo writes http response to writer (in http format).
func (httpConn *HTTPConnection) WriteTo(res ResponseParam, customHeaders []string) error {

	responseHeaders := ""

	// add original response headers
	if res.Headers != nil {
		for k, v := range res.Headers {
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
