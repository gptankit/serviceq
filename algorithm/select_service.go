package algorithm

import (
	"math"
	"model"
	"math/rand"
)

func ChooseServiceIndex(sqp *model.ServiceQProperties, initialChoice int, retry int) int {

	noOfServices := len((*sqp).ServiceList)

	if noOfServices <= 0 {
		return noOfServices
	}

	if retry == 0 { // first time
		sumErr := 0
		minErr := math.MaxUint32
		(*sqp).REMutex.Lock()
		defer (*sqp).REMutex.Unlock()
		for _, errCnt := range (*sqp).RequestErrorLog {
			sumErr += errCnt
			if errCnt < minErr {
				minErr = errCnt
			}
		}
		if sumErr == 0 {
			return randomize(0, noOfServices)
		} else {
			randx := randomize(minErr, sumErr+1)
			perm := rand.Perm(noOfServices)
			for _, si := range perm {
				diff := randx - (*sqp).RequestErrorLog[(*sqp).ServiceList[si].QualifiedUrl]
				if diff >= 0 {
					return si
				}
			}
			return randomize(0, noOfServices)
		}
	} else {
		return roundrobin(noOfServices, initialChoice)
	}
}
