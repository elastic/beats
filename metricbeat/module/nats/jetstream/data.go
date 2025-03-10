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
			"time": c.Str("now"),
		},
	}

	jetstreamStatsSchema = s.Schema{
		"config": s.Object{
			"max_memory":    c.Int("max_memory"),
			"max_storage":   c.Int("max_storage"),
			"store_dir":     c.Str("store_dir"),
			"sync_interval": c.Int("sync_interval"),
			"compress_ok":   c.Bool("compress_ok"),
		},
		"streams":   c.Int("streams"),
		"consumers": c.Int("consumers"),
		"messages":  c.Int("messages"),
		"bytes":     c.Int("bytes"),
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
		"stream_name": c.Str("stream_name"),
		"name":        c.Str("name"),
		"created":     c.Time("created"),
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
		"ts":              c.Time("ts"),
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
	AckFloor          JetstreamConsumerAckFloor  `json:"ack_floor"`
	Created           time.Time                  `json:"created"`
	Delivered         JetstreamConsumerDelivered `json:"delivered"`
	Name              string                     `json:"name"`
	NumAckPending     int                        `json:"num_ack_pending"`
	NumAckRedelivered int                        `json:"num_ack_redelivered"`
	NumPending        int                        `json:"num_pending"`
	NumWaiting        int                        `json:"num_waiting"`
	StreamName        string                     `json:"stream_name"`
	Timestamp         time.Time                  `json:"ts"`
}

type JetstreamConsumerDelivered struct {
	ConsumerSequence int `json:"consumer_seq"`
	StreamSequence   int `json:"stream_seq"`
}

type JetstreamConsumerAckFloor struct {
	ConsumerSequence int `json:"consumer_seq"`
	StreamSequence   int `json:"stream_seq"`
}

func eventMapping(metricsetName string, r mb.ReporterV2, content []byte) error {
	var response JetstreamResponse

	err := json.Unmarshal(content, &response)
	if err != nil {
		return fmt.Errorf("failure parsing NATS Jetstream API response: %w", err)
	}

	moduleFields, timestamp, err := getSharedEventDetails(response)

	if err != nil {
		return fmt.Errorf("failure applying module schema: %w", err)
	}

	switch metricsetName {
	case statsMetricset:
		return statsMapping(r, response, moduleFields, timestamp)
	case streamMetricset:
		return streamMapping(r, response, moduleFields, timestamp)
	case consumerMetricset:
		return consumerMapping(r, response, moduleFields, timestamp)
	default:
		return nil
	}
}

func statsMapping(r mb.ReporterV2, response JetstreamResponse, moduleFields mapstr.M, timestamp time.Time) error {
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

func streamMapping(r mb.ReporterV2, response JetstreamResponse, moduleFields mapstr.M, timestamp time.Time) error {
	for _, account := range response.AccountDetails {
		for _, stream := range account.StreamDetails {
			metricSetFields, err := jetstreamStatsSchema.Apply(map[string]interface{}{
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

			r.Event(event)
		}
	}

	return nil
}

func consumerMapping(r mb.ReporterV2, response JetstreamResponse, moduleFields mapstr.M, timestamp time.Time) error {
	for _, account := range response.AccountDetails {
		for _, stream := range account.StreamDetails {
			for _, consumer := range stream.Consumers {
				metricSetFields, err := jetstreamStatsSchema.Apply(map[string]interface{}{
					"stream_name":            stream.Name,
					"name":                   consumer.Name,
					"created":                consumer.Created,
					"delivered_consumer_seq": consumer.Delivered.ConsumerSequence,
					"delivered_stream_seq":   consumer.Delivered.StreamSequence,
					"ack_consumer_seq":       consumer.AckFloor.ConsumerSequence,
					"ack_stream_seq":         consumer.AckFloor.StreamSequence,
					"num_ack_pending":        consumer.NumAckPending,
					"num_ack_redelivered":    consumer.NumAckRedelivered,
					"num_waiting":            consumer.NumWaiting,
					"num_pending":            consumer.NumPending,
					"ts":                     consumer.Timestamp,
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

				r.Event(event)
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
