// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package service

import (
	"context"
	"expvar"
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"sync"
	"syscall"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/monitoring"
)

// HandleSignals manages OS signals that ask the service/daemon to stop.
// The stopFunction should break the loop in the Beat so that
// the service shut downs gracefully.
func HandleSignals(stopFunction func(), cancel context.CancelFunc) {
	var callback sync.Once

	// On ^C or SIGTERM, gracefully stop the sniffer
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigc
		logp.Debug("service", "Received sigterm/sigint, stopping")
		cancel()
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

// ProfileEnabled checks whether the beat should write a cpu or memory profile.
func ProfileEnabled() bool {
	return withMemProfile() || withCPUProfile()
}

func withMemProfile() bool { return *memprofile != "" }
func withCPUProfile() bool { return *cpuprofile != "" }

// BeforeRun takes care of necessary actions such as creating files
// before the beat should run.
func BeforeRun() {
	if withCPUProfile() {
		cpuOut, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(cpuOut)
	}

	if *httpprof != "" {
		logp.Info("start pprof endpoint")
		go func() {
			mux := http.NewServeMux()

			// register pprof handler
			mux.HandleFunc("/debug/pprof/", func(w http.ResponseWriter, r *http.Request) {
				http.DefaultServeMux.ServeHTTP(w, r)
			})

			// register metrics handler
			mux.HandleFunc("/debug/vars", metricsHandler)

			endpoint := http.ListenAndServe(*httpprof, mux)
			logp.Info("finished pprof endpoint: %v", endpoint)
		}()
	}
}

// report expvar and all libbeat/monitoring metrics
func metricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	first := true
	report := func(key string, value interface{}) {
		if !first {
			fmt.Fprintf(w, ",\n")
		}
		first = false
		if str, ok := value.(string); ok {
			fmt.Fprintf(w, "%q: %q", key, str)
		} else {
			fmt.Fprintf(w, "%q: %v", key, value)
		}
	}

	fmt.Fprintf(w, "{\n")
	monitoring.Do(monitoring.Full, report)
	expvar.Do(func(kv expvar.KeyValue) {
		report(kv.Key, kv.Value)
	})
	fmt.Fprintf(w, "\n}\n")
}

// Cleanup handles cleaning up the runtime and OS environments. This includes
// tasks such as stopping the CPU profile if it is running.
func Cleanup() {
	if withCPUProfile() {
		pprof.StopCPUProfile()
		cpuOut.Close()
	}

	if withMemProfile() {
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
