// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package atomic

import "sync/atomic"

type Bool struct {
	value uint32
}

func (b *Bool) Store(value bool) {
	atomic.StoreUint32(&b.value, encodeBool(value))
}

func (b *Bool) CAS(old bool, new bool) (swapped bool) {
	return atomic.CompareAndSwapUint32(&b.value, encodeBool(old), encodeBool(new))
}

func (b *Bool) Load() (value bool) {
	return atomic.LoadUint32(&b.value) != 0
}

func encodeBool(value bool) (result uint32) {
	if value {
		result = 1
	}
	return
}
