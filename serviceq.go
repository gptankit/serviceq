package main

import (
	"net"

	"github.com/gptankit/serviceq/errorlog"
	"github.com/gptankit/serviceq/model"
	"github.com/gptankit/serviceq/properties"
	"github.com/gptankit/serviceq/protocol/httpservice"
)

// main sets up serviceq properties, initializes work done and request buffers,
// and starts routines to accept new tcp connections and observe buffered requests
func main() {

	if sqp, err := properties.New(properties.GetFilePath()); err == nil {

		if listener, err := newListener(sqp); err == nil {
			defer (*listener).Close()

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

// listenActive forwards new requests to the cluster
func listenActive(listener *net.Listener, creq chan interface{}, cwork chan int, sqp *model.ServiceQProperties) {

	for {
		if conn, err := (*listener).Accept(); err == nil {
			if len(cwork) < cap(cwork)-1 {
				switch sqp.Proto {
				case "http":
					if httpSrv := httpservice.New(sqp, httpservice.WithIncomingTCPConn(&conn)); httpSrv != nil {
						go httpSrv.ExecuteRealTime(creq, cwork)
					}
				default:
					conn.Close()
				}
			} else {
				if httpSrv := httpservice.New(sqp, httpservice.WithIncomingTCPConn(&conn)); httpSrv != nil {
					go httpSrv.Discard()
				}
			}
		}
	}
}

// workBackground forwards buffered requests to the cluster
func workBackground(creq chan interface{}, cwork chan int, sqp *model.ServiceQProperties) {

	switch sqp.Proto {
	case "http":
		if httpSrv := httpservice.New(sqp); httpSrv != nil {
			go httpSrv.ExecuteBuffered(creq, cwork)
		}
	default:
		break
	}
}
