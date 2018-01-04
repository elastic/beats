package kubernetes

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/common/bus"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/ericchiang/k8s"
	corev1 "github.com/ericchiang/k8s/api/v1"
)

// Watcher reads Kubernetes events and keeps a list of known pods
type Watcher interface {
	// Start watching Kubernetes API for new containers
	Start() error

	// Stop watching Kubernetes API for new containers
	Stop()

	// ListenStart returns a bus listener to receive pod started events, with a `pod` key holding it
	ListenStart() bus.Listener

	// ListenUpdate returns a bus listener to receive pod updated events, with a `pod` key holding it
	ListenUpdate() bus.Listener

	// ListenStop returns a bus listener to receive pod stopped events, with a `pod` key holding it
	ListenStop() bus.Listener
}

type podWatcher struct {
	sync.RWMutex
	client              Client
	syncPeriod          time.Duration
	cleanupTimeout      time.Duration
	nodeFilter          k8s.Option
	lastResourceVersion string
	ctx                 context.Context
	stop                context.CancelFunc
	bus                 bus.Bus
	pods                map[string]*Pod      // pod id -> Pod
	deleted             map[string]time.Time // deleted annotations key -> last access time
}

// Client for Kubernetes interface
type Client interface {
	ListPods(ctx context.Context, namespace string, options ...k8s.Option) (*corev1.PodList, error)
	WatchPods(ctx context.Context, namespace string, options ...k8s.Option) (*k8s.CoreV1PodWatcher, error)
}

// NewWatcher initializes the watcher client to provide a local state of
// pods from the cluster (filtered to the given host)
func NewWatcher(client Client, syncPeriod, cleanupTimeout time.Duration, host string) Watcher {
	ctx, cancel := context.WithCancel(context.Background())
	return &podWatcher{
		client:              client,
		cleanupTimeout:      cleanupTimeout,
		syncPeriod:          syncPeriod,
		nodeFilter:          k8s.QueryParam("fieldSelector", "spec.nodeName="+host),
		lastResourceVersion: "0",
		ctx:                 ctx,
		stop:                cancel,
		pods:                make(map[string]*Pod),
		deleted:             make(map[string]time.Time),
		bus:                 bus.New("kubernetes"),
	}
}

func (p *podWatcher) syncPods() error {
	logp.Info("kubernetes: %s", "Performing a pod sync")
	pods, err := p.client.ListPods(
		p.ctx,
		"",
		p.nodeFilter,
		k8s.ResourceVersion(p.lastResourceVersion))

	if err != nil {
		return err
	}

	p.Lock()
	for _, apiPod := range pods.Items {
		pod := GetPod(apiPod)
		p.pods[pod.Metadata.UID] = pod
	}
	p.Unlock()

	// Emit all start events (avoid blocking if the bus get's blocked)
	go func() {
		for _, pod := range p.pods {
			p.bus.Publish(bus.Event{
				"start": true,
				"pod":   pod,
			})
		}
	}()

	// Store last version
	p.lastResourceVersion = pods.Metadata.GetResourceVersion()

	logp.Info("kubernetes: %s", "Pod sync done")
	return nil
}

// Start watching pods
func (p *podWatcher) Start() error {

	// Make sure that events don't flow into the annotator before informer is fully set up
	// Sync initial state:
	synced := make(chan struct{})
	go func() {
		p.syncPods()
		close(synced)
	}()

	select {
	case <-time.After(p.syncPeriod):
		p.Stop()
		return errors.New("Timeout while doing initial Kubernetes pods sync")
	case <-synced:
		// Watch for new changes
		go p.watch()
		go p.cleanupWorker()
		return nil
	}
}

func (p *podWatcher) watch() {
	for {
		logp.Info("kubernetes: %s", "Watching API for pod events")
		watcher, err := p.client.WatchPods(p.ctx, "", p.nodeFilter)
		if err != nil {
			//watch pod failures should be logged and gracefully failed over as metadata retrieval
			//should never stop.
			logp.Err("kubernetes: Watching API error %v", err)
			time.Sleep(time.Second)
			continue
		}

		for {
			_, apiPod, err := watcher.Next()
			if err != nil {
				logp.Err("kubernetes: Watching API error %v", err)
				watcher.Close()
				break
			}

			pod := GetPod(apiPod)
			if pod.Metadata.DeletionTimestamp != "" {
				// Pod deleted
				p.Lock()
				p.deleted[pod.Metadata.UID] = time.Now()
				p.Unlock()

			} else {
				if p.Pod(pod.Metadata.UID) != nil {
					// Pod updated
					p.Lock()
					p.pods[pod.Metadata.UID] = pod
					// un-delete if it's flagged (in case of update or recreation)
					delete(p.deleted, pod.Metadata.UID)
					p.Unlock()

					p.bus.Publish(bus.Event{
						"update": true,
						"pod":    pod,
					})

				} else {
					// Pod added
					p.Lock()
					p.pods[pod.Metadata.UID] = pod
					// un-delete if it's flagged (in case of update or recreation)
					delete(p.deleted, pod.Metadata.UID)
					p.Unlock()

					p.bus.Publish(bus.Event{
						"start": true,
						"pod":   pod,
					})
				}
			}
		}
	}
}

// Check annotations flagged as deleted for their last access time, fully delete
// the ones older than p.cleanupTimeout
func (p *podWatcher) cleanupWorker() {
	for {
		// Wait a full period
		time.Sleep(p.cleanupTimeout)

		select {
		case <-p.ctx.Done():
			return
		default:
			// Check entries for timeout
			var toDelete []string
			timeout := time.Now().Add(-p.cleanupTimeout)
			p.RLock()
			for key, lastSeen := range p.deleted {
				if lastSeen.Before(timeout) {
					toDelete = append(toDelete, key)
				}
			}
			p.RUnlock()

			// Delete timed out entries:
			p.Lock()
			for _, key := range toDelete {
				p.bus.Publish(bus.Event{
					"stop": true,
					"pod":  p.Pod(key),
				})

				delete(p.deleted, key)
				delete(p.pods, key)
			}
			p.Unlock()
		}
	}
}

func (p *podWatcher) Pod(uid string) *Pod {
	p.RLock()
	pod := p.pods[uid]
	_, deleted := p.deleted[uid]
	p.RUnlock()

	// Update deleted last access
	if deleted {
		p.Lock()
		p.deleted[uid] = time.Now()
		p.Unlock()
	}

	return pod
}

// ListenStart returns a bus listener to receive pod started events, with a `pod` key holding it
func (p *podWatcher) ListenStart() bus.Listener {
	return p.bus.Subscribe("start")
}

// ListenStop returns a bus listener to receive pod stopped events, with a `pod` key holding it
func (p *podWatcher) ListenStop() bus.Listener {
	return p.bus.Subscribe("stop")
}

// ListenUpdate returns a bus listener to receive updated pod events, with a `pod` key holding it
func (p *podWatcher) ListenUpdate() bus.Listener {
	return p.bus.Subscribe("update")
}

func (p *podWatcher) Stop() {
	p.stop()
}
