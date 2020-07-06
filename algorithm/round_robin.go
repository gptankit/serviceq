package algorithm

import (
	"math"
)

// roundrobin implements round robin selection where choice
// is current selection and set is total selection space.
func roundrobin(set int, choice int) int {

	next := choice + 1

	return int(math.Mod(float64(next), float64(set)))
}
