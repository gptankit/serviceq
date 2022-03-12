package model

type NetService interface {
	Read() (interface{}, error)
	Write(interface{}) error
	ExecuteRealTime(creq chan interface{}, cwork chan int)
	ExecuteBuffered(creq chan interface{}, cwork chan int)
	Discard()
}
