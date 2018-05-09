package pq

type tracer interface {
	Println(...interface{})
	Printf(string, ...interface{})
}

var (
	logTracer tracer
)
