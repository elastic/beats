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

//go:build integration

package serverless

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/elastic/elastic-agent-libs/kibana"
)

// Dashboard is a Kibana dashboard saved object.
type Dashboard struct {
	Type      string    `json:"type"`
	ID        string    `json:"id"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedAt time.Time `json:"created_at"`
}

type dashboardResponse struct {
	Total        int         `json:"total"`
	SavedObjects []Dashboard `json:"saved_objects"`
}

// kibanaHeaders marks requests as internal, which Kibana requires for saved objects APIs on serverless.
func kibanaHeaders() http.Header {
	h := http.Header{}
	h.Add("x-elastic-internal-origin", "integration-tests")
	return h
}

// DeleteDashboard removes the dashboard with the given ID.
func DeleteDashboard(_ context.Context, client *kibana.Client, id string) error {
	status, resp, err := client.Request(http.MethodDelete, fmt.Sprintf("/api/saved_objects/dashboard/%s", id), nil, kibanaHeaders(), nil)
	if err != nil {
		return fmt.Errorf("error making API request: %w, response: '%s'", err, string(resp))
	}
	if status != http.StatusOK {
		return fmt.Errorf("non-200 return code: %v, response: '%s'", status, string(resp))
	}
	return nil
}

// GetDashboards returns all dashboards known to Kibana.
func GetDashboards(_ context.Context, client *kibana.Client) ([]Dashboard, error) {
	var dashboards []Dashboard
	for page := 1; ; page++ {
		params := url.Values{}
		params.Set("type", "dashboard")
		params.Set("page", fmt.Sprint(page))

		status, resp, err := client.Request(http.MethodGet, "/api/saved_objects/_find", params, kibanaHeaders(), nil)
		if err != nil {
			return nil, fmt.Errorf("error making api request: %w", err)
		}
		if status != http.StatusOK {
			return nil, fmt.Errorf("non-200 return code: %v, response: '%s'", status, string(resp))
		}

		var dashResp dashboardResponse
		if err := json.Unmarshal(resp, &dashResp); err != nil {
			return nil, fmt.Errorf("error unmarshalling dashboard response: %w", err)
		}
		dashboards = append(dashboards, dashResp.SavedObjects...)
		if len(dashResp.SavedObjects) == 0 || len(dashboards) >= dashResp.Total {
			return dashboards, nil
		}
	}
}
