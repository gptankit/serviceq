package main

import (
	"github.com/gptankit/serviceq/errorlog"
	"github.com/gptankit/serviceq/model"
	"github.com/gptankit/serviceq/protocol"
	"net"
	"time"
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
			go workBackground(creq, cwork, &sqp)

			// accept new connections
			listenActive(listener, creq, cwork, &sqp)
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
					go protocol.HandleHttpConnection(&conn, creq, cwork, sqp)
				} else {
					conn.Close()
				}
			} else {
				go protocol.DiscardHttpConnection(&conn, sqp)
			}
		}
	}
}

// workBackground forwards buffered requests to the cluster.
func workBackground(creq chan interface{}, cwork chan int, sqp *model.ServiceQProperties) {

	for {
		if len(cwork) > 0 && len(creq) > 0 {
			if sqp.Proto == "http" {
				go protocol.HandleHttpBufferedReader((<-creq).(model.RequestParam), creq, cwork, sqp)
			}
		} else {
			time.Sleep(time.Duration(sqp.IdleGap) * time.Millisecond) // wait for more work
		}
	}
}
