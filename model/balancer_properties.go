package model

import (
	"sync"
)

type ServiceQProperties struct {
	ListenerPort          string
	Proto                 string
	ServiceList           []Endpoint
	CustomRequestHeaders  []string
	CustomResponseHeaders []string
	MaxConcurrency        int64
	MaxRetries            int
	RetryGap              int
	IdleGap               int
	RequestErrorLog       map[string]int
	OutReqTimeout         int32
	REMutex               sync.Mutex
}
