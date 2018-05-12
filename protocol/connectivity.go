package protocol

import (
	"net"
	"time"
)

func setTCPDeadline(conn *net.Conn, keepAliveTimeout int32) {

	if keepAliveTimeout >= 0 {
		(*conn).SetDeadline(time.Now().Add(time.Millisecond * time.Duration(keepAliveTimeout)))
	}
}

func isTCPAlive(service string) bool {

	dialTO := 5000
	conn, err := net.DialTimeout("tcp", service, time.Duration(dialTO) * time.Millisecond)
	if err == nil {
		conn.Close()
		return true
	}

	return false
}
