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

package dashboards

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/elastic/elastic-agent-libs/kibana"
	"github.com/elastic/elastic-agent-libs/version"
)

var (
	// We started using Saved Objects API in 7.15. But to help integration
	// developers migrate their dashboards we are more lenient.
	MinimumRequiredVersionSavedObjects = version.MustNew("7.14.0")
)

// GetDashboard returns the dashboard with the given id with the index pattern removed
func Get(client *kibana.Client, id string) ([]byte, error) {
	if client.Version.LessThan(MinimumRequiredVersionSavedObjects) {
		return nil, fmt.Errorf("Kibana version must be at least " + MinimumRequiredVersionSavedObjects.String())
	}

	// add a special header for serverless, where saved_objects is "hidden"
	headers := http.Header{}
	if serverless, _ := client.KibanaIsServerless(); serverless {
		headers.Add("x-elastic-internal-origin", "libbeat")
	}

	body := fmt.Sprintf(`{"objects": [{"type": "dashboard", "id": "%s" }], "includeReferencesDeep": true, "excludeExportDetails": true}`, id)
	statusCode, response, err := client.Request("POST", "/api/saved_objects/_export", nil, headers, strings.NewReader(body))
	if err != nil || statusCode >= 300 {
		return nil, fmt.Errorf("error exporting dashboard: %w, code: %d", err, statusCode)
	}

	result, err := RemoveIndexPattern(response)
	if err != nil {
		return nil, fmt.Errorf("error removing index pattern: %w", err)
	}

	return result, nil
}
