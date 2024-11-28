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
	ArchiveFile string   `config:"archive_file"`
	TraceFile   string   `config:"trace_file"`
	Predicate   []string `config:"predicate"`
	Process     []string `config:"process"`
	Source      bool     `config:"source"`
	Info        bool     `config:"info"`
	Debug       bool     `config:"debug"`
	Signposts   bool     `config:"signposts"`
	Start       string   `config:"start"`
	Timezone    string   `config:"timezone"`
}

func (c config) Validate() error {
	if err := checkDateFormat(c.Start); err != nil {
		return fmt.Errorf("start date is not valid: %w", err)
	}
	if c.ArchiveFile != "" && !strings.HasSuffix(c.ArchiveFile, ".logarchive") {
		return fmt.Errorf("archive_file %v has the wrong extension", c.ArchiveFile)
	}
	if c.TraceFile != "" && !strings.HasSuffix(c.TraceFile, ".tracev3") {
		return fmt.Errorf("trace_file %v has the wrong extension", c.TraceFile)
	}
	return nil
}

func defaultConfig() config {
	return config{
		Timezone: "UTC",
	}
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
