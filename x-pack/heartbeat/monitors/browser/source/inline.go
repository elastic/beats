// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.
//go:build linux || darwin || synthetics

package source

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
)

type InlineSource struct {
	Script string `config:"script"`
	BaseSource
}

var ErrNoInlineScript = fmt.Errorf("no 'script' value specified for inline source")

func (s *InlineSource) Validate() error {
	if !regexp.MustCompile(`\S`).MatchString(s.Script) {
		return ErrNoInlineScript
	}

	return nil
}

func (s *InlineSource) Fetch() (err error) {
	// "step(" is a good indicator that the script is already decoded since `(` is
	// not a valid base64 character. This ensures backwards compatibility with
	// older inline scripts that are not encoded.
	if strings.Contains(s.Script, "step(") {
		return nil
	}
	// decode the base64 encoded script and replace the original script
	decodedBytes, err := base64.StdEncoding.DecodeString(s.Script)
	if err != nil {
		return fmt.Errorf("could not decode inline script: %w", err)
	}
	s.Script = string(decodedBytes)
	return nil
}

func (s *InlineSource) Workdir() string {
	return ""
}

func (s *InlineSource) Close() error {
	return nil
}
