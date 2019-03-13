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

package elasticsearch

import (
	"encoding/json"
	"fmt"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/monitoring/report"
	esout "github.com/elastic/beats/libbeat/outputs/elasticsearch"
	"github.com/elastic/beats/libbeat/publisher"
	"github.com/elastic/beats/libbeat/testing"
)

type publishClient struct {
	es     *esout.Client
	params map[string]string
}

func newPublishClient(
	es *esout.Client,
	params map[string]string,
) *publishClient {
	p := &publishClient{
		es:     es,
		params: params,
	}
	return p
}

func (c *publishClient) Connect() error {
	debugf("Monitoring client: connect.")

	params := map[string]string{
		"filter_path": "features.monitoring.enabled",
	}
	status, body, err := c.es.Request("GET", "/_xpack", "", params, nil)
	if err != nil {
		return fmt.Errorf("X-Pack capabilities query failed with: %v", err)
	}

	if status != 200 {
		return fmt.Errorf("X-Pack capabilities query failed with status code: %v", status)
	}

	resp := struct {
		Features struct {
			Monitoring struct {
				Enabled bool
			}
		}
	}{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return err
	}

	if !resp.Features.Monitoring.Enabled {
		debugf("XPack monitoring is disabled.")
		return errNoMonitoring
	}

	debugf("XPack monitoring is enabled")
	return nil
}

func (c *publishClient) Close() error {
	return c.es.Close()
}

func (c *publishClient) Publish(batch publisher.Batch) error {
	events := batch.Events()
	var failed []publisher.Event
	var reason error
	for _, event := range events {

		// Extract time
		t, err := event.Content.Meta.GetValue("type")
		if err != nil {
			logp.Err("Type not available in monitoring reported. Please report this error: %s", err)
			continue
		}

		var params = map[string]string{}
		// Copy params
		for k, v := range c.params {
			params[k] = v
		}
		// Extract potential additional params
		p, err := event.Content.Meta.GetValue("params")
		if err == nil {
			p2, ok := p.(map[string]string)
			if ok {
				for k, v := range p2 {
					params[k] = v
				}
			}
		}

		meta := common.MapStr{
			"_index":   "",
			"_routing": nil,
			"_type":    t,
		}
		bulk := [2]interface{}{
			common.MapStr{"index": meta},
			report.Event{
				Timestamp: event.Content.Timestamp,
				Fields:    event.Content.Fields,
			},
		}

		// Currently one request per event is sent. Reason is that each event can contain different
		// interval params and X-Pack requires to send the interval param.
		_, err = c.es.BulkWith("_xpack", "monitoring", params, nil, bulk[:])
		if err != nil {
			failed = append(failed, event)
			reason = err
		}
	}

	if len(failed) > 0 {
		batch.RetryEvents(failed)
	} else {
		batch.ACK()
	}
	return reason
}

func (c *publishClient) Test(d testing.Driver) {
	c.es.Test(d)
}

func (c *publishClient) String() string {
	return "publish(" + c.es.String() + ")"
}
