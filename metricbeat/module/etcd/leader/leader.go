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

package leader

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/logp"

	"github.com/elastic/beats/v8/metricbeat/helper"
	"github.com/elastic/beats/v8/metricbeat/mb"
	"github.com/elastic/beats/v8/metricbeat/mb/parse"
)

const (
	defaultScheme = "http"
	defaultPath   = "/v2/stats/leader"
	apiVersion    = "2"

	// returned JSON management
	msgElement        = "message"
	msgValueNonLeader = "not current leader"

	logSelector = "etcd.leader"
)

var (
	hostParser = parse.URLHostParserBuilder{
		DefaultScheme: defaultScheme,
		DefaultPath:   defaultPath,
	}.Build()
)

func init() {
	mb.Registry.MustAddMetricSet("etcd", "leader", New,
		mb.WithHostParser(hostParser),
		mb.DefaultMetricSet(),
	)
}

// MetricSet for etcd.leader
type MetricSet struct {
	mb.BaseMetricSet
	http         *helper.HTTP
	logger       *logp.Logger
	debugEnabled bool
}

// New etcd.leader metricset object
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	config := struct{}{}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	http, err := helper.NewHTTP(base)
	if err != nil {
		return nil, err
	}
	return &MetricSet{
		base,
		http,
		logp.NewLogger(logSelector),
		logp.IsDebug(logSelector),
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(reporter mb.ReporterV2) error {
	res, err := m.http.FetchResponse()
	if err != nil {
		return errors.Wrap(err, "error fetching response")
	}
	defer res.Body.Close()

	content, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return errors.Wrapf(err, "error reading body response")
	}

	if res.StatusCode == http.StatusOK {
		reporter.Event(mb.Event{
			MetricSetFields: eventMapping(content),
			ModuleFields:    common.MapStr{"api_version": apiVersion},
		})
		return nil
	}

	// Errors might be reported as {"message":"<error message>"}
	// let's look for that structure
	var jsonResponse map[string]interface{}
	if err = json.Unmarshal(content, &jsonResponse); err == nil {
		if retMessage := jsonResponse[msgElement]; retMessage != "" {
			// there is an error message element, let's use it

			// If a 403 is returned and {"message":"not current leader"}
			// do not consider this an error
			// do not report events since this is not a leader
			if res.StatusCode == http.StatusForbidden &&
				retMessage == msgValueNonLeader {
				if m.debugEnabled {
					m.logger.Debugf("skipping event for non leader member %q", m.Host())
				}
				return nil
			}

			return fmt.Errorf("fetching HTTP response returned status code %d: %s",
				res.StatusCode, retMessage)
		}
	}

	// no message in the JSON payload, return standard error
	return fmt.Errorf("fetching HTTP response returned status code %d", res.StatusCode)

}
