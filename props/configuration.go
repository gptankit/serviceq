package props

import (
	"bufio"
	"fmt"
	"model"
	"net/url"
	"os"
	"strconv"
	"strings"
)

const (
	SQP_K_LISTENER_PORT              = "LISTENER_PORT"
	SQP_K_PROTOCOL                   = "PROTO"
	SQP_K_ENDPOINTS                  = "ENDPOINTS"
	SQP_K_REQUEST_HEADERS            = "CUSTOM_REQUEST_HEADERS"
	SQP_K_RESPONSE_HEADERS           = "CUSTOM_RESPONSE_HEADERS"
	SQP_K_MAX_CONCURRENT_CONNS       = "CONCURRENCY_PEAK"
	SQP_K_ENABLE_DEFERRED_Q          = "ENABLE_DEFERRED_Q"
	SQP_K_DEFERRED_Q_REQUEST_FORMATS = "DEFERRED_Q_REQUEST_FORMATS"
	SQP_K_RETRY_GAP                  = "RETRY_GAP"
	SQP_K_OUT_REQUEST_TIMEOUT        = "OUTGOING_REQUEST_TIMEOUT"
	SQP_K_ENABLE_PROFILING_FOR       = "ENABLE_PROFILING_FOR"
)

func GetConfiguration(confFilePath string) (model.Config, error) {

	confFileSize := 0
	var cfg model.Config

	if fileStat, err := os.Stat(confFilePath); err == nil {
		confFileSize = int(fileStat.Size())
	} else {
		return cfg, err
	}

	if confFileSize > 0 {
		if file, err := os.Open(confFilePath); err == nil {
			defer file.Close()

			reader := bufio.NewReader(file)
			for {
				if line, _, err := reader.ReadLine(); err == nil {
					sline := string(line)
					kvpart := strings.Split(sline, "=")
					if kvpart != nil && len(kvpart) > 0 {
						cfg = populate(cfg, kvpart)
					}
				} else {
					if err.Error() == "EOF" {
						break
					}
				}
			}
		} else {
			return cfg, err
		}
	}

	return cfg, nil
}

func GetConfFileLocation() string {

	sqwd := "/opt/serviceq"
	return sqwd + "/config/sq.properties"
}

func populate(cfg model.Config, kvpart []string) model.Config {

	switch kvpart[0] {

	case SQP_K_LISTENER_PORT:
		cfg.ListenerPort = kvpart[1]
		break
	case SQP_K_PROTOCOL:
		cfg.Proto = kvpart[1]
		break
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
			if !strings.Contains(uri.Host, ":") {
				if uri.Scheme == "http" {
					port = ":80"
				} else if uri.Scheme == "https" {
					port = ":443"
				}
			}
			endpoint.QualifiedUrl = s + port
			endpoint.Host = uri.Host + port
			cfg.Endpoints = append(cfg.Endpoints, endpoint)
			fmt.Printf("Service Addr> %s\n", endpoint.QualifiedUrl)
		}
		break
	case SQP_K_MAX_CONCURRENT_CONNS:
		cfg.ConcurrencyPeak, _ = strconv.ParseInt(kvpart[1], 10, 64)
		fmt.Printf("Concurreny Peak> %d\n", cfg.ConcurrencyPeak)
		break
	case SQP_K_ENABLE_DEFERRED_Q:
		cfg.EnableDeferredQ, _ = strconv.ParseBool(kvpart[1])
		break
	case SQP_K_DEFERRED_Q_REQUEST_FORMATS:
		cfg.DeferredQRequestFormats = strings.Split(kvpart[1], ",")
		break
	case SQP_K_RETRY_GAP:
		retryGapVal, _ := strconv.ParseInt(kvpart[1], 10, 32)
		cfg.RetryGap = int(retryGapVal)
		break
	case SQP_K_OUT_REQUEST_TIMEOUT:
		timeoutVal, _ := strconv.ParseInt(kvpart[1], 10, 32)
		cfg.OutRequestTimeout = int32(timeoutVal)
		break
	case SQP_K_RESPONSE_HEADERS:
		vpart := strings.Split(kvpart[1], "|")
		for _, s := range vpart {
			if s != "" {
				cfg.CustomResponseHeaders = append(cfg.CustomResponseHeaders, s)
			}
		}
		break
	case SQP_K_ENABLE_PROFILING_FOR:
		cfg.EnableProfilingFor = kvpart[1]
		break
	default:
		break
	}

	return cfg
}
