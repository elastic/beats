package kernel

import (
	"bufio"
	"bytes"
	"strings"

	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"

	"github.com/elastic/go-libaudit/rule"
	"github.com/elastic/go-libaudit/rule/flags"
)

// Config defines the kernel metricset's possible configuration options.
type Config struct {
	ResolveIDs   bool   `config:"kernel.resolve_ids"`         // Resolve UID/GIDs to names.
	FailureMode  string `config:"kernel.failure_mode"`        // Failure mode for the kernel (silent, log, panic).
	BacklogLimit uint32 `config:"kernel.backlog_limit"`       // Max number of message to buffer in the kernel.
	RateLimit    uint32 `config:"kernel.rate_limit"`          // Rate limit in messages/sec of messages from kernel.
	RawMessage   bool   `config:"kernel.include_raw_message"` // Include the list of raw audit messages in the event.
	Warnings     bool   `config:"kernel.include_warnings"`    // Include warnings in the event (for dev/debug purposes only).
	RulesBlob    string `config:"kernel.audit_rules"`         // Audit rules. One rule per line.
}

type auditRule struct {
	flags string
	data  []byte
}

// Validate validates the rules specified in the config.
func (c Config) Validate() error {
	var errs multierror.Errors
	_, err := c.rules()
	if err != nil {
		errs = append(errs, err)
	}
	_, err = c.failureMode()
	if err != nil {
		errs = append(errs, err)
	}
	return errs.Err()
}

// Rules returns a list of rules specified in the config.
func (c Config) rules() ([]auditRule, error) {
	var errs multierror.Errors
	var auditRules []auditRule
	ruleSet := map[string]auditRule{}
	s := bufio.NewScanner(bytes.NewBufferString(c.RulesBlob))
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		// Parse the CLI flags into an intermediate rule specification.
		r, err := flags.Parse(line)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "failed on rule '%v'", line))
			continue
		}

		// Convert rule specification to a binary rule representation.
		data, err := rule.Build(r)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "failed on rule '%v'", line))
			continue
		}

		// Detect duplicates based on the normalized binary rule representation.
		existingRule, found := ruleSet[string(data)]
		if found {
			errs = append(errs, errors.Errorf("failed on rule '%v' because its a duplicate of '%v'", line, existingRule.flags))
			continue
		}
		auditRule := auditRule{flags: line, data: []byte(data)}
		ruleSet[string(data)] = auditRule

		auditRules = append(auditRules, auditRule)
	}

	if len(errs) > 0 {
		return nil, errors.Wrap(errs.Err(), "invalid kernel.audit_rules")
	}
	return auditRules, nil
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
		return 0, errors.Errorf("invalid kernel.failure_mode '%v' (use silent, log, or panic)", c.FailureMode)
	}
}

var defaultConfig = Config{
	ResolveIDs:   true,
	FailureMode:  "silent",
	BacklogLimit: 8192,
	RateLimit:    0,
	RawMessage:   false,
	Warnings:     false,
}
