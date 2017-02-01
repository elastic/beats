package monitoring

import (
	"encoding/json"
	"expvar"
	"strconv"
)

// VisitExpvars iterates all expvar metrics using the Visitor interface.
// The top-level metrics "memstats" and "cmdline", plus all monitoring.X metric types
// are ignored.
func VisitExpvars(vs Visitor) {
	vs.OnRegistryStart()
	expvar.Do(makeExparVisitor(0, vs))
	vs.OnRegistryFinished()
}

func DoExpvars(f func(string, interface{})) {
	VisitExpvars(NewKeyValueVisitor(f))
}

func makeExparVisitor(level int, vs Visitor) func(expvar.KeyValue) {
	return func(kv expvar.KeyValue) {
		if ignoreExpvar(level, kv) {
			return
		}

		name := kv.Key
		variable := kv.Value
		switch v := variable.(type) {
		case *expvar.Int:
			i, _ := strconv.ParseInt(v.String(), 10, 64)
			vs.OnKey(name)
			vs.OnInt(i)

		case *expvar.Float:
			f, _ := strconv.ParseFloat(v.String(), 64)
			vs.OnKey(name)
			vs.OnFloat(f)

		case *expvar.Map:
			vs.OnKey(name)
			vs.OnRegistryStart()
			v.Do(makeExparVisitor(level+1, vs))
			vs.OnRegistryFinished()

		default:
			vs.OnKey(name)
			s := v.String()
			if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
				var tmp string
				if err := json.Unmarshal([]byte(s), &tmp); err == nil {
					s = tmp
				}
			}
			vs.OnString(s)
		}
	}
}

// ignore if `monitoring` variable or some other internals
// autmoatically registered by expvar against our wishes
func ignoreExpvar(level int, kv expvar.KeyValue) bool {
	switch kv.Value.(type) {
	case makeExpvar, Var:
		return true
	}

	if level == 0 {
		switch kv.Key {
		case "memstats", "cmdline":
			return true
		}
	}

	return false
}
