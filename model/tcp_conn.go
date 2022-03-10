package model

type TCPConnection interface {
	Read() (interface{}, error)
	Write(interface{}, []string) error
	ExecuteRealTime(creq chan interface{}, cwork chan int, sqp *ServiceQProperties)
	ExecuteBuffered(creq chan interface{}, cwork chan int, sqp *ServiceQProperties)
	Discard(sqp *ServiceQProperties)
}
