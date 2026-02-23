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
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
)

var reUnsafeFilenameChars = regexp.MustCompile(`[><:"/\\|?*]`)

// Export5xDashboards exports Kibana 5.x dashboards from Elasticsearch.
// Replaces dev-tools/cmd/dashboards/export_5x_dashboards.py.
// Set ES_URL (default http://localhost:9200), REGEX (required),
// KIBANA_INDEX (default .kibana), OUTPUT_DIR (default output).
func Export5xDashboards() error {
	esURL := os.Getenv("ES_URL")
	if esURL == "" {
		esURL = "http://localhost:9200"
	}
	regexPattern := os.Getenv("REGEX")
	if regexPattern == "" {
		regexPattern = ".*"
	}
	kibanaIndex := os.Getenv("KIBANA_INDEX")
	if kibanaIndex == "" {
		kibanaIndex = ".kibana"
	}
	outputDir := os.Getenv("OUTPUT_DIR")
	if outputDir == "" {
		outputDir = "output"
	}

	re, err := regexp.Compile("(?i)" + regexPattern)
	if err != nil {
		return fmt.Errorf("invalid regex %q: %w", regexPattern, err)
	}

	hits, err := esSearch(esURL, kibanaIndex, "dashboard", 1000)
	if err != nil {
		return fmt.Errorf("searching dashboards: %w", err)
	}

	for _, doc := range hits {
		title, _ := doc.source["title"].(string)
		if !re.MatchString(title) {
			fmt.Println("Ignore dashboard", title)
			continue
		}

		if err := saveKibanaJSON("dashboard", doc, outputDir); err != nil {
			return err
		}

		panelsRaw, _ := doc.source["panelsJSON"].(string)
		var panels []map[string]interface{}
		if err := json.Unmarshal([]byte(panelsRaw), &panels); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to parse panelsJSON for %s: %v\n", title, err)
			continue
		}
		for _, panel := range panels {
			ptype, _ := panel["type"].(string)
			pid, _ := panel["id"].(string)
			switch ptype {
			case "visualization":
				if err := exportVisualization(esURL, pid, kibanaIndex, outputDir); err != nil {
					return err
				}
			case "search":
				if err := exportSearch(esURL, pid, kibanaIndex, outputDir); err != nil {
					return err
				}
			default:
				fmt.Printf("Unknown type %s in dashboard\n", ptype)
			}
		}
	}
	return nil
}

func exportVisualization(esURL, id, kibanaIndex, outputDir string) error {
	doc, err := esGet(esURL, kibanaIndex, "visualization", id)
	if err != nil {
		return fmt.Errorf("getting visualization %s: %w", id, err)
	}
	if err := saveKibanaJSON("visualization", doc, outputDir); err != nil {
		return err
	}
	if searchID, ok := doc.source["savedSearchId"].(string); ok {
		return exportSearch(esURL, searchID, kibanaIndex, outputDir)
	}
	return nil
}

func exportSearch(esURL, id, kibanaIndex, outputDir string) error {
	doc, err := esGet(esURL, kibanaIndex, "search", id)
	if err != nil {
		return fmt.Errorf("getting search %s: %w", id, err)
	}
	return saveKibanaJSON("search", doc, outputDir)
}

type esDoc struct {
	id     string
	source map[string]interface{}
}

func esSearch(esURL, index, docType string, size int) ([]esDoc, error) {
	url := fmt.Sprintf("%s/%s/%s/_search?size=%d", esURL, index, docType, size)
	resp, err := http.Get(url) //nolint:gosec // URL is constructed from user-provided parameters
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ES search returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Hits struct {
			Hits []struct {
				ID     string                 `json:"_id"`
				Source map[string]interface{} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	var docs []esDoc
	for _, h := range result.Hits.Hits {
		docs = append(docs, esDoc{id: h.ID, source: h.Source})
	}
	return docs, nil
}

func esGet(esURL, index, docType, id string) (esDoc, error) {
	url := fmt.Sprintf("%s/%s/%s/%s", esURL, index, docType, id)
	resp, err := http.Get(url) //nolint:gosec // URL is constructed from user-provided parameters
	if err != nil {
		return esDoc{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return esDoc{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return esDoc{}, fmt.Errorf("ES get returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		ID     string                 `json:"_id"`
		Source map[string]interface{} `json:"_source"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return esDoc{}, err
	}
	return esDoc{id: result.ID, source: result.Source}, nil
}

func saveKibanaJSON(docType string, doc esDoc, outputDir string) error {
	dir := filepath.Join(outputDir, docType)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	safeID := reUnsafeFilenameChars.ReplaceAllString(doc.id, "")
	fp := filepath.Join(dir, safeID+".json")

	data, err := json.MarshalIndent(doc.source, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(fp, append(data, '\n'), 0o644); err != nil {
		return err
	}
	fmt.Printf("Written %s\n", fp)
	return nil
}
