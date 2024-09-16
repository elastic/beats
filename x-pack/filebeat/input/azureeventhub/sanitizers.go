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
	case "new_lines":
		s = &newLinesSanitizer{}
	case "single_quotes":
		s = &singleQuotesSanitizer{}
	case "replace_all":
		s = &replaceAllSanitizer{spec: spec.Spec}
	default:
		return nil, fmt.Errorf("unknown sanitizer type: %s", spec.Type)
	}

	// Initialize the sanitizer with the provided spec.
	err := s.Init()
	if err != nil {
		return nil, err
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
		return errors.New("missing sanitizer spec")
	}

	if _, ok := s.spec["pattern"]; !ok {
		return errors.New("missing regex pattern")
	}

	if _, ok := s.spec["pattern"].(string); !ok {
		return errors.New("regex pattern must be a string")
	}

	re, err := regexp.Compile(s.spec["pattern"].(string))
	if err != nil {
		return err
	}

	s.re = re

	if _, ok := s.spec["replacement"]; !ok {
		return errors.New("missing replacement format")
	}

	if _, ok := s.spec["replacement"].(string); !ok {
		return errors.New("replacement format must be a string")
	}

	s.replacement = s.spec["replacement"].(string)

	return nil
}
