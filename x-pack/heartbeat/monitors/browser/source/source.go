// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux || darwin

package source

import (
	"fmt"
	"os"
)

type Source struct {
	Local      *LocalSource   `config:"local"`
	Inline     *InlineSource  `config:"inline" json:"inline"`
	ZipUrl     *ZipURLSource  `config:"zip_url" json:"zip_url"`
	Project    *ProjectSource `config:"project" json:"project"`
	ActiveMemo ISource        // cache for selected source
}

var ErrUnsupportedSource = fmt.Errorf("browser monitors are now removed! Please use project monitors instead. See the Elastic synthetics docs at https://www.elastic.co/guide/en/observability/current/synthetic-run-tests.html#synthetic-monitor-choose-project")

func (s *Source) Active() ISource {
	if s.ActiveMemo != nil {
		return s.ActiveMemo
	}

	if s.Local != nil {
		s.ActiveMemo = s.Local
	} else if s.Inline != nil {
		s.ActiveMemo = s.Inline
	} else if s.ZipUrl != nil {
		s.ActiveMemo = s.ZipUrl
	} else if s.Project != nil {
		s.ActiveMemo = s.Project
	}

	return s.ActiveMemo
}

var ErrInvalidSource = fmt.Errorf("no or unknown source type specified for synthetic monitor")

var defaultMod = os.FileMode(0770)

func (s *Source) Validate() error {
	if s.Active() == nil {
		return ErrInvalidSource
	}
	return nil
}

type ISource interface {
	Fetch() error
	Workdir() string
	Close() error
}

type BaseSource struct {
	Type string `config:"type"`
}
