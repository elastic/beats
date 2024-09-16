// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix

package azureeventhub

import "errors"

// This file supports the legacy sanitization options for the Azure Event Hub input.
//
// The legacy offered two sanitization options using the `sanitize_options`
// configuration option:
//
// - NEW_LINES: replaces new lines with spaces
// - SINGLE_QUOTES: replaces single quotes with double quotes
//
// The legacy `sanitize_options` is deprecated and will be removed in the
// 9.0 release.
// Users should use the `sanitizers` configuration option instead.
//
// However, the current sanitization implementation honors the legacy sanitization
// options and applies them to the sanitizers configuration.

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
