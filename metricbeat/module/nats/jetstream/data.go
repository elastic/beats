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
		"category": c.Str("category"),
		"stats": s.Object{
			"streams":          c.Int("streams"),
			"consumers":        c.Int("consumers"),
			"messages":         c.Int("messages"),
			"bytes":            c.Int("bytes"),
			"memory":           c.Int("memory"),
			"reserved_memory":  c.Int("reserved_memory"),
			"storage":          c.Int("storage"),
			"reserved_storage": c.Int("reserved_storage"),
			"accounts":         c.Int("accounts"),
			"config": s.Object{
				"max_memory":    c.Int("max_memory"),
				"max_storage":   c.Int("max_storage"),
				"store_dir":     c.Str("store_dir"),
				"sync_interval": c.Int("sync_interval"),
			},
		},
	}

	jetstreamAccountSchema = s.Schema{
		"category": c.Str("category"),
		"account": s.Object{
			"id":                       c.Str("id"),
			"name":                     c.Str("name"),
			"memory":                   c.Int("memory"),
			"storage":                  c.Int("storage"),
			"reserved_memory":          c.Int("reserved_memory"),
			"reserved_storage":         c.Int("reserved_storage"),
			"accounts":                 c.Int("accounts"),
			"high_availability_assets": c.Int("ha_assets"),
			"api": s.Object{
				"total":  c.Int("api_total"),
				"errors": c.Int("api_errors"),
			},
		},
	}

	jetstreamStreamSchema = s.Schema{
		"category": c.Str("category"),
		"stream": s.Object{
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
				"num_deleted":    c.Int("num_deleted"),
				"num_subjects":   c.Int("num_subjects"),
			},
			"account": s.Object{
				"id":   c.Str("account_id"),
				"name": c.Str("account_name"),
			},
			"config": s.Object{
				"description":          c.Str("config_description"),
				"retention":            c.Str("config_retention"),
				"num_replicas":         c.Int("config_num_replicas"),
				"storage":              c.Str("config_storage"),
				"max_consumers":        c.Int("config_max_consumers"),
				"max_msgs":             c.Int("config_max_msgs"),
				"max_bytes":            c.Int("config_max_bytes"),
				"max_age":              c.Int("config_max_age"),
				"max_msgs_per_subject": c.Int("config_max_msgs_per_subject"),
				"max_msg_size":         c.Int("config_max_msg_size"),
				"subjects": s.Conv{
					Key: "config_subjects",
					Func: func(key string, data map[string]interface{}) (interface{}, error) {
						emptyIface, err := mapstr.M(data).GetValue(key)
						if err != nil {
							return []string{}, s.NewKeyNotFoundError(key)
						}
						switch val := emptyIface.(type) {
						case []string:
							return val, nil
						default:
							msg := fmt.Sprintf("expected []string, found %T", emptyIface)
							return []string{}, s.NewWrongFormatError(key, msg)
						}
					},
				},
			},
		},
	}

	jetstreamConsumerSchema = s.Schema{
		"category": c.Str("category"),
		"consumer": s.Object{
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
				"last_active":  c.Time("delivered_last_active"),
			},
			"ack_floor": s.Object{
				"consumer_seq": c.Int("ack_consumer_seq"),
				"stream_seq":   c.Int("ack_stream_seq"),
				"last_active":  c.Time("ack_last_active"),
			},
			"num_ack_pending":  c.Int("num_ack_pending"),
			"num_redelivered":  c.Int("num_redelivered"),
			"num_waiting":      c.Int("num_waiting"),
			"num_pending":      c.Int("num_pending"),
			"last_active_time": c.Time("ts"),
			"account": s.Object{
				"id":   c.Str("account_id"),
				"name": c.Str("account_name"),
			},
			"config": s.Object{
				"name":            c.Str("name"),
				"durable_name":    c.Str("config_durable_name"),
				"deliver_policy":  c.Str("config_deliver_policy"),
				"filter_subject":  c.Str("config_filter_subject"),
				"replay_policy":   c.Str("config_replay_policy"),
				"ack_policy":      c.Str("config_ack_policy"),
				"ack_wait":        c.Int("config_ack_wait"),
				"max_deliver":     c.Int("config_max_deliver"),
				"max_waiting":     c.Int("config_max_waiting"),
				"max_ack_pending": c.Int("config_max_ack_pending"),
				"num_replicas":    c.Int("config_num_replicas"),
			},
		},
	}
)

type NamedItem interface {
	GetName() string
}

type JetstreamResponse struct {
	AccountDetails  []JetstreamAccountDetails `json:"account_details"`
	Bytes           int                       `json:"bytes"`
	Config          JetstreamConfig           `json:"config,omitempty"`
	Consumers       int                       `json:"consumers"`
	Messages        int                       `json:"messages"`
	Now             time.Time                 `json:"now"`
	ServerID        string                    `json:"server_id"`
	Streams         int                       `json:"streams"`
	Memory          int                       `json:"memory"`
	Storage         int                       `json:"storage"`
	ReservedMemory  int                       `json:"reserved_memory"`
	ReservedStorage int                       `json:"reserved_storage"`
	Accounts        int                       `json:"accounts"`
}

type JetstreamConfig struct {
	MaxMemory    int    `json:"max_memory"`
	MaxStorage   int    `json:"max_storage"`
	StoreDir     string `json:"store_dir"`
	SyncInterval int    `json:"sync_interval"`
}

type JetstreamAccountDetails struct {
	Id                     string                  `json:"id"`
	Name                   string                  `json:"name"`
	Memory                 int                     `json:"memory"`
	Storage                int                     `json:"storage"`
	Accounts               int                     `json:"accounts"`
	ReservedMemory         int                     `json:"reserved_memory"`
	ReservedStorage        int                     `json:"reserved_storage"`
	HighAvailabilityAssets int                     `json:"ha_assets"`
	ApiStats               AccountApiStats         `json:"api"`
	StreamDetails          []JetstreamStreamDetail `json:"stream_detail"`
}

type AccountApiStats struct {
	Total  int `json:"total"`
	Errors int `json:"errors"`
}

func (me JetstreamAccountDetails) GetName() string {
	return me.Name
}

type JetstreamStreamDetail struct {
	Cluster   JetstreamStreamClusterInfo `json:"cluster"`
	Consumers []JetstreamConsumerDetail  `json:"consumer_detail"`
	Created   time.Time                  `json:"created"`
	Name      string                     `json:"name"`
	State     JetstreamStreamState       `json:"state"`
	Config    JetstreamStreamConfig      `json:"config"`
}

func (me JetstreamStreamDetail) GetName() string {
	return me.Name
}

type JetstreamStreamConfig struct {
	Description           string   `json:"description"`
	Retention             string   `json:"retention"`
	Subjects              []string `json:"subjects"`
	AllowRollupHeaders    bool     `json:"allow_rollup_hdrs"`
	DenyPurge             bool     `json:"deny_purge"`
	DenyDelete            bool     `json:"deny_delete"`
	Sealed                bool     `json:"sealed"`
	MirrorDirect          bool     `json:"mirror_direct"`
	AllowDirect           bool     `json:"allow_direct"`
	Compression           string   `json:"compression"`
	DuplicateWindow       int      `json:"duplicate_window"`
	NumReplicas           int      `json:"num_replicas"`
	Storage               string   `json:"storage"`
	MaxConsumers          int      `json:"max_consumers"`
	MaxMsgs               int      `json:"max_msgs"`
	MaxBytes              int      `json:"max_bytes"`
	MaxAge                int      `json:"max_age"`
	MaxMessagesPerSubject int      `json:"max_msgs_per_subject"`
	MaxMessageSize        int      `json:"max_msg_size"`
	Discard               string   `json:"discard"`
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
	NumSubjects    int       `json:"num_subjects"`
	NumDeleted     int       `json:"num_deleted"`
}

type JetstreamConsumerDetail struct {
	AckFloor       JetstreamConsumerAckFloor  `json:"ack_floor"`
	Created        time.Time                  `json:"created"`
	Config         JetstreamConsumerConfig    `json:"config"`
	Delivered      JetstreamConsumerDelivered `json:"delivered"`
	Name           string                     `json:"name"`
	NumAckPending  int                        `json:"num_ack_pending"`
	NumRedelivered int                        `json:"num_redelivered"`
	NumPending     int                        `json:"num_pending"`
	NumWaiting     int                        `json:"num_waiting"`
	StreamName     string                     `json:"stream_name"`
	Timestamp      time.Time                  `json:"ts"`
}

func (me JetstreamConsumerDetail) GetName() string {
	return me.Name
}

type JetstreamConsumerConfig struct {
	DurableName   string `json:"durable_name"`
	DeliverPolicy string `json:"deliver_policy"`
	AckPolicy     string `json:"ack_policy"`
	AckWait       int    `json:"ack_wait"`
	MaxDeliver    int    `json:"max_deliver"`
	FilterSubject string `json:"filter_subject"`
	ReplayPolicy  string `json:"replay_policy"`
	MaxWaiting    int    `json:"max_waiting"`
	MaxAckPending int    `json:"max_ack_pending"`
	NumReplicas   int    `json:"num_replicas"`
}

type JetstreamConsumerDelivered struct {
	ConsumerSequence int       `json:"consumer_seq"`
	StreamSequence   int       `json:"stream_seq"`
	LastActive       time.Time `json:"last_active"`
}

type JetstreamConsumerAckFloor struct {
	ConsumerSequence int       `json:"consumer_seq"`
	StreamSequence   int       `json:"stream_seq"`
	LastActive       time.Time `json:"last_active"`
}

func eventMapping(m *MetricSet, r mb.ReporterV2, content []byte) error {
	var response JetstreamResponse

	err := json.Unmarshal(content, &response)
	if err != nil {
		return fmt.Errorf("failure parsing NATS Jetstream API response: %w", err)
	}

	if m.Config.Stats.Enabled {
		err = statsMapping(r, response)
	}

	if m.Config.Account.Enabled {
		err = accountMapping(r, response, m.Config)
	}

	if m.Config.Stream.Enabled {
		err = streamMapping(r, response, m.Config)
	}

	if m.Config.Consumer.Enabled {
		err = consumerMapping(r, response, m.Config)
	}

	return err
}

func statsMapping(r mb.ReporterV2, response JetstreamResponse) error {
	moduleFields, timestamp, err := getSharedEventDetails(response)

	if err != nil {
		return fmt.Errorf("failure applying module schema: %w", err)
	}

	metricSetFields, err := jetstreamStatsSchema.Apply(map[string]interface{}{
		"category":         statsCategory,
		"max_memory":       response.Config.MaxMemory,
		"max_storage":      response.Config.MaxStorage,
		"store_dir":        response.Config.StoreDir,
		"sync_interval":    response.Config.SyncInterval,
		"streams":          response.Streams,
		"consumers":        response.Consumers,
		"messages":         response.Messages,
		"bytes":            response.Bytes,
		"memory":           response.Memory,
		"storage":          response.Storage,
		"reserved_memory":  response.ReservedMemory,
		"reserved_storage": response.ReservedStorage,
		"accounts":         response.Accounts,
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

func filterByName[T NamedItem](collection []T, allowedValues []string) []T {
	// No filters. Return all.
	if len(allowedValues) == 0 {
		return collection
	}

	// Put into map for faster lookup
	filters := map[string]bool{}
	for _, val := range allowedValues {
		filters[val] = true
	}

	filtered := make([]T, 0)
	for _, item := range collection {
		if filters[item.GetName()] {
			filtered = append(filtered, item)
		}
	}

	return filtered
}

func accountMapping(r mb.ReporterV2, response JetstreamResponse, config MetricsetConfig) error {
	for _, account := range filterByName(response.AccountDetails, config.Account.Names) {
		moduleFields, timestamp, err := getSharedEventDetails(response)

		if err != nil {
			return fmt.Errorf("failure applying module schema: %w", err)
		}

		metricSetFields, err := jetstreamAccountSchema.Apply(map[string]interface{}{
			"category":         accountCategory,
			"id":               account.Id,
			"name":             account.Name,
			"memory":           account.Memory,
			"storage":          account.Storage,
			"reserved_memory":  account.ReservedMemory,
			"reserved_storage": account.ReservedStorage,
			"accounts":         account.Accounts,
			"ha_assets":        account.HighAvailabilityAssets,
			"api_total":        account.ApiStats.Total,
			"api_errors":       account.ApiStats.Errors,
		})

		if err != nil {
			return fmt.Errorf("failure applying jetstream.account schema: %w", err)
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

	return nil
}

func streamMapping(r mb.ReporterV2, response JetstreamResponse, config MetricsetConfig) error {
	for _, account := range filterByName(response.AccountDetails, config.Account.Names) {
		for _, stream := range filterByName(account.StreamDetails, config.Stream.Names) {
			moduleFields, timestamp, err := getSharedEventDetails(response)

			if err != nil {
				return fmt.Errorf("failure applying module schema: %w", err)
			}

			metricSetFields, err := jetstreamStreamSchema.Apply(map[string]interface{}{
				"category":                    streamCategory,
				"name":                        stream.Name,
				"created":                     stream.Created,
				"leader":                      stream.Cluster.Leader,
				"messages":                    stream.State.Messages,
				"bytes":                       stream.State.Bytes,
				"first_seq":                   stream.State.FirstSequence,
				"first_ts":                    stream.State.FirstTimestamp,
				"last_seq":                    stream.State.LastSequence,
				"last_ts":                     stream.State.LastTimestamp,
				"consumer_count":              stream.State.ConsumerCount,
				"num_deleted":                 stream.State.NumDeleted,
				"num_subjects":                stream.State.NumSubjects,
				"account_id":                  account.Id,
				"account_name":                account.Name,
				"config_description":          stream.Config.Description,
				"config_retention":            stream.Config.Retention,
				"config_num_replicas":         stream.Config.NumReplicas,
				"config_storage":              stream.Config.Storage,
				"config_max_consumers":        stream.Config.MaxConsumers,
				"config_subjects":             stream.Config.Subjects,
				"config_max_msgs":             stream.Config.MaxMsgs,
				"config_max_bytes":            stream.Config.MaxBytes,
				"config_max_age":              stream.Config.MaxAge,
				"config_max_msgs_per_subject": stream.Config.MaxMessagesPerSubject,
				"config_max_msg_size":         stream.Config.MaxMessageSize,
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
	for _, account := range filterByName(response.AccountDetails, config.Account.Names) {
		for _, stream := range filterByName(account.StreamDetails, config.Stream.Names) {
			for _, consumer := range filterByName(stream.Consumers, config.Consumer.Names) {
				moduleFields, timestamp, err := getSharedEventDetails(response)

				if err != nil {
					return fmt.Errorf("failure applying module schema: %w", err)
				}

				metricSetFields, err := jetstreamConsumerSchema.Apply(map[string]interface{}{
					"category":               consumerCategory,
					"stream_name":            stream.Name,
					"name":                   consumer.Name,
					"leader":                 stream.Cluster.Leader,
					"created":                consumer.Created,
					"delivered_consumer_seq": consumer.Delivered.ConsumerSequence,
					"delivered_stream_seq":   consumer.Delivered.StreamSequence,
					"delivered_last_active":  consumer.Delivered.LastActive,
					"ack_consumer_seq":       consumer.AckFloor.ConsumerSequence,
					"ack_stream_seq":         consumer.AckFloor.StreamSequence,
					"ack_last_active":        consumer.AckFloor.LastActive,
					"num_ack_pending":        consumer.NumAckPending,
					"num_redelivered":        consumer.NumRedelivered,
					"num_waiting":            consumer.NumWaiting,
					"num_pending":            consumer.NumPending,
					"ts":                     consumer.Timestamp,
					"account_id":             account.Id,
					"account_name":           account.Name,
					"config_durable_name":    consumer.Config.DurableName,
					"config_deliver_policy":  consumer.Config.DeliverPolicy,
					"config_filter_subject":  consumer.Config.FilterSubject,
					"config_replay_policy":   consumer.Config.ReplayPolicy,
					"config_ack_policy":      consumer.Config.AckPolicy,
					"config_ack_wait":        consumer.Config.AckWait,
					"config_max_deliver":     consumer.Config.MaxDeliver,
					"config_max_waiting":     consumer.Config.MaxWaiting,
					"config_max_ack_pending": consumer.Config.MaxAckPending,
					"config_num_replicas":    consumer.Config.NumReplicas,
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
