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

//go:build unix

package auditd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

func TestConfig(t *testing.T) {
	logp.TestingSetup()

	t.Run("Validate", func(t *testing.T) {
		data := `
audit_rules: |
  # Comments and empty lines are ignored.
  -w /etc/passwd -p wa -k auth

  -a always,exit -S execve -k exec`

		config, err := parseConfig(t, data)
		if err != nil {
			t.Fatal(err)
		}
		rules := config.rules()

		assert.EqualValues(t, []string{
			"-w /etc/passwd -p wa -k auth",
			"-a always,exit -S execve -k exec",
		}, commands(rules))
	})

	t.Run("ValidateWithError", func(t *testing.T) {
		data := `
audit_rules: |
  -x bad -F flag
  -a always,exit -w /etc/passwd
  -a always,exit -S fake -k exec`

		_, err := parseConfig(t, data)
		if err == nil {
			t.Fatal("expected error")
		}
		t.Log(err)
	})

	t.Run("ValidateWithErrorIgnored", func(t *testing.T) {
		data := `
ignore_errors: true
audit_rules: |
  -x bad -F flag
  -a always,exit -w /etc/passwd
  -a always,exit -S fake -k exec
  -w /etc/passwd -k auth`

		cfg, err := parseConfig(t, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(cfg.auditRules) != 1 {
			t.Fatalf("unexpected number of rules from parseConfig: got %d, want %d", len(cfg.auditRules), 1)
		}
	})

	t.Run("ValidateWithDuplicates", func(t *testing.T) {
		data := `
audit_rules: |
  -w /etc/passwd -p rwxa -k auth
  -w /etc/passwd -k auth`

		_, err := parseConfig(t, data)
		if err == nil {
			t.Fatal("expected error")
		}
		t.Log(err)
	})

	t.Run("ValidateFailureMode", func(t *testing.T) {
		config := defaultConfig
		config.FailureMode = "boom"
		err := config.Validate()
		assert.Error(t, err)
		t.Log(err)
	})

	t.Run("ValidateConnectionType", func(t *testing.T) {
		config := defaultConfig
		config.SocketType = "Satellite"
		err := config.Validate()
		assert.Error(t, err)
		t.Log(err)
	})

	t.Run("ValidateImmutable", func(t *testing.T) {
		tcs := []struct {
			name       string
			socketType string
			mustFail   bool
		}{
			{
				name:       "Must pass for default",
				socketType: "",
				mustFail:   false,
			},
			{
				name:       "Must pass for unicast",
				socketType: "unicast",
				mustFail:   false,
			},
			{
				name:       "Must fail for multicast",
				socketType: "multicast",
				mustFail:   true,
			},
		}

		for _, tc := range tcs {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				config := defaultConfig
				config.SocketType = tc.socketType
				config.Immutable = true
				err := config.Validate()
				if tc.mustFail {
					assert.Error(t, err)
					t.Log(err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})

	t.Run("RuleOrdering", func(t *testing.T) {
		const fileMode = 0o644
		config := defaultConfig
		config.RulesBlob = strings.Join([]string{
			makeRuleFlags(0, 0),
			makeRuleFlags(0, 1),
			makeRuleFlags(0, 2),
		}, "\n")

		dir1 := t.TempDir()

		for _, file := range []struct {
			order int
			name  string
		}{
			{0, "00_first.conf"},
			{5, "99_last.conf"},
			{2, "03_auth.conf"},
			{4, "20_exec.conf"},
			{3, "10_network_access.conf"},
			{1, "01_32bit_abi.conf"},
		} {
			path := filepath.Join(dir1, file.name)
			content := []byte(strings.Join([]string{
				makeRuleFlags(1+file.order, 0),
				makeRuleFlags(1+file.order, 1),
				makeRuleFlags(1+file.order, 2),
				makeRuleFlags(1+file.order, 3),
			}, "\n"))
			if err := os.WriteFile(path, content, fileMode); err != nil {
				t.Fatal(err)
			}
		}

		dir2 := t.TempDir()

		for _, file := range []struct {
			order int
			name  string
		}{
			{3, "99_tail.conf"},
			{0, "00_head.conf"},
			{2, "50_mid.conf"},
			{1, "13.conf"},
		} {
			path := filepath.Join(dir2, file.name)
			content := []byte(strings.Join([]string{
				makeRuleFlags(10+file.order, 0),
				makeRuleFlags(10+file.order, 1),
				makeRuleFlags(10+file.order, 2),
				makeRuleFlags(10+file.order, 3),
			}, "\n"))
			if err := os.WriteFile(path, content, fileMode); err != nil {
				t.Fatal(err)
			}
		}

		config.RuleFiles = []string{
			fmt.Sprintf("%s/*.conf", dir1),
			fmt.Sprintf("%s/*.conf", dir2),
		}

		if err := config.Validate(); err != nil {
			t.Fatal(err)
		}

		rules := config.rules()
		fileNo, ruleNo := 0, 0
		for _, rule := range rules {
			parts := strings.Split(rule.flags, " ")
			assert.Len(t, parts, 6, rule.flags)
			fields := strings.Split(parts[5], ":")
			assert.Len(t, fields, 3, rule.flags)
			fileID, err := strconv.Atoi(fields[1])
			if err != nil {
				t.Fatal(err, rule.flags)
			}
			ruleID, err := strconv.Atoi(fields[2])
			if err != nil {
				t.Fatal(err, rule.flags)
			}
			if fileID > fileNo {
				fileNo = fileID
				ruleNo = 0
			}
			assert.Equal(t, fileNo, fileID, rule.flags)
			assert.Equal(t, ruleNo, ruleID, rule.flags)
			ruleNo++
		}
	})
}

func makeRuleFlags(fileID, ruleID int) string {
	return fmt.Sprintf("-w /path/%d/%d -p rwxa -k rule:%d:%d", fileID, ruleID, fileID, ruleID)
}

func parseConfig(t testing.TB, yaml string) (Config, error) {
	c, err := conf.NewConfigWithYAML([]byte(yaml), "")
	if err != nil {
		t.Fatal(err)
	}

	config := defaultConfig
	err = c.Unpack(&config)
	return config, err
}

func commands(rules []auditRule) []string {
	var cmds []string //nolint:prealloc // Preallocating doesn't bring improvements.
	for _, r := range rules {
		cmds = append(cmds, r.flags)
	}
	return cmds
}
