package layout

import (
	"time"

	"github.com/urso/ecslog/fld"
)

func DynTimestamp(layout string) fld.Field {
	return fld.Field{Key: "@timestamp", Standardized: true, Value: fld.Value{
		String: layout, Reporter: _tsReporter,
	}}
}

type tsReporter struct{}

var _tsReporter = tsReporter{}

func (tsReporter) Type() fld.Type {
	return fld.StringType
}

func (tsReporter) Ifc(v *fld.Value, fn func(interface{})) {
	fn(time.Now().Format(v.String))
}
