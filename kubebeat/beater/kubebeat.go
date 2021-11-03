package beater

import (
	"bytes"
	"context"
	"fmt"
	"github.com/elastic/beats/v7/kubebeat/bundle"
	"github.com/mitchellh/mapstructure"
	"time"

	"github.com/elastic/beats/v7/kubebeat/config"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/kubernetes"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/open-policy-agent/opa/sdk"
	sdktest "github.com/open-policy-agent/opa/sdk/test"
)

// kubebeat configuration.
type kubebeat struct {
	done         chan struct{}
	config       config.Config
	client       beat.Client
	watcher      kubernetes.Watcher
	opa          *sdk.OPA
	bundleServer *sdktest.Server
}

// New creates an instance of kubebeat.
func New(b *beat.Beat, cfg *common.Config) (beat.Beater, error) {
	c := config.DefaultConfig
	if err := cfg.Unpack(&c); err != nil {
		return nil, fmt.Errorf("error reading config file: %v", err)
	}

	logp.Info("Config initiated.")

	client, err := kubernetes.GetKubernetesClient(c.KubeConfig)
	if err != nil {
		return nil, fmt.Errorf("fail to get k8sclient client: %s", err.Error())
	}

	logp.Info("Client initiated.")

	watchOptions := kubernetes.WatchOptions{
		SyncTimeout: c.Period,
		Namespace:   "kube-system",
	}

	watcher, err := kubernetes.NewWatcher(client, &kubernetes.Pod{}, watchOptions, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating k8s client set: %v", err)
	}

	logp.Info("Watcher initiated.")

	// create a mock HTTP bundle bundleServer
	bundleServer, err := sdktest.NewServer(sdktest.MockBundle("/bundles/bundle.tar.gz", bundle.Policies))
	if err != nil {
		return nil, fmt.Errorf("fail to init bundle server: %s", err.Error())
	}

	// provide the OPA configuration which specifies
	// fetching policy bundles from the mock bundleServer
	// and logging decisions locally to the console
	config := []byte(fmt.Sprintf(bundle.Config, bundleServer.URL()))

	// create an instance of the OPA object
	opa, err := sdk.New(context.Background(), sdk.Options{
		Config: bytes.NewReader(config),
	})
	if err != nil {
		return nil, fmt.Errorf("fail to init opa: %s", err.Error())
	}

	bt := &kubebeat{
		done:         make(chan struct{}),
		config:       c,
		opa:          opa,
		bundleServer: bundleServer,
		watcher:      watcher,
	}
	return bt, nil
}

type PolicyResult map[string]RuleResult

type RuleResult struct {
	Findings []Finding `json:"findings"`
}

type Finding struct {
	Compliant bool        `json:"compliant""`
	Message   string      `json:"message"`
	Resource  interface{} `json:"resource"`
}

// Run starts kubebeat.
func (bt *kubebeat) Run(b *beat.Beat) error {
	logp.Info("kubebeat is running! Hit CTRL-C to stop it.")

	err := bt.watcher.Start()
	if err != nil {
		return err
	}

	bt.client, err = b.Publisher.Connect()
	if err != nil {
		return err
	}

	ticker := time.NewTicker(bt.config.Period)
	for {
		select {
		case <-bt.done:
			return nil
		case <-ticker.C:
		}

		pods := bt.watcher.Store().List()
		events := make([]beat.Event, 0)
		timestamp := time.Now()

		for _, p := range pods {
			pod, ok := p.(*kubernetes.Pod)
			if !ok {
				logp.Info("could not convert to pod")
				continue
			}
			pod.SetManagedFields(nil)
			pod.Status.Reset()
			pod.Kind = "Pod" // see https://github.com/kubernetes/kubernetes/issues/3030

			result, err := bt.Decision(pod)
			if err != nil {
				errEvent := beat.Event{
					Timestamp: timestamp,
					Fields: common.MapStr{
						"type":     b.Info.Name,
						"err":      fmt.Errorf("error running the policy: %v", err.Error()),
						"resource": pod,
					},
				}
				events = append(events, errEvent)
				continue
			}

			var decoded PolicyResult
			err = mapstructure.Decode(result, &decoded)
			if err != nil {
				errEvent := beat.Event{
					Timestamp: timestamp,
					Fields: common.MapStr{
						"type":       b.Info.Name,
						"err":        fmt.Errorf("error parsing the policy result: %v", err.Error()),
						"resource":   pod,
						"raw_result": result,
					},
				}
				events = append(events, errEvent)
				continue
			}

			for ruleName, ruleResult := range decoded {
				for _, Finding := range ruleResult.Findings {
					event := beat.Event{
						Timestamp: timestamp,
						Fields: common.MapStr{
							"type":      b.Info.Name,
							"rule_id":      ruleName,
							"compliant": Finding.Compliant,
							"resource":  Finding.Resource,
							"message":   Finding.Message,
						},
					}
					events = append(events, event)
				}
			}

		}

		bt.client.PublishAll(events)
		logp.Info("%v events sent", len(events))
	}
}

func (bt *kubebeat) Decision(input interface{}) (interface{}, error) {
	// get the named policy decision for the specified input
	result, err := bt.opa.Decision(context.Background(), sdk.DecisionOptions{
		Path:  "/compliance",
		Input: input,
	})
	if err != nil {
		return nil, err
	}

	return result.Result, nil
}

// Stop stops kubebeat.
func (bt *kubebeat) Stop() {
	bt.client.Close()
	bt.opa.Stop(context.Background())
	bt.bundleServer.Stop()

	close(bt.done)
}
