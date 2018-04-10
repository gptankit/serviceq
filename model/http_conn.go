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

func (httpConn *HTTPConnection) WriteTo(resp *http.Response, customHeaders []string) error {

	writer := bufio.NewWriter(*httpConn.tcpConn)

	defer resp.Body.Close()
	responseBody, _ := ioutil.ReadAll(resp.Body)
	responseProtocol := resp.Proto
	responseHeaders := ""
	responseStatus := resp.Status
	// add original response headers
	if resp.Header != nil {
		for k, v := range resp.Header {
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
	responseStr := responseProtocol + " " + responseStatus + responseHeaders + "\n\n" + string(responseBody)
	response := []byte(responseStr)

	_, err := writer.Write(response) // tunneling onto tcp conn writer
	if err == nil {
		writer.Flush()
		return nil
	}

	return errors.New("write-fail")
}
