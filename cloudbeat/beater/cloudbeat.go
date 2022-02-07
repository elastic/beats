package beater

import (
	"context"
	"fmt"
	"time"

	"github.com/elastic/beats/v7/cloudbeat/config"
	"github.com/elastic/beats/v7/cloudbeat/opa"
	_ "github.com/elastic/beats/v7/cloudbeat/processor" // Add cloudbeat default processors.
	"github.com/elastic/beats/v7/cloudbeat/resources"
	"github.com/elastic/beats/v7/cloudbeat/resources/conditions"
	"github.com/elastic/beats/v7/cloudbeat/resources/fetchers"
	"github.com/elastic/beats/v7/libbeat/beat"
	libevents "github.com/elastic/beats/v7/libbeat/beat/events"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/kubernetes"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/processors"

	"github.com/gofrs/uuid"
)

// cloudbeat configuration.
type cloudbeat struct {
	ctx    context.Context
	cancel context.CancelFunc

	config       config.Config
	client       beat.Client
	data         *resources.Data
	eval         *opa.Evaluator
	resultParser *opa.EvaluationResultParser
	scheduler    ResourceScheduler
}

const (
	cycleStatusStart = "start"
	cycleStatusEnd   = "end"
	processesDir     = "/hostfs"
	cycleStatusFail  = "fail"
)

// New creates an instance of cloudbeat.
func New(b *beat.Beat, cfg *common.Config) (beat.Beater, error) {
	ctx, cancel := context.WithCancel(context.Background())

	c := config.DefaultConfig
	if err := cfg.Unpack(&c); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	logp.Info("Config initiated.")

	fetchersRegistry, err := InitRegistry(ctx, c)
	if err != nil {
		return nil, err
	}

	data, err := resources.NewData(c.Period, fetchersRegistry)
	if err != nil {
		return nil, err
	}

	scheduler := NewSynchronousScheduler()
	evaluator, err := opa.NewEvaluator(ctx)
	if err != nil {
		return nil, err
	}

	// namespace will be passed as param from fleet on https://github.com/elastic/security-team/issues/2383 and it's user configurable
	resultsIndex := config.Datastream("", config.ResultsDatastreamIndexPrefix)
	eventParser, err := opa.NewEvaluationResultParser(resultsIndex)
	if err != nil {
		return nil, err
	}

	bt := &cloudbeat{
		ctx:          ctx,
		cancel:       cancel,
		config:       c,
		eval:         evaluator,
		data:         data,
		resultParser: eventParser,
		scheduler:    scheduler,
	}
	return bt, nil
}

// Run starts cloudbeat.
func (bt *cloudbeat) Run(b *beat.Beat) error {
	logp.Info("cloudbeat is running! Hit CTRL-C to stop it.")

	if err := bt.data.Run(bt.ctx); err != nil {
		return err
	}

	procs, err := bt.configureProcessors(bt.config.Processors)
	if err != nil {
		return err
	}

	// Connect publisher (with beat's processors)
	if bt.client, err = b.Publisher.ConnectWith(beat.ClientConfig{
		Processing: beat.ProcessingConfig{
			Processor: procs,
		},
	}); err != nil {
		return err
	}

	output := bt.data.Output()

	for {
		select {
		case <-bt.ctx.Done():
			return nil
		case o := <-output:
			cycleId, _ := uuid.NewV4()
			// update hidden-index that the beat's cycle has started
			bt.updateCycleStatus(cycleId, cycleStatusStart)

			resourceCallback := func(resource interface{}) {
				bt.resourceIteration(bt.ctx, resource, cycleId)
			}

			bt.scheduler.ScheduleResources(o, resourceCallback)

			// update hidden-index that the beat's cycle has ended
			bt.updateCycleStatus(cycleId, cycleStatusEnd)
		}
	}
}

func InitRegistry(ctx context.Context, c config.Config) (resources.FetchersRegistry, error) {
	registry := resources.NewFetcherRegistry()

	kubeCfg := fetchers.KubeApiFetcherConfig{
		Kubeconfig: c.KubeConfig,
		Interval:   c.Period,
	}
	kubef, err := fetchers.NewKubeFetcher(kubeCfg)
	if err != nil {
		return nil, err
	}

	client, err := kubernetes.GetKubernetesClient("", kubernetes.KubeClientOptions{})
	if err != nil {
		return nil, err
	}

	leaseProvider := conditions.NewLeaderLeaseProvider(ctx, client)
	condition := conditions.NewLeaseFetcherCondition(leaseProvider)

	if err = registry.Register("kube_api", kubef, condition); err != nil {
		return nil, err
	}

	procCfg := fetchers.ProcessFetcherConfig{
		Directory: processesDir,
	}
	procf := fetchers.NewProcessesFetcher(procCfg)

	if err = registry.Register("processes", procf); err != nil {
		return nil, err
	}

	fileCfg := fetchers.FileFetcherConfig{
		Patterns: c.Files,
	}
	filef := fetchers.NewFileFetcher(fileCfg)

	if err = registry.Register("file_system", filef); err != nil {
		return nil, err
	}

	return registry, nil
}

func (bt *cloudbeat) resourceIteration(ctx context.Context, resource interface{}, cycleId uuid.UUID) {
	result, err := bt.eval.Decision(ctx, resource)
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

// Stop stops cloudbeat.
func (bt *cloudbeat) Stop() {
	bt.data.Stop(bt.ctx, bt.cancel)
	bt.eval.Stop(bt.ctx)

	bt.client.Close()
}

// updateCycleStatus updates beat status in metadata ES index.
func (bt *cloudbeat) updateCycleStatus(cycleId uuid.UUID, status string) {
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

// configureProcessors configure processors to be used by the beat
func (bt *cloudbeat) configureProcessors(processorsList processors.PluginConfig) (procs *processors.Processors, err error) {
	return processors.New(processorsList)
}
