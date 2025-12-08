// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.
//go:build linux || darwin || synthetics

package source

import (
	"encoding/base64"
	"fmt"
	"regexp"
)

type InlineSource struct {
	Script   string `config:"script"`
	Encoding string `config:"encoding"`
	BaseSource
}

var ErrNoInlineScript = fmt.Errorf("no 'script' value specified for inline source")

func (s *InlineSource) Validate() error {
	if !regexp.MustCompile(`\S`).MatchString(s.Script) {
		return ErrNoInlineScript
	}

	if s.Encoding != "" && s.Encoding != "base64" {
		return fmt.Errorf("unsupported encoding: %v", s.Encoding)
	}

	return nil
}

func (s *InlineSource) Fetch() (err error) {
	return nil
}

func (s *InlineSource) Workdir() string {
	return ""
}

func (s *InlineSource) Close() error {
	return nil
}

func (s *InlineSource) Decode() error {
	// Don't decode if flag is missing
	if s.Encoding != "base64" {
		return nil
	}

	decoded, err := base64.StdEncoding.DecodeString(s.Script)
	if err != nil {
		return fmt.Errorf("error decoding from base64: %w", err)
	}

	s.Script = string(decoded)

	return nil
}
