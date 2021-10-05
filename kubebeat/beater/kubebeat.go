package beater

import (
	"context"
	"fmt"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/elastic/beats/v7/kubebeat/config"
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
		return nil, fmt.Errorf("Error reading config file: %v", err)
	}

	// could we later use code from gatekeeper/kube-mgmt?
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

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

	var err error
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

		pods, err := bt.clientset.CoreV1().Pods("kube-system").List(context.TODO(),
			metav1.ListOptions{
				LabelSelector: "tier=control-plane",
			})
		timestamp := time.Now()
		if err != nil {
			panic(err.Error())
		}

		events := make([]beat.Event, len(pods.Items))

		for _, item := range pods.Items {

			item.SetManagedFields(nil)
			item.Status.Reset()

			event := beat.Event{
				Timestamp: timestamp,
				Fields: common.MapStr{
					"type": b.Info.Name,
					"pod":  item,
				},
			}
			events = append(events, event)
		}

		bt.client.PublishAll(events)
		logp.Info("Events sent")
	}
}

// Stop stops kubebeat.
func (bt *kubebeat) Stop() {
	bt.client.Close()
	close(bt.done)
}
