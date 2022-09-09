// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package auditd

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/joeshaw/multierror"

	"github.com/elastic/go-libaudit/v2/rule"
	"github.com/elastic/go-libaudit/v2/rule/flags"
)

// Config defines the kernel metricset's possible configuration options.
type Config struct {
	ResolveIDs   bool     `config:"resolve_ids"`         // Resolve UID/GIDs to names.
	FailureMode  string   `config:"failure_mode"`        // Failure mode for the kernel (silent, log, panic).
	BacklogLimit uint32   `config:"backlog_limit"`       // Max number of message to buffer in the auditd.
	RateLimit    uint32   `config:"rate_limit"`          // Rate limit in messages/sec of messages from auditd.
	RawMessage   bool     `config:"include_raw_message"` // Include the list of raw audit messages in the event.
	Warnings     bool     `config:"include_warnings"`    // Include warnings in the event (for dev/debug purposes only).
	RulesBlob    string   `config:"audit_rules"`         // Audit rules. One rule per line.
	RuleFiles    []string `config:"audit_rule_files"`    // List of rule files.
	SocketType   string   `config:"socket_type"`         // Socket type to use with the kernel (unicast or multicast).

	// Tuning options (advanced, use with care)
	ReassemblerMaxInFlight uint32        `config:"reassembler.max_in_flight"`
	ReassemblerTimeout     time.Duration `config:"reassembler.timeout"`
	StreamBufferQueueSize  uint32        `config:"reassembler.queue_size"`
	// BackpressureStrategy defines the strategy used to mitigate backpressure
	// propagating to the kernel causing audited processes to block until
	// Auditbeat can keep-up.
	// One of "user-space", "kernel", "both", "none", "auto" (default)
	BackpressureStrategy  string `config:"backpressure_strategy"`
	StreamBufferConsumers int    `config:"stream_buffer_consumers"`

	auditRules []auditRule
}

type auditRule struct {
	flags string
	data  []byte
}

type ruleWithSource struct {
	rule   auditRule
	source string
}

type ruleSet map[string]ruleWithSource

var defaultConfig = Config{
	ResolveIDs:             true,
	FailureMode:            "silent",
	BacklogLimit:           8192,
	RateLimit:              0,
	RawMessage:             false,
	Warnings:               false,
	ReassemblerMaxInFlight: 50,
	ReassemblerTimeout:     2 * time.Second,
	StreamBufferQueueSize:  8192,
	StreamBufferConsumers:  0,
}

// Validate validates the rules specified in the config.
func (c *Config) Validate() error {
	var errs multierror.Errors
	err := c.loadRules()
	if err != nil {
		errs = append(errs, err)
	}
	_, err = c.failureMode()
	if err != nil {
		errs = append(errs, err)
	}

	c.SocketType = strings.ToLower(c.SocketType)
	switch c.SocketType {
	case "", "unicast", "multicast":
	default:
		errs = append(errs, fmt.Errorf("invalid socket_type "+
			"'%v' (use unicast, multicast, or don't set a value)", c.SocketType))
	}

	return errs.Err()
}

// Rules returns a list of rules specified in the config.
func (c Config) rules() []auditRule {
	return c.auditRules
}

func (c *Config) loadRules() error {
	var paths []string
	for _, pattern := range c.RuleFiles {
		absPattern, err := filepath.Abs(pattern)
		if err != nil {
			return fmt.Errorf("unable to get the absolute path for %s: %v", pattern, err)
		}
		files, err := filepath.Glob(absPattern)
		if err != nil {
			return err
		}
		sort.Strings(files)
		paths = append(paths, files...)
	}

	knownRules := ruleSet{}

	rules, err := readRules(bytes.NewBufferString(c.RulesBlob), "(audit_rules at auditbeat.yml)", knownRules)
	if err != nil {
		return err
	}
	c.auditRules = append(c.auditRules, rules...)

	for _, filename := range paths {
		fHandle, err := os.Open(filename)
		if err != nil {
			return fmt.Errorf("unable to open rule file '%s': %v", filename, err)
		}
		rules, err = readRules(fHandle, filename, knownRules)
		if err != nil {
			return err
		}
		c.auditRules = append(c.auditRules, rules...)
	}

	return nil
}

func (c Config) failureMode() (uint32, error) {
	switch strings.ToLower(c.FailureMode) {
	case "silent":
		return 0, nil
	case "log":
		return 1, nil
	case "panic":
		return 2, nil
	default:
		return 0, fmt.Errorf("invalid failure_mode '%v' (use silent, log, or panic)", c.FailureMode)
	}
}

func readRules(reader io.Reader, source string, knownRules ruleSet) (rules []auditRule, err error) {
	var errs multierror.Errors

	s := bufio.NewScanner(reader)
	for lineNum := 1; s.Scan(); lineNum++ {
		location := fmt.Sprintf("%s:%d", source, lineNum)
		line := strings.TrimSpace(s.Text())
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		// Parse the CLI flags into an intermediate rule specification.
		r, err := flags.Parse(line)
		if err != nil {
			errs = append(errs, fmt.Errorf("at %s: failed to parse rule '%v': %w", location, line, err))
			continue
		}

		// Convert rule specification to a binary rule representation.
		data, err := rule.Build(r)
		if err != nil {
			errs = append(errs, fmt.Errorf("at %s: failed to interpret rule '%v': %w", location, line, err))
			continue
		}

		// Detect duplicates based on the normalized binary rule representation.
		existing, found := knownRules[string(data)]
		if found {
			errs = append(errs, fmt.Errorf("at %s: rule '%v' is a duplicate of '%v' at %s", location, line, existing.rule.flags, existing.source))
			continue
		}
		rule := auditRule{flags: line, data: []byte(data)}
		knownRules[string(data)] = ruleWithSource{rule, location}

		rules = append(rules, rule)
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("failed loading rules: %w", errs.Err())
	}
	return rules, nil
}
