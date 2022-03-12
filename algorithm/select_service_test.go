package algorithm

import (
	"testing"

	"github.com/gptankit/serviceq/model"
)

func TestServiceIndexWithinBoundR1(t *testing.T) {

	var params = []struct {
		sqp *model.ServiceQProperties
		ic  int
		rt  int
	}{
		{&model.ServiceQProperties{
			RequestErrorLog: map[string]uint64{"s0": 1, "s1": 2},
			ServiceList: []model.Endpoint{
				model.Endpoint{QualifiedUrl: "s0"},
				model.Endpoint{QualifiedUrl: "s1"},
			}},
			5,
			0},
		{&model.ServiceQProperties{
			RequestErrorLog: map[string]uint64{"s0": 1, "s1": 2},
			ServiceList: []model.Endpoint{
				model.Endpoint{QualifiedUrl: "s0"},
				model.Endpoint{QualifiedUrl: "s1"},
			}},
			8,
			0},
		{&model.ServiceQProperties{
			RequestErrorLog: map[string]uint64{"s0": 1, "s1": 2},
			ServiceList: []model.Endpoint{
				model.Endpoint{QualifiedUrl: "s0"},
				model.Endpoint{QualifiedUrl: "s1"},
			}},
			1,
			0},
		{&model.ServiceQProperties{
			RequestErrorLog: map[string]uint64{"s0": 1, "s1": 2},
			ServiceList: []model.Endpoint{
				model.Endpoint{QualifiedUrl: "s0"},
				model.Endpoint{QualifiedUrl: "s1"},
			}},
			2,
			0},
	}

	for _, prm := range params {
		ce := ChooseServiceIndex(prm.sqp, prm.ic, prm.rt)

		ns := len((*prm.sqp).ServiceList)
		if ns > 0 && (ce < 0 || ce >= ns) {
			t.Errorf("service index out of bound, ns=%d, ic=%d, rt=%d --> ce=%d\n", ns, prm.ic, prm.rt, ce)
		} else if ns <= 0 && ce != ns {
			t.Errorf("service index out of bound, ns=%d, ic=%d, rt=%d --> ce=%d\n", ns, prm.ic, prm.rt, ce)
		}
	}
}

func TestServiceIndexWithinBoundRn(t *testing.T) {

	var params = []struct {
		sqp *model.ServiceQProperties
		ic  int
		rt  int
	}{
		{&model.ServiceQProperties{
			RequestErrorLog: map[string]uint64{"s0": 1, "s1": 2},
			ServiceList: []model.Endpoint{
				model.Endpoint{QualifiedUrl: "s0"},
				model.Endpoint{QualifiedUrl: "s1"},
			}},
			59,
			3},
		{&model.ServiceQProperties{
			RequestErrorLog: map[string]uint64{"s0": 1, "s1": 2},
			ServiceList: []model.Endpoint{
				model.Endpoint{QualifiedUrl: "s0"},
				model.Endpoint{QualifiedUrl: "s1"},
			}},
			8,
			2},
		{&model.ServiceQProperties{
			RequestErrorLog: map[string]uint64{"s0": 1, "s1": 2},
			ServiceList: []model.Endpoint{
				model.Endpoint{QualifiedUrl: "s0"},
				model.Endpoint{QualifiedUrl: "s1"},
			}},
			14,
			-1},
		{&model.ServiceQProperties{
			RequestErrorLog: map[string]uint64{"s0": 1, "s1": 2},
			ServiceList: []model.Endpoint{
				model.Endpoint{QualifiedUrl: "s0"},
				model.Endpoint{QualifiedUrl: "s1"},
			}},
			20,
			-1},
	}

	for _, prm := range params {
		ce := ChooseServiceIndex(prm.sqp, prm.ic, prm.rt)

		ns := len((*prm.sqp).ServiceList)
		if ns > 0 && (ce < 0 || ce >= ns) {
			t.Errorf("service index out of bound, ns=%d, ic=%d, rt=%d --> ce=%d\n", ns, prm.ic, prm.rt, ce)
		} else if ns <= 0 && ce != ns {
			t.Errorf("service index out of bound, ns=%d, ic=%d, rt=%d --> ce=%d\n", ns, prm.ic, prm.rt, ce)
		}
	}
}
