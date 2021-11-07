// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0

//+build !go1.12

package ctxfmt

import "reflect"

type mapIter struct {
	m    reflect.Value
	keys []reflect.Value
	pos  int
}

func newMapIter(m reflect.Value) *mapIter {
	return &mapIter{
		m:    m,
		keys: m.MapKeys(),
		pos:  -1,
	}
}

func (i *mapIter) Next() bool {
	i.pos++
	return i.pos < len(i.keys)
}

func (i *mapIter) Key() reflect.Value {
	return i.keys[i.pos]
}

func (i *mapIter) Value() reflect.Value {
	return i.m.MapIndex(i.Key())
}
