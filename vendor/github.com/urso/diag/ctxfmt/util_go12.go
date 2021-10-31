// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0

//+build go1.12

package ctxfmt

import "reflect"

func newMapIter(m reflect.Value) *reflect.MapIter {
	return m.MapRange()
}
