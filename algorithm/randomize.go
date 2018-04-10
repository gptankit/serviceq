package algorithm

import (
	"math/rand"
	"time"
)

func Randomize(set int) int {

	rand.Seed(time.Now().UTC().Unix())
	choice := rand.Intn(set)

	return choice
}
