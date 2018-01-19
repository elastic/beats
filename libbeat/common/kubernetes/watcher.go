package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/informers/internalinterfaces"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

func filterByNode(node string) internalinterfaces.TweakListOptionsFunc {
	return func(opts *metav1.ListOptions) {
		opts.FieldSelector = "spec.nodeName=" + node
	}
}

// Watcher reads Kubernetes events and keeps a list of known pods
type Watcher interface {
	// Start watching Kubernetes API for new containers
	Start() error

	// Stop watching Kubernetes API for new containers
	Stop()

	AddEventHandler(ResourceEventHandler)
}

type watcher struct {
	factory       informers.SharedInformerFactory
	informer      cache.SharedIndexInformer
	syncPeriod    time.Duration
	objToResource func(obj interface{}) Resource
	stop          chan struct{}
}

// NewWatcher initializes the watcher client to provide a local state of
// pod from the cluster (filtered to the given node)
func NewWatcher(clientset kubernetes.Interface, syncPeriod time.Duration, node, namespace string, r Resource) (Watcher, error) {
	var tf internalinterfaces.TweakListOptionsFunc
	if node != "" {
		tf = filterByNode(node)
	}
	f := informers.NewFilteredSharedInformerFactory(
		clientset,
		0,
		namespace,
		tf,
	)
	w := &watcher{
		factory:    f,
		syncPeriod: syncPeriod,
		stop:       make(chan struct{}),
	}
	switch r.(type) {
	case *Pod:
		w.informer = f.Core().V1().Pods().Informer()
		w.objToResource = func(obj interface{}) Resource {
			bytes, _ := json.Marshal(obj)
			r := &Pod{}
			json.Unmarshal(bytes, r)
			return r
		}
	case *Event:
		w.informer = f.Events().V1beta1().Events().Informer()
		w.objToResource = func(obj interface{}) Resource {
			bytes, _ := json.Marshal(obj)
			r := &Event{}
			json.Unmarshal(bytes, r)
			return r
		}
	default:
		return nil, fmt.Errorf("unsupported resource type for watching %T", r)
	}

	return w, nil
}

func (w *watcher) AddEventHandler(h ResourceEventHandler) {
	w.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			h.OnAdd(w.objToResource(obj))
		},
		UpdateFunc: func(old, new interface{}) {
			h.OnUpdate(w.objToResource(old), w.objToResource(new))
		},
		DeleteFunc: func(obj interface{}) {
			h.OnDelete(w.objToResource(obj))
		},
	})
}

// Start watching resources
func (w *watcher) Start() error {
	w.factory.Start(w.stop)
	// Make sure that events don't flow into the annotator before informer is fully set up
	// Sync initial state:
	ctx, cancl := context.WithTimeout(context.Background(), w.syncPeriod)
	defer cancl()
	for t, finished := range w.factory.WaitForCacheSync(ctx.Done()) {
		if !finished {
			return fmt.Errorf("kubernetes: Timeout while doing initial Kubernetes sync for %s", t)
		}
	}

	return nil
}

func (w *watcher) Stop() {
	close(w.stop)
}
