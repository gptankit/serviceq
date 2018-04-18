package errorlog

import (
	"model"
)

func IncrementErrorCount(sqp *model.ServiceQProperties, service string) {

	(*sqp).REMutex.Lock()
	defer (*sqp).REMutex.Unlock()
	(*sqp).RequestErrorLog[service] += 1
}

func ResetErrorCount(sqp *model.ServiceQProperties, service string) {

	(*sqp).REMutex.Lock()
	defer (*sqp).REMutex.Unlock()
	(*sqp).RequestErrorLog[service] = 0
}
