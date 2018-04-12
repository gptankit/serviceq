package main

import (
	"fmt"
	sqhttp "http"
	"model"
	"net"
	"net/http"
	"os"
	"profiling"
	"props"
	"time"
)

func main() {

	if listen, err := net.Listen("tcp", "localhost:8008"); err == nil {

		defer listen.Close()

		var sqprops model.ServiceQProperties
		sqwd := "/opt/serviceq"
		confFilePath := sqwd + "/config/sq.properties"

		if config, err := props.GetConfiguration(confFilePath); err == nil {

			// config.EnableProfilingFor should be "" in production, controlled in sq.properties
			profiling.Start(config.EnableProfilingFor)

			assignSQProps(&sqprops, config)

			if len(sqprops.ServiceList) > 0 {

				cwork := make(chan int, sqprops.MaxConcurrency)        // work done queue
				cconn := make(chan *net.Conn, sqprops.MaxConcurrency)  // connection queue
				creq := make(chan interface{}, sqprops.MaxConcurrency) // request queue

				// observe connections
				go orchestrate(cconn, creq, cwork, &sqprops)

				// handle connection events in a loop
				for {
					if conn, err := listen.Accept(); err == nil {
						if len(cwork) < cap(cwork) {
							cwork <- 1
							cconn <- (&conn)
						} else {
							conn.Close()
						}
					}
				}
			} else {
				fmt.Fprintf(os.Stderr, "No services listed, closing listener\n")
			}
		} else {
			fmt.Fprintf(os.Stderr, "Could not read sq.properties, closing listener -- %s\n", err.Error())
		}
	}
}

func orchestrate(cconn chan *net.Conn, creq chan interface{}, cwork chan int, sqprops *model.ServiceQProperties) {

	for {
		if len(cwork) > 0 && len(creq) > 0 { // handle bufferred requests
			if (*sqprops).Proto == "http" {
				go sqhttp.HandleBufferedReader((<-creq).(*http.Request), creq, cwork, sqprops)
			} else {
				// handle other protocols
			}
		} else if len(cwork) > 0 && len(cconn) > 0 { // handle active requests
			if (*sqprops).Proto == "http" {
				go sqhttp.HandleConnection(<-cconn, creq, cwork, sqprops)
			} else {
				// handle other protocols
			}
		} else {
			// wait for new work
			time.Sleep(time.Duration((*sqprops).IdleGap) * time.Millisecond)
		}
	}
}

func assignSQProps(sqprops *model.ServiceQProperties, config model.Config) {

	(*sqprops).Proto = config.Proto
	(*sqprops).ServiceList = config.Endpoints
	(*sqprops).CustomRequestHeaders = config.CustomRequestHeaders
	(*sqprops).CustomResponseHeaders = config.CustomResponseHeaders
	(*sqprops).MaxConcurrency = config.ConcurrencyPeak
	(*sqprops).MaxRetries = (len(sqprops.ServiceList) * 2) + 1 // atleast-once trial
	(*sqprops).RetryGap = 1000
	(*sqprops).IdleGap = 500
	(*sqprops).RequestErrorLog = make(map[string]int, len(config.Endpoints))
	(*sqprops).OutReqTimeout = config.OutReqTimeout
}
