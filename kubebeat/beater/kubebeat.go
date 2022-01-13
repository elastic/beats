package beater

import (
	"context"
	"fmt"
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

	eventParser, err := NewEvaluationResultParser()
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

	// ticker := time.NewTicker(bt.config.Period)
	output := bt.data.Output()

	for {
		select {
		case <-bt.done:
			return nil
		case o := <-output:
			timestamp := time.Now()
			runId, _ := uuid.NewV4()

			resourceCallback := func(resource interface{}) {
				// ns will be passed as param from fleet on https://github.com/elastic/security-team/issues/2383 and it's user configurable
				ns := ""
				bt.resourceIteration(config.Datastream(ns), resource, runId, timestamp)
			}

			bt.scheduler.ScheduleResources(o, resourceCallback)
		}
	}
}

// Todo - index param implemented as part of resource iteration will be added to code polishing to have proper infra
func (bt *kubebeat) resourceIteration(index, resource interface{}, runId uuid.UUID, timestamp time.Time) {

	result, err := bt.eval.Decision(resource)
	if err != nil {
		logp.Error(fmt.Errorf("error running the policy: %w", err))
		return
	}
	//Todo index added to Parse result since currently that's where event fields are appended - Should later be split on some critiria(stream,fetcher or datasource) with an util function to handle the ds provision
	events, err := bt.resultParser.ParseResult(index, result, runId, timestamp)

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
