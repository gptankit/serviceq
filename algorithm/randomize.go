package algorithm

import (
	"math/rand"
	"time"
)

func randomize(init int, set int) int {

	rand.Seed(time.Now().UTC().UnixNano())
	choice := rand.Intn(set - init) + init

	return choice
}
