package model

type Config struct {
	ListenerPort            string
	Proto                   string
	Endpoints               []Endpoint
	CustomRequestHeaders    []string
	CustomResponseHeaders   []string
	ConcurrencyPeak         int64
	EnableDeferredQ         bool
	DeferredQRequestFormats []string
	RetryGap                int
	OutRequestTimeout       int32
	SSLEnabled		bool
	SSLCertificateFile      string
	SSLPrivateKeyFile	string
	KeepAliveTimeout	int32
}
