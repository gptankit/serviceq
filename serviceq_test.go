package main

import (
	"testing"
	"time"

	"github.com/gptankit/serviceq/model"
)

func TestWorkAssigment(t *testing.T) {

	// assumption -- all services are down

	sqp := model.ServiceQProperties{
		ListenerPort: "5252",
		Proto:        "http",
		ServiceList: []model.Endpoint{
			{RawUrl: "http://example.org:2001", Scheme: "http", QualifiedUrl: "http://example.org:2001", Host: "example.org:2001"},
			{RawUrl: "http://example.org:3001", Scheme: "http", QualifiedUrl: "http://example.org:3001", Host: "example.org:3001"},
			{RawUrl: "http://example.org:4001", Scheme: "http", QualifiedUrl: "http://example.org:4001", Host: "example.org:4001"},
			{RawUrl: "http://example.org:5001", Scheme: "http", QualifiedUrl: "http://example.org:5001", Host: "example.org:5001"},
		},
		MaxConcurrency:    8, // if changing, do check value of duplicateWork
		EnableDeferredQ:   true,
		QRequestFormats:   []string{"ALL"},
		MaxRetries:        1,   // we know it's down
		RetryGap:          0,   // ms
		IdleGap:           500, // ms
		RequestErrorLog:   make(map[string]uint64, 2),
		OutRequestTimeout: 1,
	}

	cw := make(chan int, sqp.MaxConcurrency)
	cr := make(chan interface{}, sqp.MaxConcurrency)

	reqParam := model.RequestParam{
		Protocol:   "HTTP/1.1",
		Method:     "GET",
		RequestURI: "/getRefund",
		Headers: map[string][]string{
			"Content-Type": {"application/json"},
		},
		BodyBuff: nil,
	}
	cr <- reqParam
	cw <- 1

	go workBackground(cr, cw, &sqp) // this will start executing req

	// increment/decrement buffer (+1/-1) in creq, cwork and give time to orchestrate

	duplicateWork := int(sqp.MaxConcurrency/2) + 1

	time.Sleep(1000 * time.Millisecond)
	// add req and work again
	for i := 0; i < duplicateWork; i++ {
		cr <- reqParam
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
