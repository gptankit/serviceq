package netserve

import (
	"algorithm"
)

func ChooseServiceIndex(noOfServices int, initialChoice int, retry int) int {

	if noOfServices <= 0 {
		return noOfServices
	}

	if retry == 0 { // first time
		return algorithm.Randomize(noOfServices)
	} else {
		return algorithm.RoundRobin(noOfServices, initialChoice)
	}
}
