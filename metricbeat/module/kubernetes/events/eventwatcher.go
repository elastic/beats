package events

import (
	"context"
	"encoding/json"
	"time"

	"github.com/elastic/beats/libbeat/logp"

	"github.com/ericchiang/k8s"
	corev1 "github.com/ericchiang/k8s/api/v1"
)

// EventWatcher is a controller that synchronizes Pods.
type EventWatcher struct {
	kubeClient          *k8s.Client
	namespace           string
	syncPeriod          time.Duration
	eventQueue          chan *Event
	lastResourceVersion string
	ctx                 context.Context
	stop                context.CancelFunc
}

// NewEventWatcher initializes the watcher client to provide a local state of
// pods from the cluster (filtered to the given host)
func NewEventWatcher(kubeClient *k8s.Client, syncPeriod time.Duration, namespace string) *EventWatcher {
	ctx, cancel := context.WithCancel(context.Background())
	return &EventWatcher{
		kubeClient:          kubeClient,
		namespace:           namespace,
		syncPeriod:          syncPeriod,
		eventQueue:          make(chan *Event, 10),
		lastResourceVersion: "0",
		ctx:                 ctx,
		stop:                cancel,
	}
}

// watchEvents watches on the Kubernetes API server and puts them onto a channel.
// watchEvents only starts from the most recent event.
func (p *EventWatcher) watchEvents() {
	for {
		//To avoid writing old events, list events to get last resource version
		events, err := p.kubeClient.CoreV1().ListEvents(
			p.ctx,
			p.namespace,
		)

		if err != nil {
			//if listing fails try again after sometime
			logp.Err("kubernetes: List API error %v", err)
			time.Sleep(time.Second)
			continue
		}

		p.lastResourceVersion = events.Metadata.GetResourceVersion()

		logp.Info("kubernetes: %s", "Watching API for events")
		watcher, err := p.kubeClient.CoreV1().WatchEvents(
			p.ctx,
			p.namespace,
			k8s.ResourceVersion(p.lastResourceVersion),
		)
		if err != nil {
			//watch events failures should be logged and gracefully failed over as metadata retrieval
			//should never stop.
			logp.Err("kubernetes: Watching API eror %v", err)
			time.Sleep(time.Second)
			continue
		}

		for {
			_, eve, err := watcher.Next()
			if err != nil {
				logp.Err("kubernetes: Watching API error %v", err)
				break
			}

			event := p.getEventMeta(eve)
			if event != nil {
				p.eventQueue <- event
			}

		}
	}

}

func (p *EventWatcher) Run() {
	// Start watching on events
	go p.watchEvents()
}

func (p *EventWatcher) getEventMeta(pod *corev1.Event) *Event {
	bytes, err := json.Marshal(pod)
	if err != nil {
		logp.Warn("Unable to marshal %v", pod.String())
		return nil
	}

	eve := &Event{}
	err = json.Unmarshal(bytes, eve)
	if err != nil {
		logp.Warn("Unable to marshal %v", pod.String())
		return nil
	}

	return eve

}

func (p *EventWatcher) Stop() {
	p.stop()
	close(p.eventQueue)
}
