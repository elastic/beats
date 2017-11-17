package add_kubernetes_metadata

import (
	"context"
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/ericchiang/k8s"
	corev1 "github.com/ericchiang/k8s/api/v1"
)

// PodWatcher is a controller that synchronizes Pods.
type PodWatcher struct {
	kubeClient          *k8s.Client
	syncPeriod          time.Duration
	cleanupTimeout      time.Duration
	podQueue            chan *corev1.Pod
	nodeFilter          k8s.Option
	lastResourceVersion string
	ctx                 context.Context
	stop                context.CancelFunc
	annotationCache     annotationCache
	indexers            *Indexers
}

type annotationCache struct {
	sync.RWMutex
	annotations map[string]common.MapStr
	pods        map[string]*Pod      // pod uid -> Pod
	deleted     map[string]time.Time // deleted annotations key -> last access time
}

// NewPodWatcher initializes the watcher client to provide a local state of
// pods from the cluster (filtered to the given host)
func NewPodWatcher(kubeClient *k8s.Client, indexers *Indexers, syncPeriod, cleanupTimeout time.Duration, host string) *PodWatcher {
	ctx, cancel := context.WithCancel(context.Background())
	return &PodWatcher{
		kubeClient:          kubeClient,
		indexers:            indexers,
		syncPeriod:          syncPeriod,
		cleanupTimeout:      cleanupTimeout,
		podQueue:            make(chan *corev1.Pod, 10),
		nodeFilter:          k8s.QueryParam("fieldSelector", "spec.nodeName="+host),
		lastResourceVersion: "0",
		ctx:                 ctx,
		stop:                cancel,
		annotationCache: annotationCache{
			annotations: make(map[string]common.MapStr),
			pods:        make(map[string]*Pod),
			deleted:     make(map[string]time.Time),
		},
	}
}

func (p *PodWatcher) syncPods() error {
	logp.Info("kubernetes: %s", "Performing a pod sync")
	pods, err := p.kubeClient.CoreV1().ListPods(
		p.ctx,
		"",
		p.nodeFilter,
		k8s.ResourceVersion(p.lastResourceVersion))

	if err != nil {
		return err
	}

	for _, pod := range pods.Items {
		p.podQueue <- pod
	}

	// Store last version
	p.lastResourceVersion = pods.Metadata.GetResourceVersion()

	logp.Info("kubernetes: %s", "Pod sync done")
	return nil
}

func (p *PodWatcher) watchPods() {
	for {
		logp.Info("kubernetes: %s", "Watching API for pod events")
		watcher, err := p.kubeClient.CoreV1().WatchPods(p.ctx, "", p.nodeFilter)
		if err != nil {
			//watch pod failures should be logged and gracefully failed over as metadata retrieval
			//should never stop.
			logp.Err("kubernetes: Watching API error %v", err)
			time.Sleep(time.Second)
			continue
		}

		for {
			_, pod, err := watcher.Next()
			if err != nil {
				logp.Err("kubernetes: Watching API error %v", err)
				break
			}

			p.podQueue <- pod
		}
	}
}

func (p *PodWatcher) Run() bool {
	// Start pod processing & annotations cleanup workers
	go p.worker()
	go p.cleanupWorker()

	// Make sure that events don't flow into the annotator before informer is fully set up
	// Sync initial state:
	synced := make(chan struct{})
	go func() {
		p.syncPods()
		close(synced)
	}()

	select {
	case <-time.After(timeout):
		p.Stop()
		return false
	case <-synced:
		// Watch for new changes
		go p.watchPods()
		return true
	}
}

func (p *PodWatcher) onPodAdd(pod *Pod) {
	metadata := p.indexers.GetMetadata(pod)
	p.annotationCache.Lock()
	defer p.annotationCache.Unlock()

	p.annotationCache.pods[pod.Metadata.UID] = pod

	for _, m := range metadata {
		p.annotationCache.annotations[m.Index] = m.Data

		// un-delete if it's flagged (in case of update or recreation)
		delete(p.annotationCache.deleted, m.Index)
	}
}

func (p *PodWatcher) onPodUpdate(pod *Pod) {
	oldPod := p.GetPod(pod.Metadata.UID)
	if oldPod.Metadata.ResourceVersion != pod.Metadata.ResourceVersion {
		//Process the new pod changes
		p.onPodDelete(oldPod)
		p.onPodAdd(pod)
	}
}

func (p *PodWatcher) onPodDelete(pod *Pod) {
	p.annotationCache.Lock()
	defer p.annotationCache.Unlock()

	delete(p.annotationCache.pods, pod.Metadata.UID)

	// Flag all annotations as deleted (they will be still available for a while)
	now := time.Now()
	for _, index := range p.indexers.GetIndexes(pod) {
		p.annotationCache.deleted[index] = now
	}
}

func (p *PodWatcher) worker() {
	for po := range p.podQueue {
		pod := GetPodMeta(po)
		if pod.Metadata.DeletionTimestamp != "" {
			p.onPodDelete(pod)
		} else {
			existing := p.GetPod(pod.Metadata.UID)
			if existing != nil {
				p.onPodUpdate(pod)
			} else {
				p.onPodAdd(pod)
			}
		}
	}
}

// Check annotations flagged as deleted for their last access time, fully delete
// the ones older than p.cleanupTimeout
func (p *PodWatcher) cleanupWorker() {
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
			p.annotationCache.RLock()
			for key, lastSeen := range p.annotationCache.deleted {
				if lastSeen.Before(timeout) {
					toDelete = append(toDelete, key)
				}
			}
			p.annotationCache.RUnlock()

			// Delete timed out entries:
			p.annotationCache.Lock()
			for _, key := range toDelete {
				delete(p.annotationCache.deleted, key)
				delete(p.annotationCache.annotations, key)
			}
			p.annotationCache.Unlock()
		}
	}
}

func (p *PodWatcher) GetMetaData(arg string) common.MapStr {
	p.annotationCache.RLock()
	meta, ok := p.annotationCache.annotations[arg]
	var deleted bool
	if ok {
		_, deleted = p.annotationCache.deleted[arg]
	}
	p.annotationCache.RUnlock()

	// Update deleted last access
	if deleted {
		p.annotationCache.Lock()
		p.annotationCache.deleted[arg] = time.Now()
		p.annotationCache.Unlock()
	}

	if ok {
		return meta
	}

	return nil
}

func (p *PodWatcher) GetPod(uid string) *Pod {
	p.annotationCache.RLock()
	defer p.annotationCache.RUnlock()
	return p.annotationCache.pods[uid]
}

func (p *PodWatcher) Stop() {
	p.stop()
	close(p.podQueue)
}
