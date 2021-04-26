// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package browser

import (
	"context"
	"fmt"

	"github.com/elastic/beats/v7/libbeat/common"
)

type JourneyLister func(ctx context.Context, suitePath string, params common.MapStr) (journeyNames []string, err error)

var journeyListSingleton JourneyLister

type SyntheticSuite struct {
	rawCfg   *common.Config
	suiteCfg *Config
}

func NewSuite(rawCfg *common.Config) (*SyntheticSuite, error) {
	ss := &SyntheticSuite{
		rawCfg:   rawCfg,
		suiteCfg: DefaultConfig(),
	}
	err := rawCfg.Unpack(ss.suiteCfg)
	if err != nil {
		return nil, ErrBadConfig(err)
	}

	return ss, nil
}

func ErrBadConfig(err error) error {
	return fmt.Errorf("could not parse suite config: %w", err)
}

func (s *SyntheticSuite) String() string {
	panic("implement me")
}

func (s *SyntheticSuite) Fetch() error {
	return s.suiteCfg.Source.Active().Fetch()
}

func (s *SyntheticSuite) Workdir() string {
	return s.suiteCfg.Source.Active().Workdir()
}

func (s *SyntheticSuite) InlineSource() (string, bool) {
	if s.suiteCfg.Source.Inline != nil {
		return s.suiteCfg.Source.Inline.Script, true
	}
	return "", false
}

func (s *SyntheticSuite) Params() map[string]interface{} {
	return s.suiteCfg.Params
}

func (s *SyntheticSuite) Close() error {
	if s.suiteCfg.Source.ActiveMemo != nil {
		s.suiteCfg.Source.ActiveMemo.Close()
	}

	return nil
}
