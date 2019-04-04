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
	"time"

	"github.com/pkg/errors"

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
	format report.Format
}

func newPublishClient(
	es *esout.Client,
	params map[string]string,
	format report.Format,
) (*publishClient, error) {
	p := &publishClient{
		es:     es,
		params: params,
		format: format,
	}
	return p, nil
}

func (c *publishClient) Connect() error {
	debugf("Monitoring client: connect.")

	err := c.es.Connect()
	if err != nil {
		return errors.Wrap(err, "cannot connect underlying Elasticsearch client")
	}

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

		// Extract type
		t, err := event.Content.Meta.GetValue("type")
		if err != nil {
			logp.Err("Type not available in monitoring reported. Please report this error: %s", err)
			continue
		}

		typ, ok := t.(string)
		if !ok {
			logp.Err("monitoring type is not a string")
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

		switch c.format {
		case report.FormatXPackMonitoringBulk:
			err = c.publishXPackBulk(params, event, typ)
		case report.FormatBulk:
			err = c.publishBulk(event, typ)
		}

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

func (c *publishClient) publishXPackBulk(params map[string]string, event publisher.Event, typ string) error {
	meta := common.MapStr{
		"_index":   "",
		"_routing": nil,
		"_type":    typ,
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
	_, err := c.es.SendMonitoringBulk(params, bulk[:])
	return err
}

func (c *publishClient) publishBulk(event publisher.Event, typ string) error {
	meta := common.MapStr{
		"_index":   getMonitoringIndexName(),
		"_routing": nil,
	}

	if c.es.GetVersion().Major < 7 {
		meta["_type"] = "doc"
	}

	action := common.MapStr{
		"index": meta,
	}

	event.Content.Fields.Put("timestamp", event.Content.Timestamp)

	fields := common.MapStr{
		"type": typ,
		typ:    event.Content.Fields,
	}

	interval, err := event.Content.Meta.GetValue("interval_ms")
	if err != nil {
		return errors.Wrap(err, "could not determine interval_ms field")
	}
	fields.Put("interval_ms", interval)

	clusterUUID, err := event.Content.Meta.GetValue("cluster_uuid")
	if err != nil && err != common.ErrKeyNotFound {
		return errors.Wrap(err, "could not determine cluster_uuid field")
	}
	fields.Put("cluster_uuid", clusterUUID)

	document := report.Event{
		Timestamp: event.Content.Timestamp,
		Fields:    fields,
	}
	bulk := [2]interface{}{action, document}

	// Currently one request per event is sent. Reason is that each event can contain different
	// interval params and X-Pack requires to send the interval param.
	// FIXME: index name (first param below)
	_, err = c.es.BulkWith(getMonitoringIndexName(), "", nil, nil, bulk[:])
	return err
}

func getMonitoringIndexName() string {
	version := 7
	date := time.Now().Format("2006.01.02")
	return fmt.Sprintf(".monitoring-beats-%v-%s", version, date)
}
