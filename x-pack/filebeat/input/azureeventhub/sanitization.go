// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix
// +build !aix

package azureeventhub

import (
	"bytes"
	"errors"
)

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

// sanitize applies the sanitization options specified in the config
// if no sanitization options are provided, the message remains unchanged
func sanitize(jsonStr []byte, opts ...string) []byte {
	res := jsonStr

	for _, opt := range opts {
		switch sanitizationOption(opt) {
		case newLines:
			res = sanitizeNewLines(res)
		case singleQuotes:
			res = sanitizeSingleQuotes(res)
		}
	}

	return res
}

// sanitizeNewLines removes newlines found in the message
func sanitizeNewLines(jsonStr []byte) []byte {
	return bytes.ReplaceAll(jsonStr, []byte("\n"), []byte{})
}

// sanitizeSingleQuotes replaces single quotes with double quotes in the message
// single quotes that are in between double quotes remain unchanged
func sanitizeSingleQuotes(jsonStr []byte) []byte {
	var result bytes.Buffer

	inDoubleQuotes := false

	for _, r := range jsonStr {
		if r == '"' {
			inDoubleQuotes = !inDoubleQuotes
		}

		if r == '\'' && !inDoubleQuotes {
			result.WriteRune('"')
		} else {
			result.WriteByte(r)
		}
	}

	return result.Bytes()
}
