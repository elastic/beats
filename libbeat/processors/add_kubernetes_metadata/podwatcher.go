package add_kubernetes_metadata

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ericchiang/k8s"
	corev1 "github.com/ericchiang/k8s/api/v1"

	"github.com/elastic/beats/libbeat/logp"
)

type podWatcher struct {
	ctx                 context.Context
	kubeClient          *k8s.Client
	host                string
	lastResourceVersion string
}

func newPodWatcher(ctx context.Context, client *k8s.Client, host string) watcher {
	return &podWatcher{
		ctx:        ctx,
		kubeClient: client,
		host:       host,
	}
}

func (pw *podWatcher) buildOpts() []k8s.Option {
	opts := []k8s.Option{k8s.ResourceVersion(pw.lastResourceVersion)}
	if pw.host != "" {
		opts = append(opts, k8s.QueryParam("fieldSelector", "spec.nodeName="+pw.host))
	}
	return opts
}

func (pw *podWatcher) sync() ([]Resource, error) {
	logp.Info("kubernetes: %s", "Performing a pod sync")
	pods, err := pw.kubeClient.CoreV1().ListPods(
		pw.ctx,
		k8s.AllNamespaces,
		pw.buildOpts()...,
	)
	if err != nil {
		logp.Err("kubernetes: List API error %v", err)
		return nil, err
	}
	// Store last version
	pw.lastResourceVersion = pods.Metadata.GetResourceVersion()
	rs := make([]Resource, 0, len(pods.Items))
	for _, pod := range pods.Items {
		rs = append(rs, pw.convert(pod))
	}
	logp.Info("kubernetes: %s", "Pod sync done")
	return rs, nil
}

func (pw *podWatcher) convert(p interface{}) Resource {
	pod := p.(*corev1.Pod)
	bytes, err := json.Marshal(pod)
	if err != nil {
		logp.Warn("Unable to marshal %v", pod.String())
		return nil
	}

	po := &Pod{}
	err = json.Unmarshal(bytes, po)
	if err != nil {
		logp.Warn("Unable to marshal %v", pod.String())
		return nil
	}

	return po
}

func (pw *podWatcher) watch() <-chan Resource {
	ch := make(chan Resource, 10)
	go func() {
		for pw.ctx.Err() == nil {
			logp.Info("kubernetes: %s", "Watching API for pod events")
			watcher, err := pw.kubeClient.CoreV1().WatchPods(pw.ctx, "", pw.buildOpts()...)
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
				select {
				case <-pw.ctx.Done():
					watcher.Close()
					return
				case ch <- pw.convert(pod):
				}
			}
		}
	}()
	return ch
}
