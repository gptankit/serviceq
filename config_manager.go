package main

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/gptankit/serviceq/model"
)

const (
	SQP_K_LISTENER_PORT            = "LISTENER_PORT"
	SQP_K_PROTOCOL                 = "PROTO"
	SQP_K_ENDPOINTS                = "ENDPOINTS"
	SQP_K_REQUEST_HEADERS          = "CUSTOM_REQUEST_HEADERS"
	SQP_K_RESPONSE_HEADERS         = "CUSTOM_RESPONSE_HEADERS"
	SQP_K_MAX_CONCURRENT_CONNS     = "CONCURRENCY_PEAK"
	SQP_K_ENABLE_UPFRONT_Q         = "ENABLE_UPFRONT_Q"
	SQP_K_ENABLE_DEFERRED_Q        = "ENABLE_DEFERRED_Q"
	SQP_K_Q_REQUEST_FORMATS        = "Q_REQUEST_FORMATS"
	SQP_K_RETRY_GAP                = "RETRY_GAP"
	SQP_K_OUT_REQUEST_TIMEOUT      = "OUTGOING_REQUEST_TIMEOUT"
	SQP_K_SSL_ENABLED              = "SSL_ENABLE"
	SQP_K_SSL_CERTIFICATE_FILE     = "SSL_CERTIFICATE_FILE"
	SQP_K_SSL_PRIVATE_KEY_FILE     = "SSL_PRIVATE_KEY_FILE"
	SQP_K_SSL_AUTO_ENABLED         = "SSL_AUTO_ENABLE"
	SQP_K_SSL_AUTO_CERTIFICATE_DIR = "SSL_AUTO_CERTIFICATE_DIR"
	SQP_K_SSL_AUTO_EMAIL           = "SSL_AUTO_EMAIL"
	SQP_K_SSL_AUTO_DOMAINS         = "SSL_AUTO_DOMAIN_NAMES"
	SQP_K_SSL_AUTO_RENEW_BEFORE    = "SSL_AUTO_RENEW_BEFORE"
	SQP_K_KEEP_ALIVE_TIMEOUT       = "KEEP_ALIVE_TIMEOUT"

	SQ_WD  = "/usr/local/serviceq"
	SQ_VER = "serviceq/0.4"
)

// getPropertyFilePath returns path to sq.properties.
func getPropertyFilePath() string {

	return SQ_WD + "/config/sq.properties"
}

// getProperties transforms sq.properties into config model and validates it.
func getProperties(confFilePath string) (*model.ServiceQProperties, error) {

	confFileSize := 0
	var cfg *model.Config
	var sqp *model.ServiceQProperties

	if fileStat, err := os.Stat(confFilePath); err == nil {
		confFileSize = int(fileStat.Size())
	} else {
		return sqp, err
	}

	if confFileSize > 0 {
		if file, err := os.Open(confFilePath); err == nil {
			defer file.Close()

			reader := bufio.NewReader(file)
			for {
				if line, _, err := reader.ReadLine(); err == nil {
					sline := string(line)
					kvpart := strings.Split(sline, "=")
					if len(kvpart) > 0 {
						cfg = populate(cfg, kvpart)
					}
				} else {
					if err.Error() == "EOF" {
						break
					}
				}
			}
		} else {
			return sqp, err
		}
	}

	validate(cfg)
	return getAssignedProperties(cfg), nil
}

// populate maps key/value pairs in sq.properties to corresponding config fields.
func populate(cfg *model.Config, kvpart []string) *model.Config {

	switch kvpart[0] {

	case SQP_K_LISTENER_PORT:
		cfg.ListenerPort = kvpart[1]
		fmt.Printf("serviceq listening on port> %s\n", cfg.ListenerPort)
	case SQP_K_PROTOCOL:
		cfg.Proto = kvpart[1]
	case SQP_K_ENDPOINTS:
		vpart := strings.Split(kvpart[1], ",")
		for _, s := range vpart {
			uri, err := url.ParseRequestURI(s)
			if err != nil || (uri.Scheme != "http" && uri.Scheme != "https") {
				fmt.Fprintf(os.Stderr, "Invalid endpoint.. exiting\n")
				os.Exit(1)
			}
			var endpoint model.Endpoint
			port := ""
			endpoint.RawUrl = s
			endpoint.Scheme = uri.Scheme
			if strings.IndexByte(uri.Host, ':') == -1 || (strings.IndexByte(uri.Host, ']') != -1 && strings.Index(uri.Host, "]:") == -1) {
				if uri.Scheme == "http" {
					port = ":80"
				} else if uri.Scheme == "https" {
					port = ":443"
				}
			}
			endpoint.QualifiedUrl = s + port
			endpoint.Host = uri.Host + port
			cfg.Endpoints = append(cfg.Endpoints, endpoint)
			fmt.Printf("service addr> %s\n", endpoint.QualifiedUrl)
		}
	case SQP_K_MAX_CONCURRENT_CONNS:
		cfg.ConcurrencyPeak, _ = strconv.ParseInt(kvpart[1], 10, 64)
		fmt.Printf("concurreny peak> %d\n", cfg.ConcurrencyPeak)
	case SQP_K_ENABLE_UPFRONT_Q:
		cfg.EnableUpfrontQ, _ = strconv.ParseBool(kvpart[1])
	case SQP_K_ENABLE_DEFERRED_Q:
		cfg.EnableDeferredQ, _ = strconv.ParseBool(kvpart[1])
	case SQP_K_Q_REQUEST_FORMATS:
		cfg.QRequestFormats = strings.Split(kvpart[1], ",")
	case SQP_K_RETRY_GAP:
		retryGapVal, _ := strconv.ParseInt(kvpart[1], 10, 32)
		cfg.RetryGap = int(retryGapVal)
	case SQP_K_OUT_REQUEST_TIMEOUT:
		reqTimeOutVal, _ := strconv.ParseInt(kvpart[1], 10, 32)
		cfg.OutRequestTimeout = int32(reqTimeOutVal)
	case SQP_K_RESPONSE_HEADERS:
		vpart := strings.Split(kvpart[1], "|")
		for _, s := range vpart {
			if s != "" {
				if strings.ToLower(s) == "server" {
					s += ": " + SQ_VER
				}
				cfg.CustomResponseHeaders = append(cfg.CustomResponseHeaders, s)
			}
		}
	case SQP_K_SSL_ENABLED:
		cfg.SSLEnabled, _ = strconv.ParseBool(kvpart[1])
		fmt.Printf("ssl enabled> %t\n", cfg.SSLEnabled)
	case SQP_K_SSL_CERTIFICATE_FILE:
		cfg.SSLCertificateFile = kvpart[1]
	case SQP_K_SSL_PRIVATE_KEY_FILE:
		cfg.SSLPrivateKeyFile = kvpart[1]
	case SQP_K_SSL_AUTO_ENABLED:
		cfg.SSLAutoEnabled, _ = strconv.ParseBool(kvpart[1])
		fmt.Printf("sslauto enabled> %t\n", cfg.SSLAutoEnabled)
	case SQP_K_SSL_AUTO_CERTIFICATE_DIR:
		cfg.SSLAutoCertificateDir = kvpart[1]
	case SQP_K_SSL_AUTO_EMAIL:
		cfg.SSLAutoEmail = kvpart[1]
	case SQP_K_SSL_AUTO_DOMAINS:
		cfg.SSLAutoDomains = kvpart[1]
	case SQP_K_SSL_AUTO_RENEW_BEFORE:
		autoCertRenewBefore, _ := strconv.ParseInt(kvpart[1], 10, 32)
		cfg.SSLAutoRenewBefore = int32(autoCertRenewBefore)
	case SQP_K_KEEP_ALIVE_TIMEOUT:
		keepAliveTimeout, _ := strconv.ParseInt(kvpart[1], 10, 32)
		cfg.KeepAliveTimeout = int32(keepAliveTimeout)
	default:
		break
	}

	return cfg
}

// validate does a mandatory fields check on sq.properties.
func validate(cfg *model.Config) {

	if cfg.Proto == "" || cfg.ListenerPort == "" || len(cfg.Endpoints) == 0 || cfg.ConcurrencyPeak <= 0 {
		fmt.Fprintf(os.Stderr, "Something wrong with sq.properties... exiting\n")
		os.Exit(1)
	}
}

// getAssignedProperties returns a new model.ServiceQProperties object
// with configs mapped from sq.properties and other default config values.
func getAssignedProperties(cfg *model.Config) *model.ServiceQProperties {

	return &model.ServiceQProperties{
		ListenerPort:          cfg.ListenerPort,
		Proto:                 cfg.Proto,
		ServiceList:           cfg.Endpoints,
		CustomRequestHeaders:  cfg.CustomRequestHeaders,
		CustomResponseHeaders: cfg.CustomResponseHeaders,
		MaxConcurrency:        cfg.ConcurrencyPeak,
		EnableUpfrontQ:        cfg.EnableUpfrontQ,
		EnableDeferredQ:       cfg.EnableDeferredQ,
		QRequestFormats:       cfg.QRequestFormats,
		MaxRetries:            len(cfg.Endpoints),
		RetryGap:              cfg.RetryGap,
		IdleGap:               500,
		RequestErrorLog:       make(map[string]uint64, len(cfg.Endpoints)),
		OutRequestTimeout:     cfg.OutRequestTimeout,
		SSLEnabled:            cfg.SSLEnabled,
		SSLCertificateFile:    cfg.SSLCertificateFile,
		SSLPrivateKeyFile:     cfg.SSLPrivateKeyFile,
		SSLAutoEnabled:        cfg.SSLAutoEnabled,
		SSLAutoCertificateDir: cfg.SSLAutoCertificateDir,
		SSLAutoEmail:          cfg.SSLAutoEmail,
		SSLAutoDomains:        cfg.SSLAutoDomains,
		SSLAutoRenewBefore:    cfg.SSLAutoRenewBefore,
		KeepAliveTimeout:      cfg.KeepAliveTimeout,
		KeepAliveServe:        keepAliveServe(cfg.CustomResponseHeaders),
	}
}

// keepAliveServe returns whether to use keep-alive or not.
func keepAliveServe(customResponseHeaders []string) bool {

	for _, h := range customResponseHeaders {
		h = strings.Replace(h, " ", "", -1)
		if strings.Contains(h, "Connection:keep-alive") {
			return true
		}
	}

	return false
}
