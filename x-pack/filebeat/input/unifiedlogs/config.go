// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build darwin

package unifiedlogs

import (
	"fmt"
	"strings"
	"time"
)

type config struct {
	ShowConfig   showConfig   `config:",inline"`
	CommonConfig commonConfig `config:",inline"`
	Backfill     bool         `config:"backfill"`
}

type showConfig struct {
	ArchiveFile string `config:"archive_file"`
	TraceFile   string `config:"trace_file"`
	Start       string `config:"start"`
	End         string `config:"end"`
}

type commonConfig struct {
	Predicate          []string `config:"predicate"`
	Process            []string `config:"process"`
	Source             bool     `config:"source"`
	Info               bool     `config:"info"`
	Debug              bool     `config:"debug"`
	Backtrace          bool     `config:"backtrace"`
	Signpost           bool     `config:"signpost"`
	Unreliable         bool     `config:"unreliable"`
	MachContinuousTime bool     `config:"mach_continuous_time"`
}

func (c config) Validate() error {
	if err := checkDateFormat(c.ShowConfig.Start); err != nil {
		return fmt.Errorf("start date is not valid: %w", err)
	}
	if err := checkDateFormat(c.ShowConfig.End); err != nil {
		return fmt.Errorf("end date is not valid: %w", err)
	}
	if c.ShowConfig.ArchiveFile != "" && !strings.HasSuffix(c.ShowConfig.ArchiveFile, ".logarchive") {
		return fmt.Errorf("archive_file %v has the wrong extension", c.ShowConfig.ArchiveFile)
	}
	if c.ShowConfig.TraceFile != "" && !strings.HasSuffix(c.ShowConfig.TraceFile, ".tracev3") {
		return fmt.Errorf("trace_file %v has the wrong extension", c.ShowConfig.TraceFile)
	}
	return nil
}

func defaultConfig() config {
	return config{}
}

func checkDateFormat(date string) error {
	if date == "" {
		return nil
	}
	acceptedLayouts := []string{
		"2006-01-02",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04:05-0700",
	}
	for _, layout := range acceptedLayouts {
		if _, err := time.Parse(layout, date); err == nil {
			return nil
		}
	}
	return fmt.Errorf("not a valid date, accepted layouts are: %v", acceptedLayouts)
}
