package netserve

import (
	"math"
	"testing"
)

func TestServiceIndexWithinBoundR1(t *testing.T) {

	var params = []struct {
		ns int
		ic int
		rt int
	}{
		{10, 5, 0},
		{10, 12, 0},
		{0, 70, 0},
		{-1, 3, 0},
	}

	for _, prm := range params {
		ce := ChooseServiceIndex(prm.ns, prm.ic, prm.rt)

		if prm.ns > 0 && (ce < 0 || ce >= prm.ns) {
			t.Errorf("service index out of bound, ns=%d, ic=%d, rt=%d --> ce=%d\n", prm.ns, prm.ic, prm.rt, ce)
		} else if prm.ns <= 0 && ce != prm.ns {
			t.Errorf("service index out of bound, ns=%d, ic=%d, rt=%d --> ce=%d\n", prm.ns, prm.ic, prm.rt, ce)
		}
	}
}

func TestServiceIndexWithinBoundRn(t *testing.T) {

	var params = []struct {
		ns int
		ic int
		rt int
	}{
		{10, 5, 4},
		{10, 12, 2},
		{0, 70, 1},
		{-1, 3, -1},
		{-1, 3, 2},
	}

	for _, prm := range params {
		ce := ChooseServiceIndex(prm.ns, prm.ic, prm.rt)

		if prm.ns > 0 && (ce < 0 || ce >= prm.ns) {
			t.Errorf("service index out of bound, ns=%d, ic=%d, rt=%d --> ce=%d\n", prm.ns, prm.ic, prm.rt, ce)
		} else if prm.ns <= 0 && ce != prm.ns {
			t.Errorf("service index out of bound, ns=%d, ic=%d, rt=%d --> ce=%d\n", prm.ns, prm.ic, prm.rt, ce)
		}
	}
}

func TestServiceIndexCorrectnessRn(t *testing.T) {

	var params = []struct {
		ns int
		ic int
		rt int
	}{
		{10, 5, 4},
		{10, 12, 2},
		{0, 70, 1},
		{-1, 3, -1},
		{-1, 3, 2},
	}

	for _, prm := range params {
		ce := ChooseServiceIndex(prm.ns, prm.ic, prm.rt)
		ev := int(math.Mod(float64(prm.ic+1), float64(prm.ns)))
		if prm.ns > 0 && ev != ce {
			t.Errorf("service index incorrect, ns=%d, ic=%d, rt=%d --> ce=%d\n", prm.ns, prm.ic, prm.rt, ce)
		} else if prm.ns <= 0 && ce != prm.ns {
			t.Errorf("service index incorrect, ns=%d, ic=%d, rt=%d --> ce=%d\n", prm.ns, prm.ic, prm.rt, ce)
		}
	}
}
