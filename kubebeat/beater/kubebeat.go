package beater

import (
	"bytes"
	"context"
	"fmt"
	"github.com/elastic/beats/v7/kubebeat/bundle"
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
		events := make([]beat.Event, len(pods))
		timestamp := time.Now()

		for i, p := range pods {
			pod, ok := p.(*kubernetes.Pod)
			if !ok {
				logp.Info("could not convert to pod")
			}

			pod.SetManagedFields(nil)
			pod.Status.Reset()

			result, err := bt.Decision(pod)
			if err != nil {
				result = map[string]interface{}{"err": err.Error()}
			}

			event := beat.Event{
				Timestamp: timestamp,
				Fields: common.MapStr{
					"type":             b.Info.Name,
					"uid":              string(pod.UID), // UID is an alias to string
					"name":             pod.Name,
					"namespace":        pod.Namespace,
					"ip":               pod.Status.PodIP,
					"phase":            string(pod.Status.Phase),
					"service_account":  pod.Spec.ServiceAccountName,
					"node_name":        pod.Spec.NodeName,
					"security_context": pod.Spec.SecurityContext,
					"result":           result,
				},
			}
			events[i] = event
		}

		bt.client.PublishAll(events)
		logp.Info("%v events sent", len(events))
	}
}

func (bt *kubebeat) Decision(input interface{}) (interface{}, error) {
	// get the named policy decision for the specified input
	result, err := bt.opa.Decision(context.Background(), sdk.DecisionOptions{
		Path:  "/authz/allow",
		Input: map[string]interface{}{"open": "sesame"},
		//Input: input,
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
