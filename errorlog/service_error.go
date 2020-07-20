package errorlog

import (
	"github.com/gptankit/serviceq/model"
	"log"
	"os"
)

var logger *log.Logger

// init opens the log file and creates a logger object.
func init() {

	logFileLocation := "/usr/local/serviceq/logs/serviceq_error.log"
	file, err := os.OpenFile(logFileLocation, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err == nil {
		logger = log.New(file, "ServiceQ: ", log.Ldate|log.Ltime)
	}
}

// IncrementErrorCount increments session error count and logs corresponding to service.
func IncrementErrorCount(sqp *model.ServiceQProperties, service string, errType int, errReason string) {

	sqp.REMutex.Lock()
	sqp.RequestErrorLog[service] += 1
	sqp.REMutex.Unlock()
	logServiceError(service, errType, errReason)
}

// ResetErrorCount resets session error count corresponding to service.
func ResetErrorCount(sqp *model.ServiceQProperties, service string) {

	sqp.REMutex.Lock()
	sqp.RequestErrorLog[service] = 0
	sqp.REMutex.Unlock()
}

// LogGenericError logs any given error data in the log file.
func LogGenericError(errData string) {

	if logger != nil {
		logger.Printf("%s", errData)
	}
}

// logServiceError logs service error data (errType and errReason) in the log file.
func logServiceError(service string, errType int, errReason string) {

	if logger != nil {
		logger.Printf("Error detected on %s [Code: %d, %s]", service, errType, errReason)
	}
}
