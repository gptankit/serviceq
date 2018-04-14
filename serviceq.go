package main

import (
	"fmt"
	sqhttp "http"
	"model"
	"net"
	"os"
	_ "profiling"
	"props"
	"time"
)

func main() {

	var sqprops model.ServiceQProperties
	confFile := props.GetConfFileLocation()

	if config, err := props.GetConfiguration(confFile); err == nil {
		assignProperties(&sqprops, config)

		if listener, err := net.Listen("tcp", "localhost:"+sqprops.ListenerPort); err == nil {
			defer listener.Close()

			// uncomment if profiling is needed in dev env, controlled in sq.properties
			// profiling.Start(config.EnableProfilingFor)

			if len(sqprops.ServiceList) > 0 {

				cwork := make(chan int, sqprops.MaxConcurrency)        // work done queue
				creq := make(chan interface{}, sqprops.MaxConcurrency) // request queue

				// observe bufferred requests
				go workBackground(creq, cwork, &sqprops)

				// accept new connections
				listenActive(listener, creq, cwork, &sqprops)
			} else {
				fmt.Fprintf(os.Stderr, "No services listed, closing listener\n")
			}
		} else {
			fmt.Fprintf(os.Stderr, "Could not listen to localhost:"+sqprops.ListenerPort+" -- %s\n", err.Error())
		}
	} else {
		fmt.Fprintf(os.Stderr, "Could not read sq.properties, closing listener -- %s\n", err.Error())
	}
}

func listenActive(listener net.Listener, creq chan interface{}, cwork chan int, sqprops *model.ServiceQProperties) {

	for {
		if conn, err := listener.Accept(); err == nil {
			if len(cwork) < cap(cwork) {
				cwork <- 1
				if (*sqprops).Proto == "http" {
					go sqhttp.HandleConnection(&conn, creq, cwork, sqprops)
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

func workBackground(creq chan interface{}, cwork chan int, sqprops *model.ServiceQProperties) {

	for {
		if len(cwork) > 0 && len(creq) > 0 {
			if (*sqprops).Proto == "http" {
				go sqhttp.HandleBufferedReader((<-creq).(model.RequestParam), creq, cwork, sqprops)
			}
		} else {
			time.Sleep(time.Duration((*sqprops).IdleGap) * time.Millisecond) // wait for more work
		}
	}
}

func assignProperties(sqprops *model.ServiceQProperties, config model.Config) {

	(*sqprops).ListenerPort = config.ListenerPort
	(*sqprops).Proto = config.Proto
	(*sqprops).ServiceList = config.Endpoints
	(*sqprops).CustomRequestHeaders = config.CustomRequestHeaders
	(*sqprops).CustomResponseHeaders = config.CustomResponseHeaders
	(*sqprops).MaxConcurrency = config.ConcurrencyPeak
	(*sqprops).MaxRetries = (len(sqprops.ServiceList) * 2) + 1 // atleast-once trial
	(*sqprops).RetryGap = config.RetryGap
	(*sqprops).IdleGap = 500
	(*sqprops).RequestErrorLog = make(map[string]int, len(config.Endpoints))
	(*sqprops).OutReqTimeout = config.OutReqTimeout
}
