// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package ebpf

import (
	"github.com/elastic/ebpfevents"
)

type EventMask uint64

type Watcher interface {
	Subscribe(string, EventMask) <-chan ebpfevents.Record
	Unsubscribe(string)
}
