package model

type Config struct {
	Proto                 string
	Endpoints             []string
	CustomRequestHeaders  []string
	CustomResponseHeaders []string
	ConcurrencyPeak       int64
	OutReqTimeout         int32
	EnableProfilingFor    string
}
