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

func randomize64(init int64, set int64) int64 {

	if set <= init {
		return int64(0)
	}

	rand.Seed(time.Now().UTC().UnixNano())
	choice := rand.Int63n(set - init) + init

	return choice
}
