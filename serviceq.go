package main

import (
	"net"

	"github.com/gptankit/serviceq/errorlog"
	"github.com/gptankit/serviceq/model"
	"github.com/gptankit/serviceq/protocol/httpconn"
)

// main sets up serviceq properties, initializes work done and request buffers,
// and starts routines to accept new tcp connections and observe buffered requests.
func main() {

	if sqp, err := getProperties(getPropertyFilePath()); err == nil {

		if listener, err := getListener(sqp); err == nil {
			defer listener.Close()

			cwork := make(chan int, sqp.MaxConcurrency+1)      // work done queue
			creq := make(chan interface{}, sqp.MaxConcurrency) // request queue

			// observe buffered requests
			go workBackground(creq, cwork, sqp)

			// accept new connections
			listenActive(listener, creq, cwork, sqp)
		} else {
			go errorlog.LogGenericError("Could not listen on :" + sqp.ListenerPort + " -- " + err.Error())
		}
	} else {
		go errorlog.LogGenericError("Could not read sq.properties, closing listener -- " + err.Error())
	}
}

// listenActive forwards new requests to the cluster.
func listenActive(listener net.Listener, creq chan interface{}, cwork chan int, sqp *model.ServiceQProperties) {

	for {
		if conn, err := listener.Accept(); err == nil {
			if len(cwork) < cap(cwork)-1 {
				if sqp.Proto == "http" {
					httpConn := httpconn.New(&conn)
					go httpConn.ExecuteRealTime(creq, cwork, sqp)
				} else {
					conn.Close()
				}
			} else {
				httpConn := httpconn.New(&conn)
				go httpConn.Discard(sqp)
			}
		}
	}
}

// workBackground forwards buffered requests to the cluster.
func workBackground(creq chan interface{}, cwork chan int, sqp *model.ServiceQProperties) {

	switch sqp.Proto {
	case "http":
		httpConn := httpconn.NewNop()
		go httpConn.ExecuteBuffered(creq, cwork, sqp)
	default:
		break // don't do anything

	}
}
