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

package common

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	devtools "github.com/elastic/beats/v7/dev-tools/mage"
	"gopkg.in/yaml.v2"
)

// RenamedFields prints AsciiDoc tables of renamed ECS fields for each beat.
// Replaces script/renamed_fields.py.
// Reads ecs-migration-8x.yml from the script/ directory in the beats root.
func RenamedFields() error {
	beatsDir, err := devtools.ElasticBeatsDir()
	if err != nil {
		return err
	}
	migrationYML := filepath.Join(beatsDir, "script", "ecs-migration-8x.yml")

	beats := []string{"Auditbeat", "Filebeat", "Heartbeat", "Metricbeat", "Packetbeat", "Winlogbeat"}

	data, err := os.ReadFile(migrationYML)
	if err != nil {
		return fmt.Errorf("reading migration file: %w", err)
	}

	// ecsMigrationEntry is defined in kibana_migration.go (same package).
	var entries []ecsMigrationEntry
	if err := yaml.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("parsing migration YAML: %w", err)
	}

	for _, beat := range beats {
		fmt.Printf(".%s renamed fields in 7.0\n", beat)
		fields := readRenamedFieldsForBeat(entries, strings.ToLower(beat))
		fmt.Print(renamedFieldsTable(fields))
	}
	return nil
}

type renamedFieldPair struct {
	from string
	to   string
}

func readRenamedFieldsForBeat(entries []ecsMigrationEntry, beat string) []renamedFieldPair {
	fieldMap := make(map[string]string)
	for _, e := range entries {
		if e.From == "" || e.To == nil {
			continue
		}
		toString, ok := e.To.(string)
		if !ok {
			continue
		}
		if e.Beat != "" && e.Beat != beat {
			continue
		}
		fieldMap[e.From] = toString
	}

	var pairs []renamedFieldPair
	for from, to := range fieldMap {
		pairs = append(pairs, renamedFieldPair{from: from, to: to})
	}
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].from < pairs[j].from
	})
	return pairs
}

func renamedFieldsTable(fields []renamedFieldPair) string {
	var b strings.Builder
	b.WriteString("[frame=\"topbot\",options=\"header\"]\n")
	b.WriteString("|======================\n")
	b.WriteString("|Old Field|New Field\n")
	for _, f := range fields {
		fmt.Fprintf(&b, "|`%s`            |`%s`\n", f.from, f.to)
	}
	b.WriteString("|======================\n")
	return b.String()
}
