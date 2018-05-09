// +build tracing

package pq

import (
	"github.com/elastic/go-txfile/internal/tracelog"
)

var (
	tracers      []tracer
	activeTracer tracer
)

func init() {
	logTracer = tracelog.Get("pq")
	activeTracer = logTracer
}

func pushTracer(t tracer) {
	tracers = append(tracers, activeTracer)
	activeTracer = t
}

func popTracer() {
	i := len(tracers) - 1
	activeTracer = tracers[i]
	tracers = tracers[:i]
}

func traceln(vs ...interface{}) {
	activeTracer.Println(vs...)
}

func tracef(s string, vs ...interface{}) {
	activeTracer.Printf(s, vs...)
}
