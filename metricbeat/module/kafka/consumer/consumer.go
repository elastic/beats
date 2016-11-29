package consumer

import (
	"net/http"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"fmt"
	"io/ioutil"
)

var (
	debugf = logp.MakeDebug("kafka-consumer")
)

// init registers the partition MetricSet with the central registry.
func init() {
	if err := mb.Registry.AddMetricSet("kafka", "consumer", New); err != nil {
		panic(err)
	}
}

// MetricSet type defines all fields of the partition MetricSet
type MetricSet struct {
	mb.BaseMetricSet
	client          *http.Client
	host    string
	cluster    string
	consumers []string
}

// New create a new instance of the partition MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {

	logp.Warn("EXPERIMENTAL: The kafka consumer metricset is experimental")

	config := struct {
		Host      string `yaml:"host"`
		Cluster   string `yaml:"cluster"`
		Consumers []string `yaml:"consumers"`
	}{}

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		client:          &http.Client{Timeout: base.Module().Config().Timeout},
		host: config.Host,
		consumers: config.Consumers,
		cluster: config.Cluster,
	}, nil
}

// Fetch partition stats list from kafka
func (m *MetricSet) Fetch() ([]common.MapStr, error) {

	events := []common.MapStr{}
	var connected_consumers []string
	var err error
	//fetch connected consumer groups if no consumers specified in the config
	if len(m.consumers) == 0 {
		debugf("No consumer groups found in config, fetching all consumer groups from Kafka")
		connected_consumers, err = fetchConsumerGroups(m)
		if err != nil {
			return nil, fmt.Errorf("Error fetching connected consumer groups from Kafka: %#v", err)
		}
	}  else {
		debugf("Fetching consumer groups from config")
		connected_consumers = m.consumers
	}
	if len(connected_consumers) == 0 {
		return nil, fmt.Errorf("No consumer groups found in Kafka")
	}
	debugf("Consumer groups to be fetched: ", connected_consumers)
	for _, consumer := range connected_consumers {
		url := "http://" + m.host + "/v2/kafka/" + m.cluster + "/consumer/" + consumer + "/lag"
		debugf("Fetching url: ", url)
		req, err := http.NewRequest("GET", url, nil)
		resp, err := m.client.Do(req)
		if err != nil {
			_ = fmt.Errorf("Error making http request: %#v", err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			_ = fmt.Errorf("HTTP error %d: %s", resp.StatusCode, resp.Status)
			continue
		}

		resp_body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			_ = fmt.Errorf("Error converting response body: %#v", err)
			continue
		}

		event, err := eventMapping(resp_body)
		if err != nil {
			continue
		}

		events = append(events, event)

	}

	return events, nil
}
