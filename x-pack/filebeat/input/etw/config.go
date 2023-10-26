// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package etw_input

import (
	"fmt"

	"github.com/elastic/beats/v7/x-pack/libbeat/reader/etw"
)

var validTraceLevel = map[string]bool{
	"critical":    true,
	"error":       true,
	"warning":     true,
	"information": true,
	"verbose":     true,
}

type config struct {
	Logfile         string `config:"file"`
	ProviderGUID    string `config:"provider.guid"`
	ProviderName    string `config:"provider.name"`
	SessionName     string `config:"session_name"` // Tag for the new session
	TraceLevel      string `config:"trace_level"`
	MatchAnyKeyword uint64 `config:"match_any_keyword"`
	MatchAllKeyword uint64 `config:"match_all_keyword"`
	Session         string `config:"session"`
}

// Create a conversion function to convert config to etw.Config
func convertConfig(cfg config) etw.Config {
	return etw.Config{
		Logfile:         cfg.Logfile,
		ProviderGUID:    cfg.ProviderGUID,
		ProviderName:    cfg.ProviderName,
		SessionName:     cfg.SessionName,
		TraceLevel:      cfg.TraceLevel,
		MatchAnyKeyword: cfg.MatchAnyKeyword,
		MatchAllKeyword: cfg.MatchAllKeyword,
		Session:         cfg.Session,
	}
}

func defaultConfig() config {
	return config{
		Logfile:         "",
		ProviderName:    "",
		ProviderGUID:    "",
		SessionName:     "",
		TraceLevel:      "verbose",
		MatchAnyKeyword: 0xffffffffffffffff,
		MatchAllKeyword: 0,
		Session:         "",
	}
}

// Config validation
func (c *config) validate() error {
	if c.ProviderName == "" && c.ProviderGUID == "" && c.Logfile == "" && c.Session == "" {
		return fmt.Errorf("provider, existing logfile or running session must be set")
	}

	if !validTraceLevel[c.TraceLevel] {
		return fmt.Errorf("invalid Trace Level value '%s'", c.TraceLevel)
	}

	if c.ProviderGUID != "" {
		if c.ProviderName != "" {
			return fmt.Errorf("configuration constraint error: provider GUID and provider name cannot be defined together")
		}
		if c.Logfile != "" {
			return fmt.Errorf("configuration constraint error: provider GUID and file cannot be defined together")
		}
		if c.Session != "" {
			return fmt.Errorf("configuration constraint error: provider GUID and existing session cannot be defined together")
		}
	}

	if c.ProviderName != "" {
		if c.Logfile != "" {
			return fmt.Errorf("configuration constraint error: provider name and file cannot be defined together")
		}
		if c.Session != "" {
			return fmt.Errorf("configuration constraint error: provider name and existing session cannot be defined together")
		}
	}

	if c.Logfile != "" {
		if c.Session != "" {
			return fmt.Errorf("configuration constraint error: file and existing session cannot be defined together")
		}
	}

	return nil
}
