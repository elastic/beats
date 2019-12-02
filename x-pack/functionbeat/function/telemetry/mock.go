// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package telemetry

type mock struct{}

// NewMock returns a new telemetry mock.
func NewMock() T {
	return &mock{}
}

// AddTriggeredFunction does nothing.
func (m *mock) AddTriggeredFunction(_ string, _ int64, _ int64) {
	return
}
