package algorithm

import (
	"math/rand"
	"time"
)

func randomize(init int, set int) int {

	if set <= init {
		return 0
	}

	rand.Seed(time.Now().UTC().UnixNano())
	choice := rand.Intn(set - init) + init

	return choice
}
