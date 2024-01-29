// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !linux

package ebpf

import "errors"

var ErrNotSupported = errors.New("not supported")

func NewWatcher() (Watcher, error) {
	return nil, ErrNotSupported
}
