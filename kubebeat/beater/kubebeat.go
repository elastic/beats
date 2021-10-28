package beater

import (
	"context"
	"fmt"
	"time"

	"github.com/elastic/beats/v7/kubebeat/config"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

// kubebeat configuration.
type kubebeat struct {
	done      chan struct{}
	config    config.Config
	client    beat.Client
	clientset *kubernetes.Clientset
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

	bt := &kubebeat{
		done:      make(chan struct{}),
		config:    c,
		clientset: clientset,
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
				},
			}
			events[i] = event
		}

		bt.client.PublishAll(events)
		logp.Info("%v events sent", len(events))
	}
}

// Stop stops kubebeat.
func (bt *kubebeat) Stop() {
	bt.client.Close()
	close(bt.done)
}
