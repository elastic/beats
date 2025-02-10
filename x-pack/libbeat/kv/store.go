// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kv

import "time"

type KV interface {
	Open() error
	Get([]byte) ([]byte, error)
	Set([]byte, []byte, time.Duration) error
	Close() error
}
