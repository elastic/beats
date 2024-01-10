// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package provider

import (
	"github.com/elastic/beats/v7/libbeat/beat"
)

type Provider interface {
	UpdateDB(*beat.Event) error
}
