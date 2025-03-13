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

package jetstream

import (
	"encoding/json"
	"fmt"
	"time"

	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var (
	moduleSchema = s.Schema{
		"server": s.Object{
			"id":   c.Str("server_id"),
			"time": c.Time("now"),
		},
	}

	jetstreamStatsSchema = s.Schema{
		"streams":   c.Int("streams"),
		"consumers": c.Int("consumers"),
		"messages":  c.Int("messages"),
		"bytes":     c.Int("bytes"),
		"config": s.Object{
			"max_memory":    c.Int("max_memory"),
			"max_storage":   c.Int("max_storage"),
			"store_dir":     c.Str("store_dir"),
			"sync_interval": c.Int("sync_interval"),
			"compress_ok":   c.Bool("compress_ok"),
		},
	}

	jetstreamStreamSchema = s.Schema{
		"name":    c.Str("name"),
		"created": c.Time("created"),
		"cluster": s.Object{
			"leader": c.Str("leader"),
		},
		"state": s.Object{
			"messages":       c.Int("messages"),
			"bytes":          c.Int("bytes"),
			"first_seq":      c.Int("first_seq"),
			"first_ts":       c.Time("first_ts"),
			"last_seq":       c.Int("last_seq"),
			"last_ts":        c.Time("last_ts"),
			"consumer_count": c.Int("consumer_count"),
		},
		"account": s.Object{
			"id":   c.Str("account_id"),
			"name": c.Str("account_name"),
		},
	}

	jetstreamConsumerSchema = s.Schema{
		"name":    c.Str("name"),
		"created": c.Time("created"),
		"stream": s.Object{
			"name": c.Str("stream_name"),
		},
		"cluster": s.Object{
			"leader": c.Str("leader"),
		},
		"delivered": s.Object{
			"consumer_seq": c.Int("delivered_consumer_seq"),
			"stream_seq":   c.Int("delivered_stream_seq"),
		},
		"ack_floor": s.Object{
			"consumer_seq": c.Int("ack_consumer_seq"),
			"stream_seq":   c.Int("ack_stream_seq"),
		},
		"num_ack_pending": c.Int("num_ack_pending"),
		"num_redelivered": c.Int("num_redelivered"),
		"num_waiting":     c.Int("num_waiting"),
		"num_pending":     c.Int("num_pending"),
		"timestamp":       c.Time("ts"),
		"account": s.Object{
			"id":   c.Str("account_id"),
			"name": c.Str("account_name"),
		},
	}
)

type JetstreamResponse struct {
	AccountDetails []JetstreamAccountDetails `json:"account_details"`
	Bytes          int                       `json:"bytes"`
	Config         JetstreamConfig           `json:"config,omitempty"`
	Consumers      int                       `json:"consumers"`
	Messages       int                       `json:"messages"`
	Now            time.Time                 `json:"now"`
	ServerID       string                    `json:"server_id"`
	Streams        int                       `json:"streams"`
}

type JetstreamConfig struct {
	ComrpessOk   bool   `json:"compress_ok"`
	MaxMemory    int    `json:"max_memory"`
	MaxStorage   int    `json:"max_storage"`
	StoreDir     string `json:"store_dir"`
	SyncInterval int    `json:"sync_interval"`
}

type JetstreamAccountDetails struct {
	Id            string                  `json:"id"`
	Name          string                  `json:"name"`
	Memory        int                     `json:"memory"`
	Storage       int                     `json:"storage"`
	Accounts      int                     `json:"accounts"`
	StreamDetails []JetstreamStreamDetail `json:"stream_detail"`
}

type JetstreamStreamDetail struct {
	Cluster   JetstreamStreamClusterInfo `json:"cluster"`
	Consumers []JetstreamConsumerDetail  `json:"consumer_detail"`
	Created   time.Time                  `json:"created"`
	Name      string                     `json:"name"`
	State     JetstreamStreamState       `json:"state"`
}

type JetstreamStreamClusterInfo struct {
	Leader string `json:"leader"`
}

type JetstreamStreamState struct {
	Bytes          int       `json:"bytes"`
	ConsumerCount  int       `json:"consumer_count"`
	FirstSequence  int       `json:"first_seq"`
	FirstTimestamp time.Time `json:"first_ts"`
	LastSequence   int       `json:"last_seq"`
	LastTimestamp  time.Time `json:"last_ts"`
	Messages       int       `json:"messages"`
}

type JetstreamConsumerDetail struct {
	AckFloor       JetstreamConsumerAckFloor  `json:"ack_floor"`
	Created        time.Time                  `json:"created"`
	Delivered      JetstreamConsumerDelivered `json:"delivered"`
	Name           string                     `json:"name"`
	NumAckPending  int                        `json:"num_ack_pending"`
	NumRedelivered int                        `json:"num_redelivered"`
	NumPending     int                        `json:"num_pending"`
	NumWaiting     int                        `json:"num_waiting"`
	StreamName     string                     `json:"stream_name"`
	Timestamp      time.Time                  `json:"ts"`
}

type JetstreamConsumerDelivered struct {
	ConsumerSequence int `json:"consumer_seq"`
	StreamSequence   int `json:"stream_seq"`
}

type JetstreamConsumerAckFloor struct {
	ConsumerSequence int `json:"consumer_seq"`
	StreamSequence   int `json:"stream_seq"`
}

func eventMapping(m *MetricSet, r mb.ReporterV2, content []byte) error {
	var response JetstreamResponse

	err := json.Unmarshal(content, &response)
	if err != nil {
		return fmt.Errorf("failure parsing NATS Jetstream API response: %w", err)
	}

	switch m.Name() {
	case statsMetricset:
		return statsMapping(r, response)
	case streamMetricset:
		return streamMapping(r, response, m.Config)
	case consumerMetricset:
		return consumerMapping(r, response, m.Config)
	default:
		return nil
	}
}

func statsMapping(r mb.ReporterV2, response JetstreamResponse) error {
	moduleFields, timestamp, err := getSharedEventDetails(response)

	if err != nil {
		return fmt.Errorf("failure applying module schema: %w", err)
	}

	metricSetFields, err := jetstreamStatsSchema.Apply(map[string]interface{}{
		"max_memory":    response.Config.MaxMemory,
		"max_storage":   response.Config.MaxStorage,
		"store_dir":     response.Config.StoreDir,
		"sync_interval": response.Config.SyncInterval,
		"compress_ok":   response.Config.ComrpessOk,
		"streams":       response.Streams,
		"consumers":     response.Consumers,
		"messages":      response.Messages,
		"bytes":         response.Bytes,
	})

	if err != nil {
		return fmt.Errorf("failure applying jetstream.stats schema: %w", err)
	}

	// Create and emit the event
	event := mb.Event{
		MetricSetFields: metricSetFields,
		ModuleFields:    moduleFields,
		Timestamp:       timestamp,
	}

	r.Event(event)

	return nil
}

func filterStreams(streams []JetstreamStreamDetail, config MetricsetConfig) []JetstreamStreamDetail {
	// No filters. Return all.
	if len(config.Stream.Names) == 0 {
		return streams
	}

	// Put into map for faster lookup
	streamFilters := map[string]bool{}
	for _, name := range config.Stream.Names {
		streamFilters[name] = true
	}

	filtered := make([]JetstreamStreamDetail, 0)

	for _, stream := range streams {
		if streamFilters[stream.Name] {
			filtered = append(filtered, stream)
		}
	}

	return filtered
}

func filterConsumers(consumers []JetstreamConsumerDetail, config MetricsetConfig) []JetstreamConsumerDetail {
	// No filters. Return all.
	if len(config.Consumer.Names) == 0 {
		return consumers
	}

	// Put into map for faster lookup
	consumerFilters := map[string]bool{}
	for _, name := range config.Consumer.Names {
		consumerFilters[name] = true
	}

	filtered := make([]JetstreamConsumerDetail, 0)

	for _, consumer := range consumers {
		if consumerFilters[consumer.Name] {
			filtered = append(filtered, consumer)
		}
	}

	return filtered
}

func streamMapping(r mb.ReporterV2, response JetstreamResponse, config MetricsetConfig) error {
	for _, account := range response.AccountDetails {
		for _, stream := range filterStreams(account.StreamDetails, config) {
			moduleFields, timestamp, err := getSharedEventDetails(response)

			if err != nil {
				return fmt.Errorf("failure applying module schema: %w", err)
			}

			metricSetFields, err := jetstreamStreamSchema.Apply(map[string]interface{}{
				"name":           stream.Name,
				"created":        stream.Created,
				"leader":         stream.Cluster.Leader,
				"messages":       stream.State.Messages,
				"bytes":          stream.State.Bytes,
				"first_seq":      stream.State.FirstSequence,
				"first_ts":       stream.State.FirstTimestamp,
				"last_seq":       stream.State.LastSequence,
				"last_ts":        stream.State.LastTimestamp,
				"consumer_count": stream.State.ConsumerCount,
				"account_id":     account.Id,
				"account_name":   account.Name,
			})

			if err != nil {
				return fmt.Errorf("failure applying jetstream.stream schema: %w", err)
			}

			// Create and emit the event
			event := mb.Event{
				MetricSetFields: metricSetFields,
				ModuleFields:    moduleFields,
				Timestamp:       timestamp,
			}

			if !r.Event(event) {
				return nil
			}
		}
	}

	return nil
}

func consumerMapping(r mb.ReporterV2, response JetstreamResponse, config MetricsetConfig) error {
	for _, account := range response.AccountDetails {
		for _, stream := range filterStreams(account.StreamDetails, config) {
			for _, consumer := range filterConsumers(stream.Consumers, config) {
				moduleFields, timestamp, err := getSharedEventDetails(response)

				if err != nil {
					return fmt.Errorf("failure applying module schema: %w", err)
				}

				metricSetFields, err := jetstreamConsumerSchema.Apply(map[string]interface{}{
					"stream_name":            stream.Name,
					"name":                   consumer.Name,
					"leader":                 stream.Cluster.Leader,
					"created":                consumer.Created,
					"delivered_consumer_seq": consumer.Delivered.ConsumerSequence,
					"delivered_stream_seq":   consumer.Delivered.StreamSequence,
					"ack_consumer_seq":       consumer.AckFloor.ConsumerSequence,
					"ack_stream_seq":         consumer.AckFloor.StreamSequence,
					"num_ack_pending":        consumer.NumAckPending,
					"num_redelivered":        consumer.NumRedelivered,
					"num_waiting":            consumer.NumWaiting,
					"num_pending":            consumer.NumPending,
					"ts":                     consumer.Timestamp,
					"account_id":             account.Id,
					"account_name":           account.Name,
				})

				if err != nil {
					return fmt.Errorf("failure applying jetstream.consumer schema: %w", err)
				}

				// Create and emit the event
				event := mb.Event{
					MetricSetFields: metricSetFields,
					ModuleFields:    moduleFields,
					Timestamp:       timestamp,
				}

				if !r.Event(event) {
					return nil
				}
			}
		}
	}

	return nil
}

func getSharedEventDetails(response JetstreamResponse) (mapstr.M, time.Time, error) {
	moduleFields, err := moduleSchema.Apply(map[string]interface{}{
		"server_id": response.ServerID,
		"now":       response.Now,
	})

	if err != nil {
		return nil, time.Now(), fmt.Errorf("failure applying module schema: %w", err)
	}

	return moduleFields, response.Now, nil
}
