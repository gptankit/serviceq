package model

type TCPConnection interface {
	ReadFrom() (interface{}, error)
	WriteTo(interface{}, []string) error
}
