// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package etw

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
	// Logfile is the path to an .etl file to read from.
	Logfile string `config:"file"`
	// ProviderGUID is the GUID of an ETW provider.
	// Run 'logman query providers' to list the available providers.
	ProviderGUID string `config:"provider.guid"`
	// ProviderName is the name of an ETW provider.
	// Run 'logman query providers' to list the available providers.
	ProviderName string `config:"provider.name"`
	// SessionName is the name used to create a new session for the
	// defined provider. If missing, its default value is the provider ID
	// prefixed by 'Elastic-'
	SessionName string `config:"session_name"`
	// TraceLevel filters all provider events with a level value
	// that is less than or equal to this level.
	// Allowed values are critical, error, warning, information, and verbose.
	TraceLevel string `config:"trace_level"`
	// MatchAnyKeyword is an 8-byte bitmask that enables the filtering of
	// events from specific provider subcomponents. The provider will write
	// a particular event if the event's keyword bits match any of the bits
	// in this bitmask.
	// See https://learn.microsoft.com/en-us/message-analyzer/system-etw-provider-event-keyword-level-settings for more details.
	// Use logman query providers "<provider.name>" to list the available keywords.
	MatchAnyKeyword uint64 `config:"match_any_keyword"`
	// An 8-byte bitmask that enables the filtering of events from
	// specific provider subcomponents. The provider will write a particular
	// event if the event's keyword bits match all of the bits in this bitmask.
	// See https://learn.microsoft.com/en-us/message-analyzer/system-etw-provider-event-keyword-level-settings for more details.
	MatchAllKeyword uint64 `config:"match_all_keyword"`
	// Session is the name of an existing session to read from.
	// Run 'logman query -ets' to list existing sessions.
	Session string `config:"session"`
}

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
		TraceLevel:      "verbose",
		MatchAnyKeyword: 0xffffffffffffffff,
	}
}

func (c *config) Validate() error {
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
