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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.elastic.co/apm/v2"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/beat/events"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/esleg/eslegclient"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/monitoring/report"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/testing"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var createDocPrivAvailableESVersion = common.MustNewVersion("7.5.0")

type publishClient struct {
	es     *eslegclient.Connection
	params map[string]string

	log *logp.Logger
}

func newPublishClient(
	es *eslegclient.Connection,
	params map[string]string,
) (*publishClient, error) {
	p := &publishClient{
		es:     es,
		params: params,

		log: logp.NewLogger(logSelector),
	}
	return p, nil
}

func (c *publishClient) Connect() error {
	c.log.Debug("Monitoring client: connect.")

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
		c.log.Debug("XPack monitoring is disabled.")
		return errNoMonitoring
	}

	c.log.Debug("XPack monitoring is enabled")

	return nil
}

func (c *publishClient) Close() error {
	return c.es.Close()
}

func (c *publishClient) Publish(ctx context.Context, batch publisher.Batch) error {
	events := batch.Events()
	var failed []publisher.Event
	var reason error
	for _, event := range events {

		// Extract type
		t, err := event.Content.Meta.GetValue("type")
		if err != nil {
			c.log.Errorf("Type not available in monitoring reported. Please report this error: %+v", err)
			continue
		}

		typ, ok := t.(string)
		if !ok {
			c.log.Error("monitoring type is not a string")
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

		if err := c.publishBulk(ctx, event, typ); err != nil {
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
	return "monitoring(" + c.es.URL + ")"
}

func (c *publishClient) publishBulk(ctx context.Context, event publisher.Event, typ string) error {
	meta := mapstr.M{
		"_index":   getMonitoringIndexName(),
		"_routing": nil,
	}

	esVersion := c.es.GetVersion()
	if esVersion.Major < 7 {
		meta["_type"] = "doc"
	}

	opType := events.OpTypeCreate
	if esVersion.LessThan(createDocPrivAvailableESVersion) {
		opType = events.OpTypeIndex
	}

	action := mapstr.M{
		opType.String(): meta,
	}

	event.Content.Fields.Put("timestamp", event.Content.Timestamp)

	fields := mapstr.M{
		"type": typ,
		typ:    event.Content.Fields,
	}

	interval, err := event.Content.Meta.GetValue("interval_ms")
	if err != nil {
		return errors.Wrap(err, "could not determine interval_ms field")
	}
	fields.Put("interval_ms", interval)

	clusterUUID, err := event.Content.Meta.GetValue("cluster_uuid")
	if err != nil && err != mapstr.ErrKeyNotFound {
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
	_, result, err := c.es.Bulk(ctx, getMonitoringIndexName(), "", nil, bulk[:])
	if err != nil {
		apm.CaptureError(ctx, fmt.Errorf("failed to perform any bulk index operations: %w", err)).Send()
		return err
	}

	logBulkFailures(c.log, result, []report.Event{document})
	return err
}

func getMonitoringIndexName() string {
	version := 7
	date := time.Now().Format("2006.01.02")
	return fmt.Sprintf(".monitoring-beats-%v-%s", version, date)
}

func logBulkFailures(log *logp.Logger, result eslegclient.BulkResult, events []report.Event) {
	var response struct {
		Items []map[string]map[string]interface{} `json:"items"`
	}

	if err := json.Unmarshal(result, &response); err != nil {
		log.Errorf("failed to parse monitoring bulk items: %v", err)
		return
	}

	for i := range events {
		for _, innerItem := range response.Items[i] {
			var status int
			if s, exists := innerItem["status"]; exists {
				if v, ok := s.(int); ok {
					status = v
				}
			}

			var errorMsg string
			if e, exists := innerItem["error"]; exists {
				if v, ok := e.(string); ok {
					errorMsg = v
				}
			}

			switch {
			case status < 300, status == http.StatusConflict:
				continue
			default:
				log.Warnf("monitoring bulk item insert failed (i=%v, status=%v): %s", i, status, errorMsg)
			}
		}
	}
}
