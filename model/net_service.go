package model

import "context"

type NetService interface {
	Read() (interface{}, error)
	Write(interface{}) error
	ExecuteRealTime(context.Context, chan interface{}, chan int)
	ExecuteBuffered(context.Context, chan interface{}, chan int)
	Discard(context.Context)
}
