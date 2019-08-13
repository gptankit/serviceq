package algorithm

import (
	"math"
	"model"
)

func ChooseServiceIndex(sqp *model.ServiceQProperties, initialChoice int, retry int) int {

	noOfServices := len((*sqp).ServiceList)

	// single endpoint
	// invalid num of endpoints
	if noOfServices <= 1 {
		return 0
	}

	if retry == 0 { // first time
		(*sqp).REMutex.Lock()
		defer (*sqp).REMutex.Unlock()
		maxErr := uint64(0)
		slLen := len((*sqp).ServiceList)
		for _, n := range (*sqp).ServiceList {
			errCnt := (*sqp).RequestErrorLog[n.QualifiedUrl]
			effectiveErr := uint64(math.Floor(math.Pow(float64(1 + errCnt), 1.5)))
			if effectiveErr >= maxErr {
				maxErr = effectiveErr
			}
		}
		if maxErr == 1 {
			return randomize(0, noOfServices)
		} else {
			weights := make([]float64, slLen)
			prefixes := make([]float64, slLen)
			for i, n := range (*sqp).ServiceList {
				errCnt := (*sqp).RequestErrorLog[n.QualifiedUrl]
				weights[i] = math.Ceil(float64(maxErr) / float64(errCnt + 1))
			}
			for i, _ := range weights {
				if i == 0 {
					prefixes[i] = weights[i]
				} else {
					prefixes[i] = weights[i] + prefixes[i-1]
				}
			}
			prLen := len(prefixes) - 1
			randx := randomize64(1, int64(prefixes[prLen]) + 1)
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
