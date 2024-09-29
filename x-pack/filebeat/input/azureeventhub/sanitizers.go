// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix

package azureeventhub

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
)

const (
	SanitizerNewLines     = "new_lines"
	SanitizerSingleQuotes = "single_quotes"
	SanitizerReplaceAll   = "replace_all"
)

// ----------------------------------------------------------------------------
// Sanitizer API
// ----------------------------------------------------------------------------

// SanitizerSpec defines a sanitizer configuration.
//
// Sanitizers can use the spec field to provide additional
// configuration.
type SanitizerSpec struct {
	Type string                 `config:"type"`
	Spec map[string]interface{} `config:"spec"`
}

// Sanitizer defines the interface for sanitizing JSON data.
//
// Implementing `Init` is optional. If implemented, it should
// be used to initialize the sanitizer with the provided spec.
type Sanitizer interface {
	Sanitize(jsonByte []byte) []byte
	Init() error
}

// ----------------------------------------------------------------------------
// Convenience builder functions
// ----------------------------------------------------------------------------

// newSanitizer creates a new sanitizer based on the provided spec.
func newSanitizer(spec SanitizerSpec) (Sanitizer, error) {
	var s Sanitizer

	switch spec.Type {
	case SanitizerNewLines:
		s = &newLinesSanitizer{}
	case SanitizerSingleQuotes:
		s = &singleQuotesSanitizer{}
	case SanitizerReplaceAll:
		s = &replaceAllSanitizer{spec: spec.Spec}
	default:
		return nil, fmt.Errorf("unknown sanitizer type: %s", spec.Type)
	}

	// Initialize the sanitizer with the provided spec.
	err := s.Init()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize sanitizer '%s': %w", spec.Type, err)
	}

	return s, nil
}

// newSanitizers creates a list of sanitizers based on the provided specs.
//
// The legacySanitizerOptions are used to add legacy sanitizers to the list.
// `legacySanitizerOptions` should be a list of strings representing the
// legacy sanitization options (it only support the two original legacy
// options: "NEW_LINES", "SINGLE_QUOTES").
func newSanitizers(specs []SanitizerSpec, legacySanitizerOptions []string) ([]Sanitizer, error) {
	var sanitizers []Sanitizer

	// Add new sanitizers
	for _, spec := range specs {
		sanitizer, err := newSanitizer(spec)
		if err != nil {
			return nil, fmt.Errorf("failed to build sanitizer: %w", err)
		}

		sanitizers = append(sanitizers, sanitizer)
	}

	// Add legacy sanitizers
	for _, opt := range legacySanitizerOptions {
		// legacy sanitizer don't need to be initialized
		switch sanitizationOption(opt) {
		case newLines:
			sanitizers = append(sanitizers, &newLinesSanitizer{})
		case singleQuotes:
			sanitizers = append(sanitizers, &singleQuotesSanitizer{})
		}
	}

	return sanitizers, nil
}

// ----------------------------------------------------------------------------
// New line sanitizer
// ----------------------------------------------------------------------------

// newLinesSanitizer replaces new lines with spaces.
//
// This sanitizer is used to remove new lines inside JSON strings.
type newLinesSanitizer struct{}

func (s *newLinesSanitizer) Sanitize(jsonByte []byte) []byte {
	return bytes.ReplaceAll(jsonByte, []byte("\n"), []byte{})
}

// Init is a no-op for the newLinesSanitizer.
func (s *newLinesSanitizer) Init() error {
	return nil
}

// ----------------------------------------------------------------------------
// Single quote sanitizer
// ----------------------------------------------------------------------------

// singleQuotesSanitizer replaces single quotes with double quotes.
type singleQuotesSanitizer struct{}

func (s *singleQuotesSanitizer) Sanitize(jsonByte []byte) []byte {
	var result bytes.Buffer
	var prevChar byte

	inDoubleQuotes := false

	for _, r := range jsonByte {
		if r == '"' && prevChar != '\\' {
			inDoubleQuotes = !inDoubleQuotes
		}

		if r == '\'' && !inDoubleQuotes {
			result.WriteRune('"')
		} else {
			result.WriteByte(r)
		}
		prevChar = r
	}

	return result.Bytes()
}

// Init is a no-op for the singleQuotesSanitizer.
func (s *singleQuotesSanitizer) Init() error {
	return nil
}

// ----------------------------------------------------------------------------
// Replace all sanitizer
// ----------------------------------------------------------------------------

// replaceAllSanitizer replaces all occurrences of a regex pattern with a replacement string.
type replaceAllSanitizer struct {
	re          *regexp.Regexp
	replacement string
	spec        map[string]interface{}
}

func (s *replaceAllSanitizer) Sanitize(jsonByte []byte) []byte {
	if s.re == nil {
		return jsonByte
	}

	return s.re.ReplaceAll(jsonByte, []byte(s.replacement))
}

func (s *replaceAllSanitizer) Init() error {
	if s.spec == nil {
		return errors.New("missing required sanitizer spec")
	}

	pattern, err := getStringFromSpec(s.spec, "pattern")
	if err != nil {
		return fmt.Errorf("invalid pattern: %w", err)
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("can't compile regex pattern: %w", err)
	}
	s.re = re

	replacement, err := getStringFromSpec(s.spec, "replacement")
	if err != nil {
		return fmt.Errorf("invalid replacement: %w", err)
	}
	s.replacement = replacement

	return nil
}

// getStringFromSpec returns a string from the spec map.
//
// It returns an error if the spec entry key is missing or the value is not a string.
func getStringFromSpec(spec map[string]interface{}, entryKey string) (string, error) {
	value, ok := spec[entryKey]
	if !ok {
		return "", fmt.Errorf("missing sanitizer spec entry: %s", entryKey)
	}

	strValue, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("sanitizer spec entry %s must be a string", entryKey)
	}

	return strValue, nil
}
