// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package remote_write

import (
	"errors"
	"time"

	rw "github.com/elastic/beats/v7/metricbeat/module/prometheus/remote_write"
)

type config struct {
	MetricsCount           bool              `config:"metrics_count"`
	UseTypes               bool              `config:"use_types"`
	RateCounters           bool              `config:"rate_counters"`
	TypesPatterns          TypesPatterns     `config:"types_patterns" yaml:"types_patterns,omitempty"`
	HistogramAssembly      HistogramAssembly `config:"histogram_assembly"`
	Period                 time.Duration     `config:"period"     validate:"positive"`
	MaxCompressedBodyBytes int64             `config:"max_compressed_body_bytes"`
	MaxDecodedBodyBytes    int64             `config:"max_decoded_body_bytes"`
}

// HistogramAssembly configures cross-request histogram bucket assembly.
type HistogramAssembly struct {
	Enabled              bool          `config:"enabled"`
	QuietPeriod          time.Duration `config:"quiet_period"`
	HardTimeout          time.Duration `config:"hard_timeout"`
	MaxPendingHistograms int           `config:"max_pending_histograms"`
	MaxPendingBuckets    int           `config:"max_pending_buckets"`
}

type TypesPatterns struct {
	CounterPatterns   *[]string `config:"counter_patterns" yaml:"include,omitempty"`
	HistogramPatterns *[]string `config:"histogram_patterns" yaml:"exclude,omitempty"`
}

var defaultConfig = config{
	TypesPatterns: TypesPatterns{
		CounterPatterns:   nil,
		HistogramPatterns: nil},
	Period:                 time.Second * 60,
	MaxCompressedBodyBytes: rw.DefaultMaxCompressedBodyBytes,
	MaxDecodedBodyBytes:    rw.DefaultMaxDecodedBodyBytes,
}

func (c *config) Validate() error {
	if c.RateCounters && !c.UseTypes {
		return errors.New("'rate_counters' can only be enabled when `use_types` is also enabled")
	}
	if c.HistogramAssembly.Enabled && !c.UseTypes {
		return errors.New("'histogram_assembly.enabled' can only be enabled when `use_types` is also enabled")
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
	if c.HistogramAssembly.Enabled {
		if err := c.HistogramAssembly.validateAndApplyDefaults(); err != nil {
			return err
		}
	}
	return nil
}

func (h *HistogramAssembly) validateAndApplyDefaults() error {
	if h.QuietPeriod == 0 {
		h.QuietPeriod = 5 * time.Second
	}
	if h.HardTimeout == 0 {
		h.HardTimeout = 30 * time.Second
	}
	if h.MaxPendingHistograms == 0 {
		h.MaxPendingHistograms = 10_000
	}
	if h.MaxPendingBuckets == 0 {
		h.MaxPendingBuckets = 100_000
	}
	if h.QuietPeriod <= 0 {
		return errors.New("histogram_assembly.quiet_period must be greater than 0")
	}
	if h.HardTimeout <= 0 {
		return errors.New("histogram_assembly.hard_timeout must be greater than 0")
	}
	if h.QuietPeriod > h.HardTimeout {
		return errors.New("histogram_assembly.quiet_period must be less than or equal to hard_timeout")
	}
	if h.MaxPendingHistograms <= 0 {
		return errors.New("histogram_assembly.max_pending_histograms must be greater than 0")
	}
	if h.MaxPendingBuckets <= 0 {
		return errors.New("histogram_assembly.max_pending_buckets must be greater than 0")
	}
	return nil
}

func (h HistogramAssembly) assemblyConfig() histogramAssemblyConfig {
	return histogramAssemblyConfig{
		QuietPeriod:          h.QuietPeriod,
		HardTimeout:          h.HardTimeout,
		MaxPendingHistograms: h.MaxPendingHistograms,
		MaxPendingBuckets:    h.MaxPendingBuckets,
	}
}
