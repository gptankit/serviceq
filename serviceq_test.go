package main

import (
	"model"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestOrchestrationStates(t *testing.T) {

	// assumption -- all services are down

	// assign initial values for cconn, (use bufio.Writer on conn), creq, cwork, sqprops

	sqp := model.ServiceQProperties{}
	sqp.Proto = "http"
	sqp.ServiceList = []string{"http://example.org:2001", "http://example.org:3001", "http://example.org:4001", "http://example.org:5001"}
	sqp.MaxConcurrency = 64 // if changing, do check value of duplicateWork
	sqp.MaxRetries = 1      // we know it's down
	sqp.RetryGap = 1000     // ms
	sqp.IdleGap = 500       // ms
	sqp.RequestErrorLog = make(map[string]int, 2)
	sqp.OutReqTimeout = 500

	cw := make(chan int, sqp.MaxConcurrency)
	cc := make(chan *net.Conn, sqp.MaxConcurrency)
	cr := make(chan interface{}, sqp.MaxConcurrency)

	req, _ := http.NewRequest("GET", "http://example.org:1001", nil)

	cr <- req
	cw <- 1

	// call go orchestrate and note counts in cconn, creq
	go orchestrate(cc, cr, cw, &sqp) // this will start executing req

	// increment/decrement buffer (+1/-1) in creq, cwork and give time to orchestrate

	duplicateWork := int(sqp.MaxConcurrency/2) + 1

	time.Sleep(2 * time.Second)
	for i := 0; i < duplicateWork; i++ {
		// add req and work again
		cr <- req
		cw <- 1
	}

	time.Sleep(2 * time.Second)
	for i := 0; i < duplicateWork+1; i++ {
		// remove all work without removing req
		<-cw
	}

	time.Sleep(5 * time.Second)
	for i := 0; i < ((duplicateWork + 1) / 2); i++ {
		// add half of works
		cw <- 1
	}

	time.Sleep(5 * time.Second)

	if len(cw) < ((duplicateWork+1)/2)-1 || len(cw) > ((duplicateWork+1)/2) {
		t.Errorf("Work not being orchestrated properly")
	}
}
