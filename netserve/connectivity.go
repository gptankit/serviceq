package netserve

import (
	"fmt"
	"net"
	"os"
	"time"
)

func IsTCPAlive(service string) bool {

	dialTimeout := 1000
	conn, err := net.DialTimeout("tcp", service, time.Duration(dialTimeout)*time.Millisecond)
	if err == nil {
		conn.Close()
		return true
	}

	fmt.Fprintf(os.Stderr, "->Service is down at %s\n", service)
	return false
}
