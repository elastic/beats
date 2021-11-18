package beater

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"github.com/elastic/beats/v7/kubebeat/bundle"
	"github.com/gofrs/uuid"
	"github.com/mitchellh/mapstructure"
	"io/fs"
	"log"
	"os"
	"strings"
	"time"

	"github.com/elastic/beats/v7/kubebeat/config"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/open-policy-agent/opa/sdk"
	sdktest "github.com/open-policy-agent/opa/sdk/test"
)

// kubebeat configuration.
type kubebeat struct {
	done         chan struct{}
	config       config.Config
	client       beat.Client
	opa          *sdk.OPA
	bundleServer *sdktest.Server
	data         *Data
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

	kubef, err := NewKubeFetcher(c.KubeConfig, c.Period)
	if err != nil {
		return nil, err
	}

	data.RegisterFetcher("kube_api", kubef)
	data.RegisterFetcher("processes", NewProcessesFetcher(procfsdir))
	data.RegisterFetcher("file_system", NewFileFetcher(c.Files))

	policies := CreateCISPolicy(bundle.EmbeddedPolicy)
	// create a mock HTTP bundle bundleServer
	bundleServer, err := sdktest.NewServer(sdktest.MockBundle("/bundles/bundle.tar.gz", policies))
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
		data:         data,
	}
	return bt, nil
}

func CreateCISPolicy(fileSystem embed.FS) map[string]string {

	policies := make(map[string]string)

	fs.WalkDir(fileSystem, ".", func(filepath string, info os.DirEntry, err error) error {
		if err != nil {
			log.Fatal(err)
		}
		if info.IsDir() == false && strings.HasSuffix(info.Name(), ".rego") && !strings.HasSuffix(info.Name(), "test.rego") {

			data, err := fs.ReadFile(fileSystem, filepath)
			if err == nil {
				policies[filepath] = string(data)
			}
		}
		return nil
	})

	return policies
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
			events := make([]beat.Event, 0)
			timestamp := time.Now()

			result, err := bt.Decision(o)
			if err != nil {
				errEvent := beat.Event{
					Timestamp: timestamp,
					Fields: common.MapStr{
						"type":     b.Info.Name,
						"err":      fmt.Errorf("error running the policy: %v", err.Error()),
						"resource": o,
					},
				}
				events = append(events, errEvent)
			} else {
				var decoded PolicyResult
				err = mapstructure.Decode(result, &decoded)
				if err != nil {
					errEvent := beat.Event{
						Timestamp: timestamp,
						Fields: common.MapStr{
							"type":       b.Info.Name,
							"err":        fmt.Errorf("error parsing the policy result: %v", err.Error()),
							"resource":   o,
							"raw_result": result,
						},
					}
					events = append(events, errEvent)
				} else {
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
				}
			}

			bt.client.PublishAll(events)
			logp.Info("%v events sent", len(events))
		}
	}
}

func (bt *kubebeat) Decision(input interface{}) (interface{}, error) {
	// get the named policy decision for the specified input
	result, err := bt.opa.Decision(context.Background(), sdk.DecisionOptions{
		Path:  "main",
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
