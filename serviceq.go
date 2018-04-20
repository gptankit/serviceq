package main

import (
	"fmt"
	"model"
	"net"
	"os"
	"protocol"
	"time"
)

func main() {

	if config, err := getConfiguration(getConfFileLocation()); err == nil {
		sqp := assignProperties(config)

		if listener, err := net.Listen("tcp", "localhost:"+sqp.ListenerPort); err == nil {
			defer listener.Close()

			if len(sqp.ServiceList) > 0 {

				cwork := make(chan int, sqp.MaxConcurrency+1)      // work done queue
				creq := make(chan interface{}, sqp.MaxConcurrency) // request queue

				// observe bufferred requests
				go workBackground(creq, cwork, &sqp)

				// accept new connections
				listenActive(listener, creq, cwork, &sqp)
			} else {
				fmt.Fprintf(os.Stderr, "No services listed, closing listener\n")
			}
		} else {
			fmt.Fprintf(os.Stderr, "Could not listen to localhost:"+sqp.ListenerPort+" -- %s\n", err.Error())
		}
	} else {
		fmt.Fprintf(os.Stderr, "Could not read sq.properties, closing listener -- %s\n", err.Error())
	}
}

func listenActive(listener net.Listener, creq chan interface{}, cwork chan int, sqp *model.ServiceQProperties) {

	for {
		if conn, err := listener.Accept(); err == nil {
			if len(cwork) < cap(cwork)-1 {
				cwork <- 1
				if (*sqp).Proto == "http" {
					go protocol.HandleHttpConnection(&conn, creq, cwork, sqp)
				} else {
					<-cwork
					conn.Close()
				}
			} else {
				protocol.DiscardHttpConnection(&conn, sqp)
			}
		}
	}
}

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

func assignProperties(cfg model.Config) model.ServiceQProperties {

	return model.ServiceQProperties{
		ListenerPort:            cfg.ListenerPort,
		Proto:                   cfg.Proto,
		ServiceList:             cfg.Endpoints,
		CustomRequestHeaders:    cfg.CustomRequestHeaders,
		CustomResponseHeaders:   cfg.CustomResponseHeaders,
		MaxConcurrency:          cfg.ConcurrencyPeak,
		EnableDeferredQ:         cfg.EnableDeferredQ,
		DeferredQRequestFormats: cfg.DeferredQRequestFormats,
		MaxRetries:              (len(cfg.Endpoints) * 2) + 1,
		RetryGap:                cfg.RetryGap,
		IdleGap:                 500,
		RequestErrorLog:         make(map[string]int, len(cfg.Endpoints)),
		OutRequestTimeout:       cfg.OutRequestTimeout,
	}
}
