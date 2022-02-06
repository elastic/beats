package fetchers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/elastic/beats/v7/kubebeat/resources"
	"github.com/elastic/beats/v7/libbeat/common/kubernetes"
	"github.com/elastic/beats/v7/libbeat/logp"

	"k8s.io/apimachinery/pkg/runtime"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
)

const (
	KubeAPIType   = "kube-api"
	allNamespaces = "" // The Kube API treats this as "all namespaces"
)

var (
	// vanillaClusterResources represents those resources that are required for a vanilla
	// Kubernetes cluster.
	vanillaClusterResources = []requiredResource{
		{
			&kubernetes.Pod{},
			allNamespaces,
		},
		{
			&kubernetes.Secret{},
			allNamespaces,
		},
		{
			&kubernetes.Role{},
			allNamespaces,
		},
		{
			&kubernetes.RoleBinding{},
			allNamespaces,
		},
		{
			&kubernetes.ClusterRole{},
			allNamespaces,
		},
		{
			&kubernetes.ClusterRoleBinding{},
			allNamespaces,
		},
		{
			&kubernetes.PodSecurityPolicy{},
			allNamespaces,
		},
		// TODO(yashtewari): Problem: github.com/elastic/beats/vendor/k8s.io/apimachinery/pkg/api/errors/errors.go#401
		// > "the server could not find the requested resource"
		// {
		// 	&kubernetes.NetworkPolicy{},
		// 	allNamespaces,
		// },
	}

	watcherlock sync.Once
)

type requiredResource struct {
	resource  kubernetes.Resource
	namespace string
}

type KubeFetcher struct {
	kubeconfig string
	interval   time.Duration
	watchers   []kubernetes.Watcher
}

func NewKubeFetcher(kubeconfig string, interval time.Duration) (resources.Fetcher, error) {
	f := &KubeFetcher{
		kubeconfig: kubeconfig,
		interval:   interval,
		watchers:   make([]kubernetes.Watcher, 0),
	}

	return f, nil
}

func (f *KubeFetcher) initWatcher(client k8s.Interface, r requiredResource) error {
	watcher, err := kubernetes.NewWatcher(client, r.resource, kubernetes.WatchOptions{
		SyncTimeout: f.interval,
		Namespace:   r.namespace,
	}, nil)
	if err != nil {
		return fmt.Errorf("could not create watcher: %w", err)
	}

	// TODO(yashtewari): it appears that Start never returns in case of certain failures, for example
	// if the configured Client's Role does not have the necessary permissions to list the Resource
	// being watched. This needs to be handled.
	//
	// When such a failure happens, kubebeat won't shut down gracefuly, i.e. Stop will not work. This
	// happens due to a context.TODO present in the libbeat dependency. It needs to accept context
	// from the caller instead.
	if err := watcher.Start(); err != nil {
		return fmt.Errorf("could not start watcher: %w", err)
	}

	f.watchers = append(f.watchers, watcher)

	return nil
}

func (f *KubeFetcher) initWatchers() error {
	client, err := kubernetes.GetKubernetesClient(f.kubeconfig, kubernetes.KubeClientOptions{})
	if err != nil {
		return fmt.Errorf("could not get k8s client: %w", err)
	}

	logp.Info("Kubernetes client initiated.")

	f.watchers = make([]kubernetes.Watcher, 0)

	for _, r := range vanillaClusterResources {
		err := f.initWatcher(client, r)
		if err != nil {
			return err
		}
	}

	logp.Info("Kubernetes Watchers initiated.")

	return nil
}

func (f *KubeFetcher) Fetch(ctx context.Context) ([]resources.FetcherResult, error) {
	var err error
	watcherlock.Do(func() {
		err = f.initWatchers()
	})
	if err != nil {
		// Reset watcherlock if the watchers could not be initiated.
		watcherlock = sync.Once{}
		return nil, fmt.Errorf("could not initate Kubernetes watchers: %w", err)
	}

	return GetKubeData(f.watchers), nil
}

func (f *KubeFetcher) Stop() {
	for _, watcher := range f.watchers {
		watcher.Stop()
	}
}

// addTypeInformationToKubeResource adds TypeMeta information to a kubernetes.Resource based upon the loaded scheme.Scheme
// inspired by: https://github.com/kubernetes/cli-runtime/blob/v0.19.2/pkg/printers/typesetter.go#L41
func addTypeInformationToKubeResource(resource kubernetes.Resource) error {
	groupVersionKinds, _, err := scheme.Scheme.ObjectKinds(resource)
	if err != nil {
		return fmt.Errorf("missing apiVersion or kind and cannot assign it; %w", err)
	}

	for _, groupVersionKind := range groupVersionKinds {
		if len(groupVersionKind.Kind) == 0 {
			continue
		}
		if len(groupVersionKind.Version) == 0 || groupVersionKind.Version == runtime.APIVersionInternal {
			continue
		}
		resource.GetObjectKind().SetGroupVersionKind(groupVersionKind)
		break
	}

	return nil
}
