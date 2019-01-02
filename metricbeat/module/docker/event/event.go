package event

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/docker"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("docker", "event", New,
		mb.WithHostParser(docker.HostParser),
		mb.DefaultMetricSet(),
	)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	dockerClient *client.Client
	dedot        bool
	logger       *logp.Logger
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	config := docker.DefaultConfig()
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	client, err := docker.NewDockerClient(base.HostData().URI, config)
	if err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		dockerClient:  client,
		dedot:         config.DeDot,
		logger:        logp.NewLogger("docker"),
	}, nil
}

// Run listens for docker events and reports them
func (m *MetricSet) Run(reporter mb.PushReporterV2) {
	ctx, cancel := context.WithCancel(context.Background())
	options := types.EventsOptions{
		Since: fmt.Sprintf("%d", time.Now().Unix()),
	}

	for {
		events, errors := m.dockerClient.Events(ctx, options)

	WATCH:
		for {
			select {
			case event := <-events:
				m.logger.Debug("Got a new docker event: %v", event)
				m.reportEvent(reporter, event)

			case err := <-errors:
				// Restart watch call
				m.logger.Error("Error watching for docker events: %v", err)
				time.Sleep(1 * time.Second)
				break WATCH

			case <-reporter.Done():
				m.logger.Debug("docker", "event watcher stopped")
				cancel()
				return
			}
		}
	}
}

func (m *MetricSet) reportEvent(reporter mb.PushReporterV2, event events.Message) {
	time := time.Unix(event.Time, 0)

	attributes := common.MapStr{}
	for k, v := range event.Actor.Attributes {
		if m.dedot {
			k = common.DeDot(k)
		}
		attributes[k] = v
	}

	reporter.Event(mb.Event{
		Timestamp: time,
		MetricSetFields: common.MapStr{
			"status": event.Status,
			"id":     event.ID,
			"from":   event.From,
			"type":   event.Type,
			"action": event.Action,
			"actor": common.MapStr{
				"id":         event.Actor.ID,
				"attributes": attributes,
			},
			"time": time,
		},
	})
}
