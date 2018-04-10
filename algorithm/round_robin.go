package algorithm

import (
	"math"
)

func RoundRobin(set int, choice int) int {

	next := choice + 1

	return int(math.Mod(float64(next), float64(set)))
}
