package service

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"sync"
	"syscall"

	"github.com/elastic/beats/libbeat/logp"

	"net/http"
	_ "net/http/pprof"
)

// HandleSignals manages OS signals that ask the service/daemon to stop.
// The stopFunction should break the loop in the Beat so that
// the service shut downs gracefully.
func HandleSignals(stopFunction func()) {
	var callback sync.Once

	// On ^C or SIGTERM, gracefully stop the sniffer
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigc
		logp.Debug("service", "Received sigterm/sigint, stopping")
		callback.Do(stopFunction)
	}()

	// Handle the Windows service events
	go ProcessWindowsControlEvents(func() {
		logp.Debug("service", "Received svc stop/shutdown request")
		callback.Do(stopFunction)
	})
}

// cmdline flags
var memprofile, cpuprofile, httpprof *string
var cpuOut *os.File

func init() {
	memprofile = flag.String("memprofile", "", "Write memory profile to this file")
	cpuprofile = flag.String("cpuprofile", "", "Write cpu profile to file")
	httpprof = flag.String("httpprof", "", "Start pprof http server")
}

// WithMemProfile returns whether the beat should write the memory profile to file
func WithMemProfile() bool {
	return *memprofile != ""
}

// WithCpuProfile returns whether the beat should write the CPU profile file
func WithCpuProfile() bool {
	return *cpuprofile != ""
}

// BeforeRun takes care of necessary actions such as creating files
// before the beat should run.
func BeforeRun() {
	if WithCpuProfile() {
		cpuOut, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(cpuOut)
	}

	if *httpprof != "" {
		go func() {
			logp.Info("start pprof endpoint")
			logp.Info("finished pprof endpoint: %v", http.ListenAndServe(*httpprof, nil))
		}()
	}
}

// Cleanup handles cleaning up the runtime and OS environments. This includes
// tasks such as stopping the CPU profile if it is running.
func Cleanup() {
	if WithCpuProfile() {
		pprof.StopCPUProfile()
		cpuOut.Close()
	}

	if WithMemProfile() {
		runtime.GC()

		writeHeapProfile(*memprofile)

		debugMemStats()
	}
}

func debugMemStats() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	logp.Debug("mem", "Memory stats: In use: %d Total (even if freed): %d System: %d",
		m.Alloc, m.TotalAlloc, m.Sys)
}

func writeHeapProfile(filename string) {
	f, err := os.Create(filename)
	if err != nil {
		logp.Err("Failed creating file %s: %s", filename, err)
		return
	}
	pprof.WriteHeapProfile(f)
	f.Close()

	logp.Info("Created memory profile file %s.", filename)
}
