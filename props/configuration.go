package props

import (
	"bufio"
	"fmt"
	"model"
	"os"
	"strconv"
	"strings"
)

const (
	SQP_K_LISTENER_PORT        = "LISTENER_PORT"
	SQP_K_PROTOCOL             = "PROTO"
	SQP_K_ENDPOINTS            = "ENDPOINTS"
	SQP_K_REQUEST_HEADERS      = "CUSTOM_REQUEST_HEADERS"
	SQP_K_RESPONSE_HEADERS     = "CUSTOM_RESPONSE_HEADERS"
	SQP_K_MAX_CONCURRENT_CONNS = "CONCURRENCY_PEAK"
	SQP_K_OUT_REQ_TIMEOUT      = "OUTGOING_REQUEST_TIMEOUT"
	SQP_K_ENABLE_PROFILING_FOR = "ENABLE_PROFILING_FOR"
)

func GetConfiguration(confFilePath string) (model.Config, error) {

	confFileSize := 0
	var config model.Config

	if fileStat, err := os.Stat(confFilePath); err == nil {
		confFileSize = int(fileStat.Size())
	} else {
		return config, err
	}

	if confFileSize > 0 {
		if file, err := os.Open(confFilePath); err == nil {
			defer file.Close()

			reader := bufio.NewReader(file)
			for {
				if line, _, err := reader.ReadLine(); err == nil {
					sline := string(line)
					kvpart := strings.Split(sline, "=")
					if kvpart != nil {
						if kvpart[0] == SQP_K_LISTENER_PORT {
							config.ListenerPort = kvpart[1]
						} else if kvpart[0] == SQP_K_PROTOCOL {
							config.Proto = kvpart[1]
						} else if kvpart[0] == SQP_K_ENDPOINTS {
							vpart := strings.Split(kvpart[1], ",")
							for _, s := range vpart {
								if s != "" {
									config.Endpoints = append(config.Endpoints, s)
									fmt.Printf("Service Addr> %s\n", s)
								}
							}
						} else if kvpart[0] == SQP_K_MAX_CONCURRENT_CONNS {
							config.ConcurrencyPeak, _ = strconv.ParseInt(kvpart[1], 10, 64)
							fmt.Printf("Concurreny Peak> %s\n", kvpart[1])
						} else if kvpart[0] == SQP_K_OUT_REQ_TIMEOUT {
							timeoutVal, _ := strconv.ParseInt(kvpart[1], 10, 32)
							config.OutReqTimeout = int32(timeoutVal)
						} else if kvpart[0] == SQP_K_RESPONSE_HEADERS {
							vpart := strings.Split(kvpart[1], "|")
							for _, s := range vpart {
								if s != "" {
									config.CustomResponseHeaders = append(config.CustomResponseHeaders, s)
								}
							}
						} else if kvpart[0] == SQP_K_ENABLE_PROFILING_FOR {
							config.EnableProfilingFor = kvpart[1]
						}
					}
				} else {
					if err.Error() == "EOF" {
						break
					}
				}
			}
		} else {
			return config, err
		}
	}

	return config, nil
}
