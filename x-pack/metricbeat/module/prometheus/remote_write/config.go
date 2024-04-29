// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package remote_write

import (
	"errors"
	"time"
)

type config struct {
	UseTypes      bool          `config:"use_types"`
	RateCounters  bool          `config:"rate_counters"`
	TypesPatterns TypesPatterns `config:"types_patterns" yaml:"types_patterns,omitempty"`
	Period        time.Duration `config:"period"     validate:"positive"`
}

type TypesPatterns struct {
	CounterPatterns   *[]string `config:"counter_patterns" yaml:"include,omitempty"`
	HistogramPatterns *[]string `config:"histogram_patterns" yaml:"exclude,omitempty"`
}

var defaultConfig = config{
	TypesPatterns: TypesPatterns{
		CounterPatterns:   nil,
		HistogramPatterns: nil},
	Period: time.Second * 60,
}

func (c *config) Validate() error {
	if c.RateCounters && !c.UseTypes {
		return errors.New("'rate_counters' can only be enabled when `use_types` is also enabled")
	}
	duration, err := time.ParseDuration(c.Period.String())
	{
		if err != nil {
			return err
		} else if duration < 60*time.Second {
			// by default prometheus push data with the interval 60s, in order to calculate counter rate we are setting Period to 60secs accordingly
			c.Period = time.Second * 60
		}
	}
	return nil
}
