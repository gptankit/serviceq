package profiling

import(
	"github.com/pkg/profile"
	"os/signal"
	"os"
	"syscall"
	"fmt"
)

func Start(profilingFor string) {

	if profilingFor != "" {
		var prof interface{}
		if profilingFor == "mem" {
			prof = profile.Start(profile.MemProfile) // options - CPUProfile, MemProfile, MutexProfile, BlockProfile, TraceProfile
		} else if profilingFor == "cpu" {
			prof = profile.Start(profile.CPUProfile) // options - CPUProfile, MemProfile, MutexProfile, BlockProfile, TraceProfile
		}

		csig := make(chan os.Signal, 1)
		signal.Notify(csig, syscall.SIGQUIT)
		go hookInterrupt(csig, prof)
	}
}

func hookInterrupt(csig chan os.Signal, prof interface{}) {
	for {
		s := <-csig
		switch s {
		case syscall.SIGQUIT: // ctrl + \
			fmt.Println("stop and core dump")
			if prof != nil {
				prof.(*profile.Profile).Stop() // write profile to disk
			}
			os.Exit(0)
		default:
			fmt.Println("Unknown signal.")
		}
	}
}
