// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package id

import (
	"math/rand"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"
)

// ID represents a unique ID.
type ID = ulid.ULID

// rand.New is not threadsafe, so we create a pool of rand to speed up the id generation.
var randPool = sync.Pool{
	New: func() interface{} {
		t := time.Now()
		return rand.New(rand.NewSource(t.UnixNano()))
	},
}

// Generate returns and ID or an error if we cannot generate an ID.
func Generate() (ID, error) {
	r := randPool.Get().(*rand.Rand)
	defer randPool.Put(r)

	t := time.Now()
	entropy := ulid.Monotonic(r, 0)
	return ulid.New(ulid.Timestamp(t), entropy)
}
