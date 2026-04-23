// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package utils

import (
	"strings"

	"github.com/gofrs/uuid/v5"
)

// Generate a random UUID using the default algorithm (v7) without dashes.
func NewUUID() string {
	return strings.ReplaceAll(uuid.Must(uuid.NewV7()).String(), "-", "")
}
