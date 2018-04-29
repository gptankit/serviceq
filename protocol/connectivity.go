package protocol

import (
	"net"
	"time"
)

func isTCPAlive(service string) bool {

	dialTO := 5000
	conn, err := net.DialTimeout("tcp", service, time.Duration(dialTO) * time.Millisecond)
	if err == nil {
		conn.Close()
		return true
	}

	return false
}
