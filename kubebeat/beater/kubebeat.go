package beater

import (
	"context"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/mitchellh/mapstructure"

	"github.com/elastic/beats/v7/kubebeat/config"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

// kubebeat configuration.
type kubebeat struct {
	done   chan struct{}
	config config.Config
	client beat.Client
	eval   *evaluator
	data   *Data
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
	evaluator, err := NewEvaluator()
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
		done:   make(chan struct{}),
		config: c,
		eval:   evaluator,
		data:   data,
	}
	return bt, nil
}

type PolicyResult map[string]RuleResult

type RuleResult struct {
	Findings []Finding   `json:"findings"`
	Resource interface{} `json:"resource"`
}

type Finding struct {
	Result interface{} `json:"result"`
	Rule   interface{} `json:"rule"`
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
	runId, err := uuid.NewV4()

	for {
		select {
		case <-bt.done:
			return nil
		case o := <-output:
			bt.resourceIteration(runId, o)
		}
	}
}

func (bt *kubebeat) resourceIteration(runId uuid.UUID, resource interface{}) {
	events := make([]beat.Event, 0)
	timestamp := time.Now()

	result, err := bt.eval.Decision(resource)
	if err != nil {
		logp.Error(fmt.Errorf("error running the policy: %w", err))
		return
	}

	var decoded PolicyResult
	err = mapstructure.Decode(result, &decoded)
	if err != nil {
		logp.Error(fmt.Errorf("error parsing the policy result: %w", err))
		return
	}

	for _, ruleResult := range decoded {
		for _, Finding := range ruleResult.Findings {
			event := beat.Event{
				Timestamp: timestamp,
				Fields: common.MapStr{
					"run_id":   runId,
					"result":   Finding.Result,
					"resource": ruleResult.Resource,
					"rule":     Finding.Rule,
				},
			}
			events = append(events, event)
		}
	}

	bt.client.PublishAll(events)
	logp.Info("%v events sent", len(events))
}

// Stop stops kubebeat.
func (bt *kubebeat) Stop() {
	bt.client.Close()
	bt.eval.Stop()

	close(bt.done)
}
