// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package utils

import (
	"github.com/gofrs/uuid/v5"
)

// Generate a random UUID using the v4 algorithm
func NewUUIDV4() string {
	return uuid.Must(uuid.NewV4()).String()
}
