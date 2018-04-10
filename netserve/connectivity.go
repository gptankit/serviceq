package netserve

import (
	"fmt"
	"net"
	"net/url"
	"os"
)

var conn net.Conn
var uri *url.URL
var err error

func IsTCPAlive(service string) bool {

	uri, err = url.ParseRequestURI(service)
	if err != nil {
		fmt.Fprintf(os.Stderr, "->Not a valid url\n")
		return false
	}

	hostport := uri.Host
	conn, err = net.Dial("tcp", hostport)
	if err == nil {
		conn.Close()
		return true
	}

	fmt.Fprintf(os.Stderr, "->Service is down at %s\n", service)
	return false
}
