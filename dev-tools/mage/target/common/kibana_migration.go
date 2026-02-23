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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	devtools "github.com/elastic/beats/v7/dev-tools/mage"
	"gopkg.in/yaml.v2"
)

type ecsMigrationEntry struct {
	From   string      `yaml:"from"`
	To     interface{} `yaml:"to"`
	Alias  interface{} `yaml:"alias"`
	Rename interface{} `yaml:"rename"`
	Beat   string      `yaml:"beat"`
}

type kibanaObject struct {
	ID         string `json:"id"`
	Attributes struct {
		Title    string `json:"title"`
		VisState struct {
			Title string `json:"title"`
		} `json:"visState"`
	} `json:"attributes"`
}

type kibanaFile struct {
	Objects []kibanaObject `json:"objects"`
}

// KibanaMigration migrates field names in Kibana dashboard JSON files to ECS.
// Replaces script/kibana-migration.py.
// Set APPEND_ECS=true to also append "-ecs" to IDs and " ECS" to titles.
// Reads ecs-migration-8x.yml from the script/ directory in the beats root.
func KibanaMigration() error {
	beatsDir, err := devtools.ElasticBeatsDir()
	if err != nil {
		return err
	}
	migrationYML := filepath.Join(beatsDir, "script", "ecs-migration-8x.yml")
	appendECS := os.Getenv("APPEND_ECS") == "true"

	fmt.Println("Start Kibana files migration")

	fmt.Println("Migrate all fields to the ECS fields")
	migrationFields, err := readMigrationFields(migrationYML)
	if err != nil {
		return err
	}
	files := getKibanaFiles(beatsDir)
	if err := renameKibanaEntries(files, migrationFields); err != nil {
		return err
	}

	if appendECS {
		fmt.Println("Postfix all ids with -ecs")
		ids := getReplaceableIDs(files)
		if err := renameKibanaEntries(files, ids); err != nil {
			return err
		}

		fmt.Println("Postfix all titles with ` ECS`")
		titles := getReplaceableTitles(files)
		if err := renameKibanaEntries(files, titles); err != nil {
			return err
		}
	}
	return nil
}

func readMigrationFields(migrationYML string) (map[string]string, error) {
	data, err := os.ReadFile(migrationYML)
	if err != nil {
		return nil, fmt.Errorf("reading migration file: %w", err)
	}

	var entries []ecsMigrationEntry
	if err := yaml.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("parsing migration YAML: %w", err)
	}

	fields := make(map[string]string)
	for _, e := range entries {
		if e.From == "" || e.To == nil {
			continue
		}
		toString, ok := e.To.(string)
		if !ok {
			continue
		}
		if rename, ok := e.Rename.(bool); ok && !rename {
			continue
		}
		if alias, ok := e.Alias.(bool); ok && !alias {
			continue
		}
		fields[`"`+e.From+`"`] = `"` + toString + `"`
		fields[e.From+":"] = toString + ":"
	}
	return fields, nil
}

func getReplaceableIDs(files []string) map[string]string {
	ids := make(map[string]string)
	for _, file := range files {
		kf := parseKibanaFile(file)
		if kf == nil {
			continue
		}
		for _, obj := range kf.Objects {
			if !strings.Contains(obj.ID, "-ecs") {
				ids[`"`+obj.ID+`"`] = `"` + obj.ID + `-ecs"`
				ids["/"+obj.ID] = "/" + obj.ID + "-ecs"
			}
		}
	}
	return ids
}

func getReplaceableTitles(files []string) map[string]string {
	titles := make(map[string]string)
	for _, file := range files {
		kf := parseKibanaFile(file)
		if kf == nil {
			continue
		}
		for _, obj := range kf.Objects {
			if obj.Attributes.Title != "" && !strings.Contains(obj.Attributes.Title, "ECS") {
				titles[`"`+obj.Attributes.Title+`"`] = `"` + obj.Attributes.Title + ` ECS"`
			}
			if obj.Attributes.VisState.Title != "" && !strings.Contains(obj.Attributes.VisState.Title, "ECS") {
				titles[`"`+obj.Attributes.VisState.Title+`"`] = `"` + obj.Attributes.VisState.Title + ` ECS"`
			}
		}
	}
	return titles
}

func parseKibanaFile(path string) *kibanaFile {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var kf kibanaFile
	if err := json.Unmarshal(data, &kf); err != nil {
		return nil
	}
	return &kf
}

func renameKibanaEntries(files []string, renames map[string]string) error {
	for _, file := range files {
		fmt.Println(file)
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		s := string(data)
		for old, new := range renames {
			s = strings.ReplaceAll(s, old, new)
		}
		if err := os.WriteFile(file, []byte(s), 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", file, err)
		}
	}
	return nil
}

func getKibanaFiles(beatsRoot string) []string {
	patterns := []string{
		filepath.Join(beatsRoot, "*", "_meta", "kibana", "7", "dashboard", "*.json"),
		filepath.Join(beatsRoot, "*", "module", "*", "_meta", "kibana", "7", "dashboard", "*.json"),
		filepath.Join(beatsRoot, "heartbeat", "monitors", "active", "*", "_meta", "kibana", "7", "dashboard", "*.json"),
		filepath.Join(beatsRoot, "x-pack", "*", "module", "*", "_meta", "kibana", "7", "dashboard", "*.json"),
	}
	var files []string
	for _, p := range patterns {
		matches, _ := filepath.Glob(p)
		files = append(files, matches...)
	}
	return files
}
