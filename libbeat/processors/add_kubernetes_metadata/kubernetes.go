package add_kubernetes_metadata

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors"

	"github.com/ericchiang/k8s"
	"github.com/ghodss/yaml"
)

const (
	timeout = time.Second * 5
)

var (
	fatalError = errors.New("Unable to create kubernetes processor")
)

type kubernetesAnnotator struct {
	podWatcher *PodWatcher
	matchers   *Matchers
}

func init() {
	processors.RegisterPlugin("add_kubernetes_metadata", newKubernetesAnnotator)

	// Register default indexers
	Indexing.AddIndexer(PodNameIndexerName, NewPodNameIndexer)
	Indexing.AddIndexer(ContainerIndexerName, NewContainerIndexer)
	Indexing.AddIndexer(IPPortIndexerName, NewIPPortIndexer)
	Indexing.AddMatcher(FieldMatcherName, NewFieldMatcher)
	Indexing.AddMatcher(FieldFormatMatcherName, NewFieldFormatMatcher)
}

func newKubernetesAnnotator(cfg *common.Config) (processors.Processor, error) {
	cfgwarn.Beta("The kubernetes processor is beta")

	config := defaultKubernetesAnnotatorConfig()

	err := cfg.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("fail to unpack the kubernetes configuration: %s", err)
	}

	err = validate(config)
	if err != nil {
		return nil, err
	}

	//Load default indexer configs
	if config.DefaultIndexers.Enabled == true {
		Indexing.RLock()
		for key, cfg := range Indexing.GetDefaultIndexerConfigs() {
			config.Indexers = append(config.Indexers, map[string]common.Config{key: cfg})
		}
		Indexing.RUnlock()
	}

	//Load default matcher configs
	if config.DefaultMatchers.Enabled == true {
		Indexing.RLock()
		for key, cfg := range Indexing.GetDefaultMatcherConfigs() {
			config.Matchers = append(config.Matchers, map[string]common.Config{key: cfg})
		}
		Indexing.RUnlock()
	}

	metaGen := NewGenDefaultMeta(config.IncludeAnnotations, config.IncludeLabels, config.ExcludeLabels)
	indexers := NewIndexers(config.Indexers, metaGen)

	matchers := NewMatchers(config.Matchers)

	if matchers.Empty() {
		return nil, fmt.Errorf("Can not initialize kubernetes plugin with zero matcher plugins")
	}

	var client *k8s.Client
	if config.InCluster == true {
		client, err = k8s.NewInClusterClient()
		if err != nil {
			return nil, fmt.Errorf("Unable to get in cluster configuration: %v", err)
		}
	} else {
		data, err := ioutil.ReadFile(config.KubeConfig)
		if err != nil {
			return nil, fmt.Errorf("read kubeconfig: %v", err)
		}

		// Unmarshal YAML into a Kubernetes config object.
		var config k8s.Config
		if err = yaml.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("unmarshal kubeconfig: %v", err)
		}
		client, err = k8s.NewClient(&config)
		if err != nil {
			return nil, err
		}
	}

	ctx := context.Background()
	if config.Host == "" {
		podName := os.Getenv("HOSTNAME")
		logp.Info("Using pod name %s and namespace %s", podName, client.Namespace)
		if podName == "localhost" {
			config.Host = "localhost"
		} else {
			pod, error := client.CoreV1().GetPod(ctx, podName, client.Namespace)
			if error != nil {
				logp.Err("Querying for pod failed with error: ", error.Error())
				logp.Info("Unable to find pod, setting host to localhost")
				config.Host = "localhost"
			} else {
				config.Host = pod.Spec.GetNodeName()
			}

		}
	}

	logp.Debug("kubernetes", "Using host ", config.Host)
	logp.Debug("kubernetes", "Initializing watcher")
	if client != nil {
		watcher := NewPodWatcher(client, indexers, config.SyncPeriod, config.CleanupTimeout, config.Host)

		if watcher.Run() {
			return &kubernetesAnnotator{podWatcher: watcher, matchers: matchers}, nil
		}

		return nil, fatalError
	}

	return nil, fatalError
}

func (k *kubernetesAnnotator) Run(event *beat.Event) (*beat.Event, error) {
	index := k.matchers.MetadataIndex(event.Fields)
	if index == "" {
		return event, nil
	}

	metadata := k.podWatcher.GetMetaData(index)
	if metadata == nil {
		return event, nil
	}

	meta := common.MapStr{}
	metaIface, ok := event.Fields["kubernetes"]
	if !ok {
		event.Fields["kubernetes"] = common.MapStr{}
	} else {
		meta = metaIface.(common.MapStr)
	}

	meta.Update(metadata)
	event.Fields["kubernetes"] = meta

	return event, nil
}

func (*kubernetesAnnotator) String() string { return "add_kubernetes_metadata" }

func validate(config kubeAnnotatorConfig) error {
	if !config.InCluster && config.KubeConfig == "" {
		return errors.New("`kube_config` path can't be empty when in_cluster is set to false")
	}
	return nil
}
