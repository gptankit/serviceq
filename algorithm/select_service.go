package algorithm

import (
	"math"

	"github.com/gptankit/serviceq/model"
)

// ChooseServiceIndex implements the routing logic to the cluster of upstream services. On
// first try, an error log lookup is done to determine the service-wise error count and effective
// error is calculated. If no error found for any service, random service selection (equal probability)
// is done, else weighted random service selection is done, where weights are inversely proportional
// to error count on the particular service. If the request to the selected service fails, round robin
// selection is done to deterministically select the next service.
func ChooseServiceIndex(sqp *model.ServiceQProperties, initialChoice int, retry int) int {

	noOfServices := len(sqp.ServiceList)

	// single endpoint
	// invalid num of endpoints
	if noOfServices <= 1 {
		return 0
	}

	if retry == 0 { // first time
		sqp.REMutex.Lock()
		defer sqp.REMutex.Unlock()
		maxErr := uint64(0)
		for _, n := range sqp.ServiceList {
			errCnt := sqp.RequestErrorLog[n.QualifiedUrl]
			effectiveErr := uint64(math.Floor(math.Pow(float64(1+errCnt), 1.5)))
			if effectiveErr >= maxErr {
				maxErr = effectiveErr
			}
		}
		if maxErr == 1 {
			return randomize(0, noOfServices)
		} else {
			weights := make([]float64, noOfServices)
			prefixes := make([]float64, noOfServices)
			for i, n := range sqp.ServiceList {
				errCnt := sqp.RequestErrorLog[n.QualifiedUrl]
				weights[i] = math.Ceil(float64(maxErr) / float64(errCnt+1))
			}
			for i, _ := range weights {
				if i == 0 {
					prefixes[i] = weights[i]
				} else {
					prefixes[i] = weights[i] + prefixes[i-1]
				}
			}
			prLen := noOfServices - 1
			randx := randomize64(1, int64(prefixes[prLen])+1)
			ceil := findCeilIn(randx, prefixes, 0, prLen)
			if ceil >= 0 {
				return ceil
			}
		}
		return randomize(0, noOfServices)
	} else {
		return roundrobin(noOfServices, initialChoice)
	}
}

// findCeilIn does a binary search to find position of selected random
// number and returns corresponding ceil index in prefixes array
func findCeilIn(randx int64, prefixes []float64, start int, end int) int {

	var mid int
	for {
		if start >= end {
			break
		}
		mid = start + ((end - start) >> 1)
		if randx > int64(prefixes[mid]) {
			start = mid + 1
		} else {
			end = mid
		}
	}

	if randx <= int64(prefixes[start]) {
		return start
	}
	return -1
}
