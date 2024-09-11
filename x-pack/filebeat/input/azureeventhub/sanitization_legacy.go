// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix

package azureeventhub

import "errors"

type sanitizationOption string

const (
	newLines     sanitizationOption = "NEW_LINES"
	singleQuotes sanitizationOption = "SINGLE_QUOTES"
)

// sanitizeOptionsValidate validates for supported sanitization options
func sanitizeOptionsValidate(s string) error {
	switch s {
	case "NEW_LINES":
		return nil
	case "SINGLE_QUOTES":
		return nil
	default:
		return errors.New("invalid sanitization option")
	}
}
