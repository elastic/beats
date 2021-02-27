// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package reporter

import "time"

// Event is a reported event.
type Event interface {
	Type() string
	SubType() string
	Time() time.Time
	Message() string
	Payload() map[string]interface{}
}

type event struct {
	eventype  string
	subType   string
	timestamp time.Time
	message   string
	payload   map[string]interface{}
}

func (e event) Type() string                    { return e.eventype }
func (e event) SubType() string                 { return e.subType }
func (e event) Time() time.Time                 { return e.timestamp }
func (e event) Message() string                 { return e.message }
func (e event) Payload() map[string]interface{} { return e.payload }
