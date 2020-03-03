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
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/elastic/go-libaudit"
	"github.com/elastic/go-libaudit/rule"

	"github.com/elastic/beats/v7/auditbeat/cmd"
)

var (
	dontResolveIDs   bool
	noOutputIfEmpty  bool
	singleLineStatus bool
)

func init() {
	showRules := cobra.Command{
		Use:     "auditd-rules",
		Short:   "Show currently installed auditd rules",
		Aliases: []string{"audit-rules", "audit_rules", "rules", "auditdrules", "auditrules"},
		Run: func(cmd *cobra.Command, args []string) {
			if err := showAuditdRules(); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to show auditd rules: %v\n", err)
				os.Exit(1)
			}
		},
	}
	showRules.Flags().BoolVarP(&dontResolveIDs, "no-resolve", "n", false, "Don't resolve numeric IDs (UIDs, GIDs and file_type fields)")
	showRules.Flags().BoolVarP(&noOutputIfEmpty, "no-output", "z", false, "Don't generate output when the rule list is empty")

	showStatus := cobra.Command{
		Use:     "auditd-status",
		Short:   "Show kernel auditd status",
		Aliases: []string{"audit-status", "audit_status", "status", "auditdstatus", "auditrules"},
		Run: func(cmd *cobra.Command, args []string) {
			if err := showAuditdStatus(); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to show auditd rules: %v\n", err)
				os.Exit(1)
			}
		},
	}
	showStatus.Flags().BoolVarP(&singleLineStatus, "single-line", "s", false, "Output status as a single line")
	cmd.ShowCmd.AddCommand(&showRules, &showStatus)
}

func showAuditdRules() error {
	client, err := libaudit.NewAuditClient(nil)
	if err != nil {
		return errors.Wrap(err, "failed to create audit client")
	}
	defer client.Close()

	rules, err := client.GetRules()
	if err != nil {
		return errors.Wrap(err, "failed to list existing rules")
	}

	for idx, raw := range rules {
		r, err := rule.ToCommandLine(raw, !dontResolveIDs)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error decoding rule %d: %v\n", idx, err)
			fmt.Fprintf(os.Stderr, "Raw dump: <<<%v>>>\n", raw)
		}
		fmt.Println(r)
	}

	if !noOutputIfEmpty && len(rules) == 0 {
		fmt.Println("No rules")
	}
	return nil
}

func showAuditdStatus() error {
	client, err := libaudit.NewAuditClient(nil)
	if err != nil {
		return errors.Wrap(err, "failed to create audit client")
	}
	defer client.Close()

	status, err := client.GetStatus()
	if err != nil {
		return errors.Wrap(err, "failed to get audit status")
	}

	if status.FeatureBitmap == libaudit.AuditFeatureBitmapBacklogWaitTime {
		// If FeatureBitmap value is "2", means we're running under an old kernel
		// in which FeatureBitmap meant Version. Version 2 supports both
		// backlog_wait_time and backlog_limit.
		status.FeatureBitmap |= libaudit.AuditFeatureBitmapBacklogLimit
	}
	separator := '\n'
	if singleLineStatus {
		separator = ' '
	}

	fmt.Printf("enabled %d%c"+
		"failure %d%c"+
		"pid %d%c"+
		"rate_limit %d%c"+
		"backlog_limit %d%c"+
		"lost %d%c"+
		"backlog %d%c"+
		"backlog_wait_time %d%c"+
		"features %s\n",
		status.Enabled, separator,
		status.Failure, separator,
		status.PID, separator,
		status.RateLimit, separator,
		status.BacklogLimit, separator,
		status.Lost, separator,
		status.Backlog, separator,
		status.BacklogWaitTime, separator,
		fmt.Sprintf("%#x", status.FeatureBitmap))

	return nil
}
