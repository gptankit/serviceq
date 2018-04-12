package netserve

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"time"
)

var conn net.Conn
var uri *url.URL
var err error

func IsTCPAlive(service string) bool {

	dialTimeout := 1000
	uri, err = url.ParseRequestURI(service)
	if err != nil {
		fmt.Fprintf(os.Stderr, "->Not a valid url\n")
		return false
	}

	hostport := uri.Host
	conn, err = net.DialTimeout("tcp", hostport, time.Duration(dialTimeout)*time.Millisecond)
	if err == nil {
		conn.Close()
		return true
	}

	fmt.Fprintf(os.Stderr, "->Service is down at %s\n", service)
	return false
}
