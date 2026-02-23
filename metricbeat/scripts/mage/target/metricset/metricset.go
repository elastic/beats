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

package metricset

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	devtools "github.com/elastic/beats/v7/dev-tools/mage"
)

// CreateMetricset creates a new metricset.
// Replaces metricbeat/scripts/create_metricset.py.
//
// Required ENV variables:
// * MODULE: Name of the module
// * METRICSET: Name of the metricset
func CreateMetricset() error {
	beatsDir, err := devtools.ElasticBeatsDir()
	if err != nil {
		return err
	}
	basePath := devtools.CWD()
	metricbeatPath := filepath.Join(beatsDir, "metricbeat")

	module := strings.ToLower(os.Getenv("MODULE"))
	metricset := strings.ToLower(os.Getenv("METRICSET"))
	if module == "" {
		return fmt.Errorf("MODULE environment variable is required")
	}
	if metricset == "" {
		return fmt.Errorf("METRICSET environment variable is required")
	}

	return generateMetricset(basePath, metricbeatPath, module, metricset)
}

func generateMetricset(basePath, metricbeatPath, module, metricset string) error {
	if err := generateModule(basePath, metricbeatPath, module, metricset); err != nil {
		return err
	}

	metricsetPath := filepath.Join(basePath, "module", module, metricset)
	metaPath := filepath.Join(metricsetPath, "_meta")

	if info, err := os.Stat(metricsetPath); err == nil && info.IsDir() {
		fmt.Printf("Metricset already exists. Skipping creating metricset %s\n", metricset)
		return nil
	}

	if err := os.MkdirAll(metaPath, 0o755); err != nil {
		return fmt.Errorf("creating metricset directories: %w", err)
	}

	templates := filepath.Join(metricbeatPath, "scripts", "module", "metricset")

	content, err := loadTemplate(filepath.Join(templates, "metricset.go.tmpl"), module, metricset)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(metricsetPath, metricset+".go"), []byte(content), 0o644); err != nil {
		return err
	}

	content, err = loadTemplate(filepath.Join(templates, "fields.yml"), module, metricset)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(metaPath, "fields.yml"), []byte(content), 0o644); err != nil {
		return err
	}

	content, err = loadTemplate(filepath.Join(templates, "docs.md"), module, metricset)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(metaPath, "docs.md"), []byte(content), 0o644); err != nil {
		return err
	}

	content, err = loadTemplate(filepath.Join(templates, "data.json"), module, metricset)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(metaPath, "data.json"), []byte(content), 0o644); err != nil {
		return err
	}

	fmt.Printf("Metricset %s created.\n", metricset)
	return nil
}

func generateModule(basePath, metricbeatPath, module, metricset string) error {
	modulePath := filepath.Join(basePath, "module", module)
	metaPath := filepath.Join(modulePath, "_meta")

	if info, err := os.Stat(modulePath); err == nil && info.IsDir() {
		fmt.Printf("Module already exists. Skipping creating module %s\n", module)
		return nil
	}

	if err := os.MkdirAll(metaPath, 0o755); err != nil {
		return fmt.Errorf("creating module directories: %w", err)
	}

	templates := filepath.Join(metricbeatPath, "scripts", "module")

	content, err := loadTemplate(filepath.Join(templates, "fields.yml"), module, "")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(metaPath, "fields.yml"), []byte(content), 0o644); err != nil {
		return err
	}

	content, err = loadTemplate(filepath.Join(templates, "docs.md"), module, "")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(metaPath, "docs.md"), []byte(content), 0o644); err != nil {
		return err
	}

	content, err = loadTemplate(filepath.Join(templates, "config.yml"), module, metricset)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(metaPath, "config.yml"), []byte(content), 0o644); err != nil {
		return err
	}

	content, err = loadTemplate(filepath.Join(templates, "doc.go.tmpl"), module, "")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(modulePath, "doc.go"), []byte(content), 0o644); err != nil {
		return err
	}

	fmt.Printf("Module %s created.\n", module)
	return nil
}

func loadTemplate(path, module, metricset string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading template %s: %w", path, err)
	}
	content := strings.ReplaceAll(string(data), "{module}", module)
	content = strings.ReplaceAll(content, "{metricset}", metricset)
	return content, nil
}
