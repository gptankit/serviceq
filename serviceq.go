package main

import (
	"fmt"
	sqhttp "http"
	"model"
	"net"
	"os"
	"props"
	"time"
)

func main() {

	if config, err := props.GetConfiguration(props.GetConfFileLocation()); err == nil {
		sqp := assignProperties(config)

		if listener, err := net.Listen("tcp", "localhost:"+sqp.ListenerPort); err == nil {
			defer listener.Close()

			if len(sqp.ServiceList) > 0 {

				cwork := make(chan int, sqp.MaxConcurrency)        // work done queue
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
			if len(cwork) < cap(cwork) {
				cwork <- 1
				if (*sqp).Proto == "http" {
					go sqhttp.HandleConnection(&conn, creq, cwork, sqp)
				} else {
					<-cwork
					conn.Close()
				}
			} else {
				conn.Close() // refuse connection
			}
		}
	}
}

func workBackground(creq chan interface{}, cwork chan int, sqp *model.ServiceQProperties) {

	for {
		if len(cwork) > 0 && len(creq) > 0 {
			if (*sqp).Proto == "http" {
				go sqhttp.HandleBufferedReader((<-creq).(model.RequestParam), creq, cwork, sqp)
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
