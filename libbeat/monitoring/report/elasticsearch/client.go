package elasticsearch

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/outputs"
	esout "github.com/elastic/beats/libbeat/outputs/elasticsearch"
)

type publishClient struct {
	es         *esout.Client
	params     map[string]string
	windowSize int
}

var (
	// monitoring data action
	actMonitoringData = common.MapStr{
		"index": common.MapStr{
			"_index":   "_data",
			"_type":    "beats",
			"_routing": nil,
		},
	}

	// monitoring beats action
	actMonitoringBeats = common.MapStr{
		"index": common.MapStr{
			"_index":   "",
			"_type":    "beats_stats",
			"_routing": nil,
		},
	}
)

func newPublishClient(
	es *esout.Client,
	params map[string]string,
	windowSize int,
) *publishClient {
	p := &publishClient{
		es:         es,
		params:     params,
		windowSize: windowSize,
	}
	return p
}

func (c *publishClient) Connect(timeout time.Duration) error {
	debugf("Monitoring client: connect.")

	params := map[string]string{
		"filter_path": "features.monitoring.enabled",
	}
	status, body, err := c.es.Request("GET", "/_xpack", "", params, nil)
	if err != nil {
		debugf("XPack capabilities query failed with: %v", err)
		return err
	}

	if status != 200 {
		err := fmt.Errorf("XPack capabilities query failed with status code: %v", status)
		debugf("%s", err)
		return err
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

func (c *publishClient) PublishEvent(data outputs.Data) error {
	_, err := c.PublishEvents([]outputs.Data{data})
	return err
}

func (c *publishClient) PublishEvents(data []outputs.Data) (nextEvents []outputs.Data, err error) {

	for len(data) > 0 {
		windowSize := c.windowSize / 2 // events are send twice right now -> split default windows size in half
		if len(data) < windowSize {
			windowSize = len(data)
		}

		err := c.publish(data[:windowSize])
		if err != nil {
			return data, err
		}

		data = data[windowSize:]
	}

	return nil, nil
}

func (c *publishClient) publish(data []outputs.Data) error {
	// TODO: add event id to reduce chance of duplicates in case of send retry

	bulk := make([]interface{}, 0, 4*len(data))
	for _, d := range data {
		bulk = append(bulk,
			actMonitoringData, d.Event,
			actMonitoringBeats, d.Event)
	}

	_, err := c.es.BulkWith("_xpack", "monitoring", c.params, nil, bulk)
	// TODO: extend error message with details from response
	return err
}
