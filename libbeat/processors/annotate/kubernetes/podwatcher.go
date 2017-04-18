package kubernetes

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
	podQueue            chan *corev1.Pod
	nodeFilter          k8s.Option
	lastResourceVersion string
	ctx                 context.Context
	stop                context.CancelFunc
	annotationCache     annotationCache
	indexers            *Indexers
}

type annotationCache struct {
	sync.Mutex
	annotations map[string]common.MapStr
	pods        map[string]*corev1.Pod // pod uid -> Pod
}

type NodeOption struct{}

// NewPodWatcher initializes the watcher client to provide a local state of
// pods from the cluster (filtered to the given host)
func NewPodWatcher(kubeClient *k8s.Client, indexers *Indexers, syncPeriod time.Duration, host string) *PodWatcher {
	ctx, cancel := context.WithCancel(context.Background())
	return &PodWatcher{
		kubeClient:          kubeClient,
		indexers:            indexers,
		syncPeriod:          syncPeriod,
		podQueue:            make(chan *corev1.Pod, 10),
		nodeFilter:          k8s.QueryParam("fieldSelector", "spec.nodeName="+host),
		lastResourceVersion: "0",
		ctx:                 ctx,
		stop:                cancel,
		annotationCache: annotationCache{
			annotations: make(map[string]common.MapStr),
			pods:        make(map[string]*corev1.Pod),
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
			logp.Err("kubernetes: Watching API eror %v", err)
			time.Sleep(time.Second)
			continue
		}
		for {
			_, pod, err := watcher.Next()
			if err != nil {
				logp.Err("kubernetes: Watching API eror %v", err)
				time.Sleep(time.Second)
				continue
			}

			p.podQueue <- pod
		}
	}

}

func (p *PodWatcher) Run() bool {
	// Start pod processing worker:
	go p.worker()

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

func (p *PodWatcher) onPodAdd(pod *corev1.Pod) {
	metadata := p.indexers.GetMetadata(pod)
	p.annotationCache.Lock()
	defer p.annotationCache.Unlock()

	p.annotationCache.pods[pod.Metadata.GetUid()] = pod

	for _, m := range metadata {
		p.annotationCache.annotations[m.Index] = m.Data
	}
}

func (p *PodWatcher) onPodUpdate(pod *corev1.Pod) {
	oldPod := p.GetPod(pod.Metadata.GetUid())
	if oldPod.Metadata.GetResourceVersion() != pod.Metadata.GetResourceVersion() {
		//Process the new pod changes
		p.onPodDelete(oldPod)
		p.onPodAdd(pod)
	}
}

func (p *PodWatcher) onPodDelete(pod *corev1.Pod) {
	p.annotationCache.Lock()
	defer p.annotationCache.Unlock()

	delete(p.annotationCache.pods, pod.Metadata.GetUid())

	for _, index := range p.indexers.GetIndexes(pod) {
		delete(p.annotationCache.annotations, index)
	}
}

func (p *PodWatcher) worker() {
	for pod := range p.podQueue {
		if pod.Metadata.GetDeletionTimestamp() != nil {
			p.onPodDelete(pod)
		} else {
			existing := p.GetPod(pod.Metadata.GetUid())
			if existing != nil {
				p.onPodUpdate(pod)
			} else {
				p.onPodAdd(pod)
			}
		}
	}

}

func (p *PodWatcher) GetMetaData(arg string) common.MapStr {
	p.annotationCache.Lock()
	defer p.annotationCache.Unlock()
	if meta, ok := p.annotationCache.annotations[arg]; ok {
		return meta
	}
	return nil
}

func (p *PodWatcher) GetPod(uid string) *corev1.Pod {
	p.annotationCache.Lock()
	defer p.annotationCache.Unlock()
	return p.annotationCache.pods[uid]
}

func (p *PodWatcher) Stop() {
	p.stop()
	close(p.podQueue)
}
