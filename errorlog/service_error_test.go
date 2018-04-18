package errorlog

import (
	"model"
	"testing"
)

func BenchmarkConcurrentErrorIncrements(b *testing.B) {

	sqp := model.ServiceQProperties{}
	sqp.RequestErrorLog = make(map[string]int, 1)

	sqp.RequestErrorLog["s0"] = 0

	// concurrent access to map
	for i := 0; i < b.N; i++ {
		go IncrementErrorCount(&sqp, "s0", 1, "SERVICE_DOWN")
	}
}

func BenchmarkSequentialErrorIncrements(b *testing.B) {

	sqp := model.ServiceQProperties{}
	sqp.RequestErrorLog = make(map[string]int, 1)

	sqp.RequestErrorLog["s0"] = 0

	// sequential access to map
	for i := 0; i < b.N; i++ {
		IncrementErrorCount(&sqp, "s0", 1, "SERVICE_DOWN")
	}
}
