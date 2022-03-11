package tcputils

import (
	"errors"
	"net"
)

const (
	SERVICEQ_NO_ERR      = 600
	SERVICEQ_FLOODED_ERR = 601
	UPSTREAM_NO_ERR      = 700
	UPSTREAM_TCP_ERR     = 701
	UPSTREAM_HTTP_ERR    = 702

	RESPONSE_FLOODED      = "SERVICEQ_FLOODED"
	RESPONSE_TIMED_OUT    = "UPSTREAM_TIMED_OUT"
	RESPONSE_SERVICE_DOWN = "UPSTREAM_DOWN"
	RESPONSE_NO_RESPONSE  = "UPSTREAM_NO_RESPONSE"
)

// EvalError evaluates the type of errors from upstream node
func EvalError(err error) error {

	nodeErr := err
	if nodeErr != nil {
		if e, ok := nodeErr.(net.Error); ok && e.Timeout() {
			nodeErr = errors.New(RESPONSE_TIMED_OUT)
		} else {
			nodeErr = errors.New(RESPONSE_NO_RESPONSE)
		}
	} else {
		nodeErr = errors.New(RESPONSE_NO_RESPONSE)
	}

	return nodeErr
}
