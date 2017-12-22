package add_kubernetes_metadata

import (
	"context"
	"sync"
	"time"

	"github.com/ericchiang/k8s"

	"github.com/elastic/beats/libbeat/common"
)

type watcher interface {
	sync() ([]Resource, error)
	watch() (ch <-chan Resource)
}

type watcherConstructor func(ctx context.Context, client *k8s.Client, host string) watcher

// ResourceWatcher is a controller that synchronizes Resources.
type ResourceWatcher struct {
	watchers        []watcher
	cleanupTimeout  time.Duration
	resourceQueue   chan Resource
	ctx             context.Context
	stop            context.CancelFunc
	annotationCache annotationCache
	indexers        *Indexers
}

type annotationCache struct {
	sync.RWMutex
	annotations map[string]common.MapStr
	resources   map[string]Resource  // resource uid -> Resource
	deleted     map[string]time.Time // deleted annotations key -> last access time
}

// newresourceWatcher initializes the watcher client to provide a local state of
// resources from the cluster (filtered to the given host)
func newResourceWatcher(kubeClient *k8s.Client, indexers *Indexers, cleanupTimeout time.Duration, host string, wcs ...watcherConstructor) *ResourceWatcher {
	ctx, cancel := context.WithCancel(context.Background())
	rq := make(chan Resource, 10)
	rw := &ResourceWatcher{
		indexers:       indexers,
		cleanupTimeout: cleanupTimeout,
		resourceQueue:  rq,
		ctx:            ctx,
		stop:           cancel,
		annotationCache: annotationCache{
			annotations: make(map[string]common.MapStr),
			resources:   make(map[string]Resource),
			deleted:     make(map[string]time.Time),
		},
	}
	for _, wc := range wcs {
		rw.watchers = append(rw.watchers, wc(ctx, kubeClient, host))
	}
	return rw
}

func (p *ResourceWatcher) Run() bool {
	// Start processing & annotations cleanup workers
	go p.worker()
	go p.cleanupWorker()

	b := make(chan bool)

	for _, watcher := range p.watchers {
		watcher := watcher
		go func() {
			// Make sure that events don't flow into the annotator before informer is fully set up
			// Sync initial state:
			var list []Resource
			var err error
			for i := 0; i < 3; i++ {
				list, err = watcher.sync()
				if err == nil {
					break
				}
				time.Sleep(time.Second)
			}
			if err != nil {
				b <- false
				return
			}
			for _, r := range list {
				if p.ctx.Err() != nil {
					b <- false
					return
				}
				p.resourceQueue <- r
			}

			// Watch for new changes
			b <- true
			rq := watcher.watch()
			for r := range rq {
				if p.ctx.Err() != nil {
					return
				}
				p.resourceQueue <- r
			}
		}()
	}

	for range p.watchers {
		if !<-b {
			p.Stop()
			return false
		}
	}
	return true
}

func (p *ResourceWatcher) onAdd(r Resource) {
	metadata := p.indexers.GetMetadata(r)
	p.annotationCache.Lock()
	defer p.annotationCache.Unlock()

	p.annotationCache.resources[r.GetMetadata().UID] = r

	for _, m := range metadata {
		p.annotationCache.annotations[m.Index] = m.Data

		// un-delete if it's flagged (in case of update or recreation)
		delete(p.annotationCache.deleted, m.Index)
	}
}

func (p *ResourceWatcher) onUpdate(r Resource) {
	oldR := p.GetResource(r.GetMetadata().UID)
	if oldR.GetMetadata().ResourceVersion != r.GetMetadata().ResourceVersion {
		//Process the new changes
		p.onDelete(oldR)
		p.onAdd(r)
	}
}

func (p *ResourceWatcher) onDelete(r Resource) {
	p.annotationCache.Lock()
	defer p.annotationCache.Unlock()

	delete(p.annotationCache.resources, r.GetMetadata().UID)

	// Flag all annotations as deleted (they will be still available for a while)
	now := time.Now()
	for _, index := range p.indexers.GetIndexes(r) {
		p.annotationCache.deleted[index] = now
	}
}

func (p *ResourceWatcher) worker() {
	for res := range p.resourceQueue {
		if res.GetMetadata().DeletionTimestamp != "" {
			p.onDelete(res)
		} else {
			existing := p.GetResource(res.GetMetadata().UID)
			if existing != nil {
				p.onUpdate(res)
			} else {
				p.onAdd(res)
			}
		}
	}
}

// Check annotations flagged as deleted for their last access time, fully delete
// the ones older than p.cleanupTimeout
func (p *ResourceWatcher) cleanupWorker() {
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

func (p *ResourceWatcher) GetMetaData(arg string) common.MapStr {
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

func (p *ResourceWatcher) GetResource(uid string) Resource {
	p.annotationCache.RLock()
	defer p.annotationCache.RUnlock()
	return p.annotationCache.resources[uid]
}

func (p *ResourceWatcher) Stop() {
	p.stop()
	close(p.resourceQueue)
}
