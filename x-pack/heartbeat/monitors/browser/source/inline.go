// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package source

import (
	"fmt"
	"regexp"
)

type InlineSource struct {
	Script string `config:"script"`
	BaseSource
}

func (s *InlineSource) Validate() error {
	if !regexp.MustCompile("\\S").MatchString(s.Script) {
		return fmt.Errorf("no 'script' value specified for inline source")
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
