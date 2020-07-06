package main

import (
	"fmt"
	"github.com/gptankit/serviceq/model"
	"net"
	"os"
	"github.com/gptankit/serviceq/protocol"
	"time"
)

// main sets up serviceq properties, initializes work done and request buffers,
// and starts routines to accept new connections and observe buffered requests.
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
			fmt.Fprintf(os.Stderr, "Could not listen on :"+sqp.ListenerPort+" -- %s\n", err.Error())
		}
	} else {
		fmt.Fprintf(os.Stderr, "Could not read sq.properties, closing listener -- %s\n", err.Error())

	}
}

// listenActive forwards new requests to the cluster.
func listenActive(listener net.Listener, creq chan interface{}, cwork chan int, sqp *model.ServiceQProperties) {

	for {
		if conn, err := listener.Accept(); err == nil {
			if len(cwork) < cap(cwork)-1 {
				if (*sqp).Proto == "http" {
					go protocol.HandleHttpConnection(&conn, creq, cwork, sqp)
				} else {
					<-cwork
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
			if (*sqp).Proto == "http" {
				go protocol.HandleHttpBufferedReader((<-creq).(model.RequestParam), creq, cwork, sqp)
			}
		} else {
			time.Sleep(time.Duration((*sqp).IdleGap) * time.Millisecond) // wait for more work
		}
	}
}
