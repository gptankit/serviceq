package model

import (
	"sync"
)

type ServiceQProperties struct {
	ListenerPort            string
	Proto                   string
	ServiceList             []Endpoint
	CustomRequestHeaders    []string
	CustomResponseHeaders   []string
	MaxConcurrency          int64
	EnableDeferredQ         bool
	DeferredQRequestFormats []string
	MaxRetries              int
	RetryGap                int
	IdleGap                 int
	RequestErrorLog         map[string]uint64
	OutRequestTimeout       int32
	SSLEnabled              bool
	SSLCertificateFile      string
	SSLPrivateKeyFile       string
	SSLAutoEnabled          bool
	SSLAutoCertificateDir   string
	SSLAutoEmail            string
	SSLAutoDomains          string
	SSLAutoRenewBefore      int32
	KeepAliveTimeout        int32
	KeepAliveServe          bool
	REMutex                 sync.Mutex
}
