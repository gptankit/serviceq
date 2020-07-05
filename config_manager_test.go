package main

import (
	"github.com/gptankit/serviceq/model"
	"testing"
)

type Properties struct {
	c model.ServiceQProperties
	e error
}

var props Properties

func TestReadConfiguration(t *testing.T) {

	cfPath := "sq.properties"
	sqp, err := getProperties(cfPath)

	props = Properties{c: sqp, e: err}
	if err != nil {
		t.Error(err.Error())
	}
}

func TestMandatoryProperties(t *testing.T) {

	if props.e == nil {
		if props.c.ListenerPort == "" {
			t.Error("LISTENER_PORT missing in sq.properties\n")
		}
		if props.c.Proto == "" {
			t.Error("PROTO missing in sq.properties\n")
		}
		if len(props.c.ServiceList) == 0 {
			t.Error("ENDPOINTS missing in sq.properties\n")
		}
		if props.c.MaxConcurrency == 0 {
			t.Error("CONCURRENCY_PEAK missing in sq.properties\n")
		}
	}
}
