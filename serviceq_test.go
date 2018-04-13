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

	sqp := model.ServiceQProperties{}
	sqp.ListenerPort = "5252"
	sqp.Proto = "http"
	sqp.ServiceList = []string{"http://example.org:2001", "http://example.org:3001", "http://example.org:4001", "http://example.org:5001"}
	sqp.MaxConcurrency = 8 // if changing, do check value of duplicateWork
	sqp.MaxRetries = 1     // we know it's down
	sqp.RetryGap = 1000    // ms
	sqp.IdleGap = 500      // ms
	sqp.RequestErrorLog = make(map[string]int, 2)
	sqp.OutReqTimeout = 500

	cw := make(chan int, sqp.MaxConcurrency)
	cc := make(chan *net.Conn, sqp.MaxConcurrency)
	cr := make(chan interface{}, sqp.MaxConcurrency)

	req, _ := http.NewRequest("GET", "http://example.org:1001", nil)

	cr <- req
	cw <- 1

	go orchestrate(cc, cr, cw, &sqp) // this will start executing req

	// increment/decrement buffer (+1/-1) in creq, cwork and give time to orchestrate

	duplicateWork := int(sqp.MaxConcurrency/2) + 1

	time.Sleep(1000 * time.Millisecond)
	// add req and work again
	for i := 0; i < duplicateWork; i++ {
		cr <- req
		cw <- 1
	}

	time.Sleep(1000 * time.Millisecond)
	// remove all work without removing req
	for i := 0; i < duplicateWork+1; i++ {
		<-cw
	}

	time.Sleep(1000 * time.Millisecond)
	// add half of works
	for i := 0; i < ((duplicateWork + 1) / 2); i++ {
		cw <- 1
	}

	time.Sleep(3000 * time.Millisecond)

	if len(cw) < ((duplicateWork+1)/2)-1 || len(cw) > ((duplicateWork+1)/2) {
		t.Errorf("Work not being orchestrated properly\n")
	}
}
