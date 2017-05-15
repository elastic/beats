package event

import (
	"context"
	"encoding/json"
	"time"

	"github.com/elastic/beats/libbeat/logp"

	"github.com/ericchiang/k8s"
	corev1 "github.com/ericchiang/k8s/api/v1"
)

// Watcher is a controller that synchronizes Pods.
type Watcher struct {
	kubeClient          *k8s.Client
	namespace           string
	syncPeriod          time.Duration
	eventQueue          chan *Event
	lastResourceVersion string
	ctx                 context.Context
	stop                context.CancelFunc
}

// NewWatcher initializes the watcher client to provide a local state of
// pods from the cluster (filtered to the given host)
func NewWatcher(kubeClient *k8s.Client, syncPeriod time.Duration, namespace string) *Watcher {
	ctx, cancel := context.WithCancel(context.Background())
	return &Watcher{
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
func (w *Watcher) watchEvents() {
	for {
		//To avoid writing old events, list events to get last resource version
		events, err := w.kubeClient.CoreV1().ListEvents(
			w.ctx,
			w.namespace,
		)

		if err != nil {
			//if listing fails try again after sometime
			logp.Err("kubernetes: List API error %v", err)
			// Sleep for a second to prevent API server from being bombarded
			// API server could be down
			time.Sleep(time.Second)
			continue
		}

		w.lastResourceVersion = events.Metadata.GetResourceVersion()

		logp.Info("kubernetes: %s", "Watching API for events")
		watcher, err := w.kubeClient.CoreV1().WatchEvents(
			w.ctx,
			w.namespace,
			k8s.ResourceVersion(w.lastResourceVersion),
		)
		if err != nil {
			//watch events failures should be logged and gracefully failed over as metadata retrieval
			//should never stop.
			logp.Err("kubernetes: Watching API eror %v", err)
			// Sleep for a second to prevent API server from being bombarded
			// API server could be down
			time.Sleep(time.Second)
			continue
		}

		for {
			_, eve, err := watcher.Next()
			if err != nil {
				logp.Err("kubernetes: Watching API error %v", err)
				break
			}

			event := w.getEventMeta(eve)
			if event != nil {
				w.eventQueue <- event
			}

		}
	}

}

func (w *Watcher) Run() {
	// Start watching on events
	go w.watchEvents()
}

func (w *Watcher) getEventMeta(pod *corev1.Event) *Event {
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

func (w *Watcher) Stop() {
	w.stop()
	close(w.eventQueue)
}
