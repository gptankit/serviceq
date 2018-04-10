package errorlog

import (
	"model"
)

func IncrementErrorCount(sqprops *model.ServiceQProperties, service string) {

	(*sqprops).REMutex.Lock()
	defer (*sqprops).REMutex.Unlock()
	(*sqprops).RequestErrorLog[service] += 1
}

func ResetErrorCount(sqprops *model.ServiceQProperties, service string) {

	(*sqprops).REMutex.Lock()
	defer (*sqprops).REMutex.Unlock()
	(*sqprops).RequestErrorLog[service] = 0
}
