// +build !tracing

package txfile

func pushTracer(t tracer) {}
func popTracer()          {}

func traceln(vs ...interface{})            {}
func tracef(fmt string, vs ...interface{}) {}
