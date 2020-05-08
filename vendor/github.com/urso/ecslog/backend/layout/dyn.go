// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0

package layout

import (
	"time"

	"github.com/urso/diag"
)

func DynTimestamp(layout string) diag.Field {
	return diag.Field{Key: "@timestamp", Standardized: true, Value: diag.Value{
		String: layout, Reporter: _tsReporter,
	}}
}

type tsReporter struct{}

var _tsReporter = tsReporter{}

func (tsReporter) Type() diag.Type {
	return diag.StringType
}

func (tsReporter) Ifc(v *diag.Value, fn func(interface{})) {
	fn(time.Now().Format(v.String))
}
