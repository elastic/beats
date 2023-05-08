// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package shipper

import (
	"sync/atomic"
)

type shipperAcker struct {
	persistedIndex uint64
}

func newShipperAcker() *shipperAcker {
	return &shipperAcker{persistedIndex: 0}
}

// Update the input's persistedIndex by adding total to it.
// Despite the name, "total" here means an incremental total, i.e.
// the total number of events that are being acknowledged by this callback, not the total that have been sent overall.
// The acked parameter includes only those events that were successfully sent upstream rather than dropped by processors, etc.,
// but since we don't make that distinction in persistedIndex we can probably ignore it.
func (acker *shipperAcker) Track(_ int, total int) {
	atomic.AddUint64(&acker.persistedIndex, uint64(total))
}

func (acker *shipperAcker) PersistedIndex() uint64 {
	return acker.persistedIndex
}
