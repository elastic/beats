package beater

import (
	"fmt"
	"time"

	"github.com/elastic/beats/v7/libbeat/common/kubernetes"
	"github.com/elastic/beats/v7/libbeat/logp"
)

const ()

type KubeFetcher struct {
	watcher kubernetes.Watcher
}

func NewKubeFetcher(kubeconfig string, interval time.Duration) (Fetcher, error) {
	client, err := kubernetes.GetKubernetesClient(kubeconfig, kubernetes.KubeClientOptions{})
	if err != nil {
		return nil, fmt.Errorf("fail to get k8sclient client: %s", err.Error())
	}

	logp.Info("Client initiated.")

	watchOptions := kubernetes.WatchOptions{
		SyncTimeout: interval,
		Namespace:   "kube-system",
	}

	watcher, err := kubernetes.NewWatcher(client, &kubernetes.Pod{}, watchOptions, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating k8s client set: %v", err)
	}

	logp.Info("Watcher initiated.")

	return &KubeFetcher{
		watcher: watcher,
	}, nil
}

func (f *KubeFetcher) Fetch() (interface{}, error) {
	pods := f.watcher.Store().List()

	for _, p := range pods {
		pod, ok := p.(*kubernetes.Pod)
		if !ok {
			logp.Info("could not convert to pod")
			continue
		}
		pod.SetManagedFields(nil)
		pod.Status.Reset()
		pod.Kind = "Pod" // see https://github.com/kubernetes/kubernetes/issues/3030
	}

	return pods, nil
}

func (f *KubeFetcher) Stop() {
}
