package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/gptankit/serviceq/errorlog"
	"github.com/gptankit/serviceq/model"
	"github.com/gptankit/serviceq/properties"
	"github.com/gptankit/serviceq/protocol/httpservice"
)

// main sets up serviceq properties, initializes work done and request buffers,
// and starts routines to accept new tcp connections and observe buffered requests
func main() {

	ctx := context.Background()
	stopCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	if sqp, err := properties.New(properties.GetFilePath()); err == nil {

		if ln, err := newListener(sqp); err == nil {
			defer closeListener(ln)

			cwork := make(chan int, sqp.MaxConcurrency+1)      // work done queue
			creq := make(chan interface{}, sqp.MaxConcurrency) // request queue

			// observe buffered requests
			go workBackground(stopCtx, creq, cwork, sqp)

			// accept new connections
			listenActive(stopCtx, ln, creq, cwork, sqp)
		} else {
			go errorlog.LogGenericError("Could not listen on :" + sqp.ListenerPort + " -- " + err.Error())
		}
	} else {
		go errorlog.LogGenericError("Could not read sq.properties, closing listener -- " + err.Error())
	}
}

// listenActive forwards new requests to the cluster
func listenActive(ctx context.Context, ln *net.Listener, creq chan interface{}, cwork chan int, sqp *model.ServiceQProperties) {

	shutListener := func(ln *net.Listener) {
		<-ctx.Done()
		closeListener(ln)
	}
	go shutListener(ln)

	for {
		if conn, err := (*ln).Accept(); err == nil {
			if len(cwork) < cap(cwork)-1 {
				switch sqp.Proto {
				case "http":
					if httpSrv := httpservice.New(sqp, httpservice.WithIncomingTCPConn(&conn)); httpSrv != nil {
						go httpSrv.ExecuteRealTime(ctx, creq, cwork)
					}
				default:
					conn.Close()
				}
			} else {
				if httpSrv := httpservice.New(sqp, httpservice.WithIncomingTCPConn(&conn)); httpSrv != nil {
					go httpSrv.Discard(ctx)
				}
			}
		} else {
			// handle permanent accept error including closed listener
			if ae, ok := err.(net.Error); ok && !ae.Temporary() {
				break
			}
		}
	}
}

// workBackground forwards buffered requests to the cluster
func workBackground(ctx context.Context, creq chan interface{}, cwork chan int, sqp *model.ServiceQProperties) {

	switch sqp.Proto {
	case "http":
		if httpSrv := httpservice.New(sqp); httpSrv != nil {
			go httpSrv.ExecuteBuffered(ctx, creq, cwork)
		}
	default:
		break
	}
}
