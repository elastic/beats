// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0

package ctxfmt

type argstate struct {
	idx  int
	args []interface{}
}

func (a *argstate) next() (arg interface{}, idx int, has bool) {
	if a.idx < len(a.args) {
		arg, idx = a.args[a.idx], a.idx
		a.idx++
		return arg, idx, true
	}
	return nil, len(a.args), false
}
