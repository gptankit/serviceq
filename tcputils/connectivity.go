package tcputils

import (
	"net"
	"time"
)

// SetTCPDeadline sets keep-alive timeout to tcp connection
func SetTCPDeadline(conn *net.Conn, keepAliveTimeout int32) {

	if keepAliveTimeout >= 0 {
		(*conn).SetDeadline(time.Now().Add(time.Second * time.Duration(keepAliveTimeout)))
	}
}

// IsTCPAlive is a ping service to determine tcp connection state
func IsTCPAlive(service string) bool {

	dialTO := 5000
	conn, err := net.DialTimeout("tcp", service, time.Duration(dialTO)*time.Millisecond)
	if err == nil {
		conn.Close()
		return true
	}

	return false
}
