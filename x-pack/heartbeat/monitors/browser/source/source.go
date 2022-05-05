// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package source

import (
	"fmt"
)

type Source struct {
	Local      *LocalSource  `config:"local"`
	Inline     *InlineSource `config:"inline" json:"inline"`
	ZipUrl     *ZipURLSource `config:"zip_url" json:"zip_url"`
	ActiveMemo ISource       // cache for selected source
}

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
	}

	return s.ActiveMemo
}

var ErrInvalidSource = fmt.Errorf("no or unknown source type specified for synthetic monitor")

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
