package beater

import (
	"context"
	"fmt"
	libevents "github.com/elastic/beats/v7/libbeat/beat/events"
	"time"

	"github.com/elastic/beats/v7/kubebeat/config"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/gofrs/uuid"
)

// kubebeat configuration.
type kubebeat struct {
	done         chan struct{}
	config       config.Config
	client       beat.Client
	eval         *evaluator
	data         *Data
	resultParser *evaluationResultParser
	scheduler    ResourceScheduler
}

const (
	cycleStatusStart = "start"
	cycleStatusEnd   = "end"
	cycleStatusFail  = "fail"
)

// New creates an instance of kubebeat.
func New(b *beat.Beat, cfg *common.Config) (beat.Beater, error) {
	ctx := context.Background()

	c := config.DefaultConfig
	if err := cfg.Unpack(&c); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	logp.Info("Config initiated.")

	data := NewData(ctx, c.Period)
	scheduler := NewSynchronousScheduler()
	evaluator, err := NewEvaluator()
	if err != nil {
		return nil, err
	}

	// namespace will be passed as param from fleet on https://github.com/elastic/security-team/issues/2383 and it's user configurable
	resultsIndex := config.Datastream("", config.ResultsDatastreamIndexPrefix)
	eventParser, err := NewEvaluationResultParser(resultsIndex)
	if err != nil {
		return nil, err
	}

	kubef, err := NewKubeFetcher(c.KubeConfig, c.Period)
	if err != nil {
		return nil, err
	}

	data.RegisterFetcher("kube_api", kubef)
	data.RegisterFetcher("processes", NewProcessesFetcher(procfsdir))
	data.RegisterFetcher("file_system", NewFileFetcher(c.Files))

	bt := &kubebeat{
		done:         make(chan struct{}),
		config:       c,
		eval:         evaluator,
		data:         data,
		resultParser: eventParser,
		scheduler:    scheduler,
	}
	return bt, nil
}

// Run starts kubebeat.
func (bt *kubebeat) Run(b *beat.Beat) error {
	logp.Info("kubebeat is running! Hit CTRL-C to stop it.")

	err := bt.data.Run()
	if err != nil {
		return err
	}
	defer bt.data.Stop()

	if bt.client, err = b.Publisher.Connect(); err != nil {
		return err
	}

	output := bt.data.Output()

	for {
		select {
		case <-bt.done:
			return nil
		case o := <-output:
			cycleId, _ := uuid.NewV4()
			// update hidden-index that the beat's cycle has started
			bt.updateCycleStatus(cycleId, cycleStatusStart)

			resourceCallback := func(resource interface{}) {
				bt.resourceIteration(resource, cycleId)
			}

			bt.scheduler.ScheduleResources(o, resourceCallback)

			// update hidden-index that the beat's cycle has ended
			bt.updateCycleStatus(cycleId, cycleStatusEnd)
		}
	}
}

func (bt *kubebeat) resourceIteration(resource interface{}, cycleId uuid.UUID) {
	result, err := bt.eval.Decision(resource)
	if err != nil {
		logp.Error(fmt.Errorf("error running the policy: %w", err))
		return
	}
	events, err := bt.resultParser.ParseResult(result, cycleId)

	if err != nil {
		logp.Error(fmt.Errorf("error running the policy: %w", err))
		return
	}

	bt.client.PublishAll(events)
}

// Stop stops kubebeat.
func (bt *kubebeat) Stop() {
	bt.client.Close()
	bt.eval.Stop()

	close(bt.done)
}

// updateCycleStatus updates beat status in metadata ES index.
func (bt *kubebeat) updateCycleStatus(cycleId uuid.UUID, status string) {
	metadataIndex := config.Datastream("", config.MetadataDatastreamIndexPrefix)
	cycleEndedEvent := beat.Event{
		Timestamp: time.Now(),
		Meta:      common.MapStr{libevents.FieldMetaIndex: metadataIndex},
		Fields: common.MapStr{
			"cycle_id": cycleId,
			"status":   status,
		},
	}
	bt.client.Publish(cycleEndedEvent)
}
