// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package util

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/elastic/beats/v7/metricbeat/mb"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"

	"github.com/stretchr/testify/assert"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	clientfeatures "k8s.io/client-go/features"
	clientfeaturestesting "k8s.io/client-go/features/testing"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	kubernetes2 "github.com/elastic/beats/v7/libbeat/autodiscover/providers/kubernetes"
	"github.com/elastic/beats/v7/pkg/autodiscover/kubernetes/metadata"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/stretchr/testify/require"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8smetafake "k8s.io/client-go/metadata/fake"

	"github.com/elastic/beats/v7/pkg/autodiscover/kubernetes"
)

const (
	podBName                  = "pod-b"
	informerTestContainerName = "container"
)

type constructorRollbackMetricSet struct {
	mb.BaseMetricSet
}

func (*constructorRollbackMetricSet) Fetch(mb.ReporterV2) {}

func TestWatchOptions(t *testing.T) {
	log := logptest.NewTestingLogger(t, "test")

	client := k8sfake.NewSimpleClientset()
	config := &kubernetesConfig{
		Namespace:  "test-ns",
		SyncPeriod: time.Minute,
		Node:       "test-node",
	}

	options, err := getWatchOptions(config, false, client, log)
	require.NoError(t, err)
	require.Equal(t, options.SyncTimeout, config.SyncPeriod)
	require.NotEqual(t, options.Node, config.Node)

	options, err = getWatchOptions(config, true, client, log)
	require.NoError(t, err)
	require.Equal(t, options.SyncTimeout, config.SyncPeriod)
	require.Equal(t, options.Node, config.Node)
}

func TestCreateWatcher(t *testing.T) {
	resourceWatchers := NewWatchers()
	metricsRepo := NewMetricsRepo()

	client := k8sfake.NewSimpleClientset()
	metadataClient := k8smetafake.NewSimpleMetadataClient(k8smetafake.NewTestScheme())
	config := &kubernetesConfig{
		Namespace:  "test-ns",
		SyncPeriod: time.Minute,
		Node:       "test-node",
	}
	log := logptest.NewTestingLogger(t, "test")

	options, err := getWatchOptions(config, false, client, log)
	require.NoError(t, err)
	namespaceEnricher := newMetadataEnricher("state_namespace", NamespaceResource, config, log)

	created, err := createWatcher(
		NamespaceResource,
		&kubernetes.Node{},
		*options,
		client,
		metadataClient,
		resourceWatchers,
		metricsRepo,
		config.Namespace,
		false,
		namespaceEnricher,
		logptest.NewTestingLogger(t, ""),
	)
	require.True(t, created)
	require.NoError(t, err)

	resourceWatchers.lock.Lock()
	require.Len(t, resourceWatchers.metaWatchersMap, 1)
	require.NotNil(t, resourceWatchers.metaWatchersMap[NamespaceResource])
	require.NotNil(t, resourceWatchers.metaWatchersMap[NamespaceResource].watcher)
	resourceWatchers.lock.Unlock()

	created, err = createWatcher(
		NamespaceResource,
		&kubernetes.Namespace{},
		*options, client,
		metadataClient,
		resourceWatchers,
		metricsRepo,
		config.Namespace,
		true,
		newMetadataEnricher("state_deployment", DeploymentResource, config, log),
		logptest.NewTestingLogger(t, ""),
	)
	require.False(t, created)
	require.NoError(t, err)

	resourceWatchers.lock.Lock()
	require.Len(t, resourceWatchers.metaWatchersMap, 1)
	require.NotNil(t, resourceWatchers.metaWatchersMap[NamespaceResource])
	require.NotNil(t, resourceWatchers.metaWatchersMap[NamespaceResource].watcher)
	resourceWatchers.lock.Unlock()

	created, err = createWatcher(
		DeploymentResource,
		&kubernetes.Deployment{},
		*options, client,
		metadataClient,
		resourceWatchers,
		metricsRepo,
		config.Namespace,
		false,
		newMetadataEnricher("state_deployment", DeploymentResource, config, log),
		logptest.NewTestingLogger(t, ""))
	require.True(t, created)
	require.NoError(t, err)

	resourceWatchers.lock.Lock()
	require.Len(t, resourceWatchers.metaWatchersMap, 2)
	require.NotNil(t, resourceWatchers.metaWatchersMap[DeploymentResource])
	require.NotNil(t, resourceWatchers.metaWatchersMap[NamespaceResource])
	resourceWatchers.lock.Unlock()
}

func TestWatcherUserPointerIdentity(t *testing.T) {
	metaWatcher := &metaWatcher{users: make(map[*enricher]watcherRegistration)}
	first := &enricher{metricsetName: "pod"}
	second := &enricher{metricsetName: "pod"}

	require.True(t, addWatcherUser(metaWatcher, first, true), "first pointer must acquire ownership")
	require.True(t, addWatcherUser(metaWatcher, second, false), "second pointer with the same name must acquire ownership")
	require.False(t, addWatcherUser(metaWatcher, first, false), "the same pointer must not acquire ownership twice")
	require.Len(t, metaWatcher.users, 2, "ownership must be keyed by pointer identity")
	require.True(t, metaWatcher.users[first].nodeScope, "first pointer's scope must be preserved")
	require.False(t, metaWatcher.users[second].nodeScope, "second pointer's scope must be preserved")

	require.False(t, removeWatcherUser(metaWatcher, first), "one pointer remains")
	require.True(t, removeWatcherUser(metaWatcher, second), "the final pointer was removed")
}

func TestWatcherContainerMetrics(t *testing.T) {
	resourceWatchers := NewWatchers()
	metricsRepo := NewMetricsRepo()

	containerName := "test"
	cpuLimit := resource.MustParse("100m")
	memoryLimit := resource.MustParse("100Mi")
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			UID:  types.UID("mockuid"),
			Name: "enrich",
			Labels: map[string]string{
				"label": "value",
			},
			Namespace: "default",
		},
		Spec: v1.PodSpec{
			NodeName: "test-node",
			Containers: []v1.Container{
				{
					Name: containerName,
					Resources: v1.ResourceRequirements{
						Limits: v1.ResourceList{
							v1.ResourceCPU:    cpuLimit,
							v1.ResourceMemory: memoryLimit,
						},
					},
				},
			},
		},
	}
	podId := NewPodId(pod.Namespace, pod.Name)
	resourceWatchers.lock.Lock()

	watcher := newMockWatcher()
	metaWatcher := &metaWatcher{
		watcher:     watcher,
		started:     false,
		users:       make(map[*enricher]watcherRegistration),
		enrichers:   make(map[*enricher]struct{}),
		metricsRepo: metricsRepo,
	}
	resourceWatchers.metaWatchersMap[PodResource] = metaWatcher
	addEventHandlersToWatcher(metaWatcher, resourceWatchers)
	resourceWatchers.lock.Unlock()

	// add Pod and verify container metrics are present and valid
	watcher.handler.OnAdd(pod)

	containerStore := metricsRepo.GetNodeStore(pod.Spec.NodeName).GetPodStore(podId).GetContainerStore(containerName)
	metrics := containerStore.GetContainerMetrics()
	require.NotNil(t, metrics)
	assert.Equal(t, 0.1, metrics.CoresLimit.Value)              //nolint:testifylint // exact float comparison
	assert.Equal(t, 100*1024*1024.0, metrics.MemoryLimit.Value) //nolint:testifylint // exact float comparison

	// modify the limit and verify the new value is present
	pod.Spec.Containers[0].Resources.Limits[v1.ResourceCPU] = resource.MustParse("200m")
	watcher.handler.OnUpdate(pod)
	metrics = containerStore.GetContainerMetrics()
	require.NotNil(t, metrics)
	assert.Equal(t, 0.2, metrics.CoresLimit.Value) //nolint:testifylint // exact float comparison

	// delete the pod and verify no metrics are present
	watcher.handler.OnDelete(pod)
	containerStore = metricsRepo.GetNodeStore(pod.Spec.NodeName).GetPodStore(podId).GetContainerStore(containerName)
	metrics = containerStore.GetContainerMetrics()
	require.NotNil(t, metrics)
	assert.Nil(t, metrics.CoresLimit)
	assert.Nil(t, metrics.MemoryLimit)
}

func TestWatcherNodeMetrics(t *testing.T) {
	resourceWatchers := NewWatchers()
	metricsRepo := NewMetricsRepo()

	cpuLimit := resource.MustParse("100m")
	memoryLimit := resource.MustParse("100Mi")
	node := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			UID:  types.UID("mockuid"),
			Name: "enrich",
			Labels: map[string]string{
				"label": "value",
			},
			Namespace: "default",
		},
		Status: v1.NodeStatus{
			Allocatable: v1.ResourceList{
				v1.ResourceCPU:    cpuLimit,
				v1.ResourceMemory: memoryLimit,
			},
		},
	}
	resourceWatchers.lock.Lock()

	watcher := newMockWatcher()
	metaWatcher := &metaWatcher{
		watcher:     watcher,
		started:     false,
		users:       make(map[*enricher]watcherRegistration),
		enrichers:   make(map[*enricher]struct{}),
		metricsRepo: metricsRepo,
	}
	resourceWatchers.metaWatchersMap[NodeResource] = metaWatcher
	addEventHandlersToWatcher(metaWatcher, resourceWatchers)
	resourceWatchers.lock.Unlock()

	// add node and verify container metrics are present and valid
	watcher.handler.OnAdd(node)

	nodeStore := metricsRepo.GetNodeStore(node.Name)
	metrics := nodeStore.GetNodeMetrics()
	require.NotNil(t, metrics)
	assert.Equal(t, 0.1, metrics.CoresAllocatable.Value)              //nolint:testifylint // exact float comparison
	assert.Equal(t, 100*1024*1024.0, metrics.MemoryAllocatable.Value) //nolint:testifylint // exact float comparison

	// modify the limit and verify the new value is present
	node.Status.Allocatable[v1.ResourceCPU] = resource.MustParse("200m")
	watcher.handler.OnUpdate(node)
	metrics = nodeStore.GetNodeMetrics()
	require.NotNil(t, metrics)
	assert.Equal(t, 0.2, metrics.CoresAllocatable.Value) //nolint:testifylint // exact float comparison

	// delete the node and verify no metrics are present
	watcher.handler.OnDelete(node)
	nodeStore = metricsRepo.GetNodeStore(node.Name)
	metrics = nodeStore.GetNodeMetrics()
	require.NotNil(t, metrics)
	assert.Nil(t, metrics.CoresAllocatable)
	assert.Nil(t, metrics.MemoryAllocatable)
}

func TestCreateAllWatchers(t *testing.T) {
	resourceWatchers := NewWatchers()
	metricsRepo := NewMetricsRepo()

	client := k8sfake.NewSimpleClientset()
	metadataClient := k8smetafake.NewSimpleMetadataClient(k8smetafake.NewTestScheme())
	config := &kubernetesConfig{
		Namespace:  "test-ns",
		SyncPeriod: time.Minute,
		Node:       "test-node",
		AddResourceMetadata: &metadata.AddResourceMetadataConfig{
			CronJob:    false,
			Deployment: true,
		},
	}
	log := logptest.NewTestingLogger(t, "test")

	// Start watchers based on a resource that does not exist should cause an error
	err := createAllWatchers(
		client,
		metadataClient,
		newMetadataEnricher("does-not-exist", "does-not-exist", config, log),
		false,
		config,
		log,
		resourceWatchers,
		metricsRepo)
	require.Error(t, err)
	resourceWatchers.lock.Lock()
	require.Empty(t, resourceWatchers.metaWatchersMap)
	resourceWatchers.lock.Unlock()

	// Start watcher for a resource that requires other resources, should start all the watchers
	metricsetPod := "pod"
	extras := getExtraWatchers(PodResource, config.AddResourceMetadata)
	err = createAllWatchers(
		client,
		metadataClient,
		newMetadataEnricher(metricsetPod, PodResource, config, log),
		false,
		config,
		log,
		resourceWatchers,
		metricsRepo)
	require.NoError(t, err)

	// Check that all the required watchers are in the map
	resourceWatchers.lock.Lock()
	// we add 1 to the expected result to represent the resource itself
	require.Len(t, resourceWatchers.metaWatchersMap, len(extras)+1)
	for _, extra := range extras {
		require.NotNil(t, resourceWatchers.metaWatchersMap[extra])
	}
	resourceWatchers.lock.Unlock()
}

func TestCreateMetaGen(t *testing.T) {
	resourceWatchers := NewWatchers()
	metricsRepo := NewMetricsRepo()

	commonMetaConfig := metadata.Config{}
	commonConfig, err := conf.NewConfigFrom(&commonMetaConfig)
	require.NoError(t, err)

	log := logptest.NewTestingLogger(t, "test")

	config := &kubernetesConfig{
		Namespace:  "test-ns",
		SyncPeriod: time.Minute,
		Node:       "test-node",
		AddResourceMetadata: &metadata.AddResourceMetadataConfig{
			CronJob:    false,
			Deployment: true,
		},
	}
	client := k8sfake.NewSimpleClientset()
	metadataClient := k8smetafake.NewSimpleMetadataClient(k8smetafake.NewTestScheme())

	_, err = createMetadataGen(client, commonConfig, config.AddResourceMetadata, DeploymentResource, resourceWatchers)
	// At this point, no watchers were created
	require.Error(t, err)

	// Create the watchers necessary for the metadata generator
	metricsetDeployment := "state_deployment"
	err = createAllWatchers(
		client,
		metadataClient,
		newMetadataEnricher(metricsetDeployment, DeploymentResource, config, log),
		false,
		config,
		log,
		resourceWatchers,
		metricsRepo)
	require.NoError(t, err)

	// Create the generators, this time without error
	_, err = createMetadataGen(client, commonConfig, config.AddResourceMetadata, DeploymentResource, resourceWatchers)
	require.NoError(t, err)
}

func TestCreateMetaGenSpecific(t *testing.T) {
	resourceWatchers := NewWatchers()
	metricsRepo := NewMetricsRepo()

	commonMetaConfig := metadata.Config{}
	commonConfig, err := conf.NewConfigFrom(&commonMetaConfig)
	require.NoError(t, err)

	log := logptest.NewTestingLogger(t, "test")

	namespaceConfig, err := conf.NewConfigFrom(map[string]any{
		"enabled": true,
	})
	require.NoError(t, err)

	config := &kubernetesConfig{
		Namespace:  "test-ns",
		SyncPeriod: time.Minute,
		Node:       "test-node",
		AddResourceMetadata: &metadata.AddResourceMetadataConfig{
			CronJob:    false,
			Deployment: true,
			Namespace:  namespaceConfig,
		},
	}
	client := k8sfake.NewSimpleClientset()
	metadataClient := k8smetafake.NewSimpleMetadataClient(k8smetafake.NewTestScheme())

	// For pod:
	metricsetPod := "pod"

	_, err = createMetadataGenSpecific(client, commonConfig, config.AddResourceMetadata, PodResource, resourceWatchers, nil)
	// At this point, no watchers were created
	require.Error(t, err)

	// Create the pod resource + the extras
	err = createAllWatchers(
		client,
		metadataClient,
		newMetadataEnricher(metricsetPod, PodResource, config, log),
		false,
		config,
		log,
		resourceWatchers,
		metricsRepo)
	require.NoError(t, err)

	_, err = createMetadataGenSpecific(client, commonConfig, config.AddResourceMetadata, PodResource, resourceWatchers, nil)
	require.NoError(t, err)

	// For service:
	_, err = createMetadataGenSpecific(client, commonConfig, config.AddResourceMetadata, ServiceResource, resourceWatchers, nil)
	// At this point, no watchers were created
	require.Error(t, err)

	// Create the service resource + the extras
	metricsetService := "state_service"
	err = createAllWatchers(
		client,
		metadataClient,
		newMetadataEnricher(metricsetService, ServiceResource, config, log),
		false,
		config,
		log,
		resourceWatchers,
		metricsRepo)
	require.NoError(t, err)

	_, err = createMetadataGenSpecific(client, commonConfig, config.AddResourceMetadata, ServiceResource, resourceWatchers, nil)
	require.NoError(t, err)
}

func TestEnricherStopUsesPointerOwnershipAndEvictsFinalWatcher(t *testing.T) {
	resourceWatchers := NewWatchers()
	watcher := newMockWatcher()
	resourceWatchers.lock.Lock()
	resourceWatchers.metaWatchersMap[PodResource] = &metaWatcher{
		watcher:   watcher,
		users:     make(map[*enricher]watcherRegistration),
		enrichers: make(map[*enricher]struct{}),
	}
	resourceWatchers.lock.Unlock()

	funcs := mockFuncs{}
	config := &kubernetesConfig{
		AddResourceMetadata: &metadata.AddResourceMetadataConfig{},
	}
	log := logptest.NewTestingLogger(t, selector)
	first := buildTestMetadataEnricher("pod", PodResource, resourceWatchers, config, &funcs, log)
	second := buildTestMetadataEnricher("pod", PodResource, resourceWatchers, config, &funcs, log)

	resourceWatchers.lock.Lock()
	require.Len(t, resourceWatchers.metaWatchersMap[PodResource].users, 2, "same-name enrichers must both own the watcher")
	require.Len(t, resourceWatchers.metaWatchersMap[PodResource].enrichers, 2, "same-name enrichers must both receive invalidation")
	resourceWatchers.lock.Unlock()

	assert.True(t, first.Start(resourceWatchers), "Start() must return true")
	resourceWatchers.lock.Lock()
	require.True(t, resourceWatchers.metaWatchersMap[PodResource].started, "watcher must start")
	resourceWatchers.lock.Unlock()

	// Call stop twice to assert its idempotency.
	first.Stop(resourceWatchers)
	first.Stop(resourceWatchers)
	resourceWatchers.lock.Lock()
	require.Contains(t, resourceWatchers.metaWatchersMap, PodResource, "remaining pointer must retain the watcher")
	require.Len(t, resourceWatchers.metaWatchersMap[PodResource].users, 1, "only the second pointer remains")
	resourceWatchers.lock.Unlock()
	require.Equal(t, 0, watcher.stopCalls, "idempotent non-final stop must not stop the shared watcher")

	second.Stop(resourceWatchers)
	resourceWatchers.lock.Lock()
	require.NotContains(t, resourceWatchers.metaWatchersMap, PodResource, "final owner must evict the watcher")
	resourceWatchers.lock.Unlock()
	require.Equal(t, 1, watcher.stopCalls, "final owner must stop the watcher exactly once")
}

func TestPodAndContainerEnrichersShareWatcherByPointer(t *testing.T) {
	resourceWatchers := NewWatchers()
	watcher := newMockWatcher()
	resourceWatchers.metaWatchersMap[PodResource] = &metaWatcher{
		watcher:   watcher,
		users:     make(map[*enricher]watcherRegistration),
		enrichers: make(map[*enricher]struct{}),
	}

	config := &kubernetesConfig{AddResourceMetadata: &metadata.AddResourceMetadataConfig{}}
	log := logptest.NewTestingLogger(t, selector)
	funcs := mockFuncs{}
	// Both metricsets enrich kubelet events from Pod API objects, so their
	// distinct enrichers intentionally share the same PodResource watcher.
	pod := buildTestMetadataEnricher("pod", PodResource, resourceWatchers, config, &funcs, log)
	container := buildTestMetadataEnricher("container", PodResource, resourceWatchers, config, &funcs, log)

	// Starting either owner starts their shared watcher; using pod here is
	// arbitrary. Stopping container then simulates one metricset shutting down
	// while pod still needs the watcher, so the watcher must remain running.
	assert.True(t, pod.Start(resourceWatchers), "Start() must return true")
	container.Stop(resourceWatchers)
	resourceWatchers.lock.RLock()
	require.Contains(t, resourceWatchers.metaWatchersMap, PodResource, "pod pointer must retain the shared watcher")
	require.Len(t, resourceWatchers.metaWatchersMap[PodResource].users, 1, "only pod ownership remains")
	resourceWatchers.lock.RUnlock()
	require.Equal(t, 0, watcher.stopCalls, "container release must not stop pod's watcher")

	// Pod is now the final owner, so releasing it must stop the watcher.
	pod.Stop(resourceWatchers)
	require.Equal(t, 1, watcher.stopCalls, "final pod release must stop the watcher")
}

func TestEnricherTracksAndReleasesExactExtraWatchers(t *testing.T) {
	resourceWatchers := NewWatchers()
	metricsRepo := NewMetricsRepo()
	client := k8sfake.NewSimpleClientset()
	metadataClient := k8smetafake.NewSimpleMetadataClient(k8smetafake.NewTestScheme())
	log := logptest.NewTestingLogger(t, selector)

	firstConfig := &kubernetesConfig{
		Node:                "test-node",
		SyncPeriod:          time.Second,
		AddResourceMetadata: resourceMetadataConfig(t, true, false, true, false),
	}
	first := newMetadataEnricher("pod", PodResource, firstConfig, log)
	require.NoError(
		t,
		createAllWatchers(client, metadataClient, first, true, firstConfig, log, resourceWatchers, metricsRepo),
		"first enricher watcher registration must succeed",
	)
	commitWatcherOwnership(first, resourceWatchers)

	secondConfig := &kubernetesConfig{
		SyncPeriod:          time.Second,
		AddResourceMetadata: resourceMetadataConfig(t, false, true, false, false),
	}
	second := newMetadataEnricher("pod", PodResource, secondConfig, log)
	require.NoError(
		t,
		createAllWatchers(client, metadataClient, second, false, secondConfig, log, resourceWatchers, metricsRepo),
		"second enricher watcher registration must succeed",
	)
	commitWatcherOwnership(second, resourceWatchers)

	require.ElementsMatch(
		t,
		[]string{PodResource, NodeResource, ReplicaSetResource},
		first.watchedResources,
		"first enricher must record only its successful watcher dependencies",
	)
	require.ElementsMatch(
		t,
		[]string{PodResource, NamespaceResource},
		second.watchedResources,
		"second enricher must record only its successful watcher dependencies",
	)

	resourceWatchers.lock.RLock()
	require.Len(t, resourceWatchers.metaWatchersMap[PodResource].users, 2, "both pointers own the shared pod watcher")
	require.Len(t, resourceWatchers.metaWatchersMap[PodResource].enrichers, 2, "pod events invalidate both pod metadata caches")
	require.True(t, resourceWatchers.metaWatchersMap[PodResource].users[first].nodeScope, "primary watcher must retain the enricher's node scope")
	require.False(t, resourceWatchers.metaWatchersMap[NodeResource].users[first].nodeScope, "extra node watcher must be cluster scoped")
	require.False(t, resourceWatchers.metaWatchersMap[ReplicaSetResource].users[first].nodeScope, "extra ReplicaSet watcher must be cluster scoped")
	require.Empty(t, resourceWatchers.metaWatchersMap[NodeResource].enrichers, "node dependency must not invalidate pod caches")
	require.Empty(t, resourceWatchers.metaWatchersMap[NamespaceResource].enrichers, "namespace dependency must not invalidate pod caches")
	require.Empty(t, resourceWatchers.metaWatchersMap[ReplicaSetResource].enrichers, "ReplicaSet dependency must not invalidate pod caches")
	resourceWatchers.lock.RUnlock()

	first.Stop(resourceWatchers)
	resourceWatchers.lock.RLock()
	require.Contains(t, resourceWatchers.metaWatchersMap, PodResource, "second enricher still owns the pod watcher")
	require.Contains(t, resourceWatchers.metaWatchersMap, NamespaceResource, "second enricher still owns its namespace watcher")
	require.NotContains(t, resourceWatchers.metaWatchersMap, NodeResource, "first enricher's node watcher must be evicted")
	require.NotContains(t, resourceWatchers.metaWatchersMap, ReplicaSetResource, "first enricher's ReplicaSet watcher must be evicted")
	resourceWatchers.lock.RUnlock()

	second.Stop(resourceWatchers)
	resourceWatchers.lock.RLock()
	require.Empty(t, resourceWatchers.metaWatchersMap, "all exact dependencies must be evicted after their final owners exit")
	resourceWatchers.lock.RUnlock()
}

func TestNewResourceMetadataEnricherRollsBackRegisteredWatchers(t *testing.T) {
	resourceWatchers := NewWatchers()
	metricsRepo := NewMetricsRepo()

	// Service watcher registration succeeds, then metadata generator creation
	// fails because Namespace metadata is disabled. The constructor must roll
	// back the provisional Service watcher ownership before returning.
	enricher := newFailingStateServiceEnricher(t, metricsRepo, resourceWatchers, false)

	require.IsType(t, &nilEnricher{}, enricher, "metadata generator failure must disable enrichment")
	assert.True(t, enricher.Start(resourceWatchers), "nilEnricher.Start() must return true for compatibility")
	resourceWatchers.lock.RLock()
	require.Empty(t, resourceWatchers.metaWatchersMap, "constructor rollback must evict an unstarted final-owner watcher")
	resourceWatchers.lock.RUnlock()
}

func TestNewResourceMetadataEnricherRollbackDoesNotUpgradeNodeScopedOwner(t *testing.T) {
	resourceWatchers := NewWatchers()
	metricsRepo := NewMetricsRepo()
	active := newMockWatcher()
	active.AddEventHandler(kubernetes.ResourceEventHandlerFuncs{})
	metaWatcher := &metaWatcher{
		watcher:   active,
		started:   true,
		users:     make(map[*enricher]watcherRegistration),
		enrichers: make(map[*enricher]struct{}),
		nodeScope: true,
	}
	resourceWatchers.metaWatchersMap[ServiceResource] = metaWatcher

	config := &kubernetesConfig{AddResourceMetadata: &metadata.AddResourceMetadataConfig{}}
	log := logptest.NewTestingLogger(t, selector)
	funcs := mockFuncs{}
	nodeScoped := buildTestMetadataEnricherWithScope("service", ServiceResource, resourceWatchers, config, &funcs, log, true)

	// The cluster-scoped constructor prepares a replacement for the existing
	// node-scoped watcher, then fails because Namespace metadata is disabled.
	// Its real rollback path must discard that provisional replacement.
	enricher := newFailingStateServiceEnricher(t, metricsRepo, resourceWatchers, false)

	require.IsType(t, &nilEnricher{}, enricher, "metadata generator failure must disable enrichment")
	assert.True(t, enricher.Start(resourceWatchers), "nilEnricher.Start() must return true for compatibility")
	resourceWatchers.lock.RLock()
	require.Nil(t, metaWatcher.replacementWatcher, "rolling back the last cluster-scoped registration must discard its pending replacement")
	require.Nil(t, metaWatcher.replacementWatcherFactory, "rollback must discard the replacement factory")
	require.True(t, metaWatcher.nodeScope, "rollback must preserve the active node-scoped watcher's scope")
	resourceWatchers.lock.RUnlock()

	assert.True(t, nodeScoped.Start(resourceWatchers), "Start() must return true")
	require.Equal(t, 0, active.stopCalls, "remaining node-scoped owner must not stop the active watcher")
	require.Same(t, active, metaWatcher.watcher, "remaining node-scoped owner must keep the active watcher")

	nodeScoped.Stop(resourceWatchers)
}

func TestPendingScopeUpgradeRetainedForCommittedClusterScopedOwner(t *testing.T) {
	resourceWatchers := NewWatchers()
	active := newMockWatcher()
	pending := newMockWatcher()
	metaWatcher := &metaWatcher{
		watcher:            active,
		started:            true,
		users:              make(map[*enricher]watcherRegistration),
		enrichers:          make(map[*enricher]struct{}),
		nodeScope:          true,
		replacementWatcher: pending,
		replacementWatcherFactory: func() (kubernetes.Watcher, error) {
			return newMockWatcher(), nil
		},
	}
	resourceWatchers.metaWatchersMap[PodResource] = metaWatcher

	config := &kubernetesConfig{AddResourceMetadata: &metadata.AddResourceMetadataConfig{}}
	log := logptest.NewTestingLogger(t, selector)
	funcs := mockFuncs{}
	nodeScoped := buildTestMetadataEnricherWithScope("pod", PodResource, resourceWatchers, config, &funcs, log, true)
	firstClusterScoped := buildTestMetadataEnricher("state_pod", PodResource, resourceWatchers, config, &funcs, log)
	secondClusterScoped := buildTestMetadataEnricher("state_container", PodResource, resourceWatchers, config, &funcs, log)

	firstClusterScoped.Stop(resourceWatchers)
	resourceWatchers.lock.RLock()
	require.Same(t, pending, metaWatcher.replacementWatcher, "another committed cluster-scoped owner must retain the pending replacement")
	require.NotNil(t, metaWatcher.replacementWatcherFactory, "another committed cluster-scoped owner must retain the replacement factory")
	resourceWatchers.lock.RUnlock()

	secondClusterScoped.Stop(resourceWatchers)
	resourceWatchers.lock.RLock()
	require.Nil(t, metaWatcher.replacementWatcher, "the last cluster-scoped owner must discard the pending replacement")
	require.Nil(t, metaWatcher.replacementWatcherFactory, "the last cluster-scoped owner must discard the replacement factory")
	require.True(t, metaWatcher.nodeScope, "discarding a pending replacement must preserve the active watcher scope")
	resourceWatchers.lock.RUnlock()

	nodeScoped.Stop(resourceWatchers)
}

func TestNodeScopedStartDoesNotActivateStagedUpgrade(t *testing.T) {
	resourceWatchers := NewWatchers()
	active := newMockWatcher()
	pending := newMockWatcher()
	metaWatcher := &metaWatcher{
		watcher:   active,
		started:   true,
		users:     make(map[*enricher]watcherRegistration),
		enrichers: make(map[*enricher]struct{}),
		nodeScope: true,
	}
	resourceWatchers.metaWatchersMap[PodResource] = metaWatcher

	config := &kubernetesConfig{AddResourceMetadata: &metadata.AddResourceMetadataConfig{}}
	log := logptest.NewTestingLogger(t, selector)
	funcs := mockFuncs{}
	nodeScoped := buildTestMetadataEnricherWithScope("pod", PodResource, resourceWatchers, config, &funcs, log, true)
	provisional := newMetadataEnricher("state_pod", PodResource, config, log)

	resourceWatchers.lock.Lock()
	metaWatcher.replacementWatcher = pending
	registerWatcherUser(PodResource, metaWatcher, provisional, true, false)
	started := make(chan struct{})
	go func() {
		nodeScoped.Start(resourceWatchers)
		close(started)
	}()
	resourceWatchers.lock.Unlock()
	<-started

	require.Equal(t, 0, pending.startCalls, "Start must not apply a scope change required only by a provisional owner")
	require.Equal(t, 0, active.stopCalls, "Start must leave the active watcher uninterrupted")
	require.Same(t, pending, metaWatcher.replacementWatcher, "the provisional owner's replacement must remain available during construction")

	releaseWatcherOwnership(provisional, resourceWatchers)
	nodeScoped.Stop(resourceWatchers)
}

func TestFailedScopeUpgradeLeavesActiveWatcherRunning(t *testing.T) {
	resourceWatchers := NewWatchers()
	active := newMockWatcher()
	firstFailed := newMockWatcher()
	firstFailed.startErr = fmt.Errorf("replacement start failed")
	secondFailed := newMockWatcher()
	secondFailed.startErr = fmt.Errorf("replacement start failed again")
	replacement := newMockWatcher()
	replacements := []kubernetes.Watcher{secondFailed, replacement}
	factoryCalls := 0
	metaWatcher := &metaWatcher{
		watcher:            active,
		started:            true,
		users:              make(map[*enricher]watcherRegistration),
		enrichers:          make(map[*enricher]struct{}),
		nodeScope:          true,
		replacementWatcher: firstFailed,
		replacementWatcherFactory: func() (kubernetes.Watcher, error) {
			watcher := replacements[factoryCalls]
			factoryCalls++
			return watcher, nil
		},
	}
	resourceWatchers.metaWatchersMap[PodResource] = metaWatcher

	config := &kubernetesConfig{AddResourceMetadata: &metadata.AddResourceMetadataConfig{}}
	log := logptest.NewTestingLogger(t, selector)
	funcs := mockFuncs{}
	nodeScoped := buildTestMetadataEnricherWithScope("pod", PodResource, resourceWatchers, config, &funcs, log, true)
	clusterScoped := buildTestMetadataEnricher("state_pod", PodResource, resourceWatchers, config, &funcs, log)

	assert.False(t, clusterScoped.Start(resourceWatchers), "Start() must return false when replacement fails")
	require.Equal(t, 1, firstFailed.startCalls, "pending replacement must be attempted")
	require.Equal(t, 1, firstFailed.stopCalls, "failed replacement must be stopped")
	require.Equal(t, 0, active.stopCalls, "failed replacement must not stop the active watcher")
	require.Same(t, active, metaWatcher.watcher, "failed replacement must not replace the active watcher")
	require.Nil(t, metaWatcher.replacementWatcher, "failed replacement must be discarded")
	require.True(t, metaWatcher.nodeScope, "failed replacement must preserve the active watcher scope")
	require.True(t, metaWatcher.started, "failed replacement must preserve active watcher state")

	assert.False(t, clusterScoped.Start(resourceWatchers), "Start() must return false on repeated replacement failure")
	require.Equal(t, 1, secondFailed.startCalls, "next Start must attempt a fresh replacement")
	require.Equal(t, 1, secondFailed.stopCalls, "each failed replacement must be stopped")
	require.Equal(t, 1, firstFailed.startCalls, "failed replacement must not be reused")
	require.Equal(t, 0, active.stopCalls, "repeated replacement failures must preserve the active watcher")

	assert.True(t, clusterScoped.Start(resourceWatchers), "Start() must return true after successful scope upgrade")
	require.Equal(t, 2, factoryCalls, "each retry must construct one fresh replacement")
	require.Equal(t, 1, replacement.startCalls, "fresh replacement must be started")
	require.Equal(t, 1, active.stopCalls, "active watcher must stop only after its replacement starts")
	require.Same(t, replacement, metaWatcher.watcher, "successful replacement must become active")
	require.Nil(t, metaWatcher.replacementWatcher, "successful scope upgrade must clear the pending replacement")
	require.Nil(t, metaWatcher.replacementWatcherFactory, "successful scope upgrade must clear the replacement factory")
	require.False(t, metaWatcher.nodeScope, "successful replacement must restore cluster-scoped coverage")

	clusterScoped.Stop(resourceWatchers)
	nodeScoped.Stop(resourceWatchers)
}

func TestNodeScopeReplacementWatcherLifecycle(t *testing.T) {
	resourceWatchers := NewWatchers()
	active := newMockWatcher()
	replacement := newMockWatcher()
	metaWatcher := &metaWatcher{
		watcher:            active,
		started:            true,
		users:              make(map[*enricher]watcherRegistration),
		enrichers:          make(map[*enricher]struct{}),
		nodeScope:          true,
		replacementWatcher: replacement,
	}
	resourceWatchers.metaWatchersMap[PodResource] = metaWatcher

	config := &kubernetesConfig{AddResourceMetadata: &metadata.AddResourceMetadataConfig{}}
	log := logptest.NewTestingLogger(t, selector)
	funcs := mockFuncs{}
	nodeScoped := buildTestMetadataEnricherWithScope("pod", PodResource, resourceWatchers, config, &funcs, log, true)
	clusterScoped := buildTestMetadataEnricher("state_pod", PodResource, resourceWatchers, config, &funcs, log)

	assert.True(t, clusterScoped.Start(resourceWatchers), "Start() must return true after scope upgrade")
	require.Equal(t, 1, active.stopCalls, "scope upgrade must stop the old active watcher")
	require.Equal(t, 1, replacement.startCalls, "scope upgrade must start the pending cluster-scoped watcher")
	require.Same(t, replacement, metaWatcher.watcher, "replacement must become the active watcher")
	require.Nil(t, metaWatcher.replacementWatcher, "successful scope upgrade must clear the pending watcher")

	clusterScoped.Stop(resourceWatchers)
	require.Equal(t, 0, replacement.stopCalls, "node-scoped owner must retain the replacement watcher")
	nodeScoped.Stop(resourceWatchers)
	require.Equal(t, 1, replacement.stopCalls, "final owner must stop the active replacement watcher")
	resourceWatchers.lock.RLock()
	require.NotContains(t, resourceWatchers.metaWatchersMap, PodResource, "final owner must evict the scope-upgraded watcher")
	resourceWatchers.lock.RUnlock()
}

func TestFinalOwnerEvictionDiscardsPendingReplacementWatcher(t *testing.T) {
	resourceWatchers := NewWatchers()
	active := newMockWatcher()
	pending := newMockWatcher()
	metaWatcher := &metaWatcher{
		watcher:            active,
		started:            true,
		users:              make(map[*enricher]watcherRegistration),
		enrichers:          make(map[*enricher]struct{}),
		nodeScope:          true,
		replacementWatcher: pending,
	}
	resourceWatchers.metaWatchersMap[PodResource] = metaWatcher

	config := &kubernetesConfig{AddResourceMetadata: &metadata.AddResourceMetadataConfig{}}
	log := logptest.NewTestingLogger(t, selector)
	funcs := mockFuncs{}
	e := buildTestMetadataEnricher("state_pod", PodResource, resourceWatchers, config, &funcs, log)

	e.Stop(resourceWatchers)
	require.Equal(t, 1, active.stopCalls, "final owner must stop only the active watcher")
	require.Equal(t, 0, pending.stopCalls, "unstarted pending replacement must be discarded without stopping")
	resourceWatchers.lock.RLock()
	require.NotContains(t, resourceWatchers.metaWatchersMap, PodResource, "watcher with a pending replacement must still be evicted")
	resourceWatchers.lock.RUnlock()
}

func TestRealInformerIsRecreatedAfterFinalOwnerStops(t *testing.T) {
	clientfeaturestesting.SetFeatureDuringTest(t, clientfeatures.WatchListClient, false)

	resourceWatchers := NewWatchers()
	metricsRepo := NewMetricsRepo()
	client := k8sfake.NewSimpleClientset()
	metadataClient := k8smetafake.NewSimpleMetadataClient(k8smetafake.NewTestScheme())
	log := logptest.NewTestingLogger(t, selector)
	config := &kubernetesConfig{
		Namespace:           "default",
		SyncPeriod:          5 * time.Second,
		AddResourceMetadata: resourceMetadataConfig(t, false, false, false, false),
	}

	createGeneration := func() (*enricher, *enricher, kubernetes.Watcher) {
		pod := newMetadataEnricher("pod", PodResource, config, log)
		require.NoError(
			t,
			createAllWatchers(client, metadataClient, pod, false, config, log, resourceWatchers, metricsRepo),
			"pod watcher creation must succeed",
		)
		configureRealInformerTestEnricher(pod, false)
		commitWatcherOwnership(pod, resourceWatchers)

		container := newMetadataEnricher("container", PodResource, config, log)
		require.NoError(
			t,
			createAllWatchers(client, metadataClient, container, false, config, log, resourceWatchers, metricsRepo),
			"container watcher sharing must succeed",
		)
		configureRealInformerTestEnricher(container, true)
		commitWatcherOwnership(container, resourceWatchers)

		return pod, container, pod.watcher.watcher
	}

	podA, containerA, watcherA := createGeneration()
	t.Cleanup(func() {
		podA.Stop(resourceWatchers)
		containerA.Stop(resourceWatchers)
	})
	assert.True(t, podA.Start(resourceWatchers), "Start() must return true")
	assert.True(t, containerA.Start(resourceWatchers), "Start() must return true")
	_, err := client.CoreV1().Pods("default").Create(context.Background(), informerTestPod("pod-a", "a"), metav1.CreateOptions{})
	require.NoError(t, err, "generation-A pod creation must succeed")
	require.Eventually(t, func() bool {
		_, exists, getErr := watcherA.Store().GetByKey("default/pod-a")
		return getErr == nil && exists
	}, 5*time.Second, 10*time.Millisecond, "generation-A informer must observe pod A")

	podA.Stop(resourceWatchers)
	containerA.Stop(resourceWatchers)
	resourceWatchers.lock.RLock()
	require.NotContains(t, resourceWatchers.metaWatchersMap, PodResource, "generation-A final owner must evict the stopped watcher")
	resourceWatchers.lock.RUnlock()

	podB, containerB, watcherB := createGeneration()
	t.Cleanup(func() {
		podB.Stop(resourceWatchers)
		containerB.Stop(resourceWatchers)
	})
	require.NotEqual(t, watcherA, watcherB, "generation B must receive a fresh watcher and informer lifecycle")
	assert.True(t, podB.Start(resourceWatchers), "Start() must return true")
	assert.True(t, containerB.Start(resourceWatchers), "Start() must return true")
	_, err = client.CoreV1().Pods("default").Create(context.Background(), informerTestPod("pod-b", "b"), metav1.CreateOptions{})
	require.NoError(t, err, "generation-B pod creation must succeed")
	require.Eventually(t, func() bool {
		_, exists, getErr := watcherB.Store().GetByKey("default/pod-b")
		return getErr == nil && exists
	}, 5*time.Second, 10*time.Millisecond, "fresh generation-B informer must observe pod B")

	podEvents := []mapstr.M{{
		"name": podBName,
		mb.ModuleDataKey: mapstr.M{
			"namespace": "default",
		},
	}}
	podB.Enrich(podEvents)
	podLabel, err := podEvents[0].GetValue(mb.ModuleDataKey + ".labels.generation")
	require.NoError(t, err, "pod event must contain generation-B labels")
	require.Equal(t, "b", podLabel, "pod event must be enriched from the fresh informer")

	containerEvents := []mapstr.M{{
		"name": informerTestContainerName,
		mb.ModuleDataKey: mapstr.M{
			"namespace": "default",
			"pod":       mapstr.M{"name": podBName},
		},
	}}
	containerB.Enrich(containerEvents)
	containerLabel, err := containerEvents[0].GetValue(mb.ModuleDataKey + ".labels.generation")
	require.NoError(t, err, "container event must contain generation-B labels")
	require.Equal(t, "b", containerLabel, "container event must be enriched from the fresh informer")
}

func TestConcurrentInvalidationAndEnrichment(t *testing.T) {
	resourceWatchers := NewWatchers()
	log := logptest.NewTestingLogger(t, selector)
	config := &kubernetesConfig{AddResourceMetadata: &metadata.AddResourceMetadataConfig{}}
	resource := &kubernetes.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deployment",
			Namespace: "default",
		},
	}

	watcher := newMockWatcher()
	metaWatcher := &metaWatcher{
		watcher:     watcher,
		users:       make(map[*enricher]watcherRegistration),
		enrichers:   make(map[*enricher]struct{}),
		metricsRepo: NewMetricsRepo(),
	}
	resourceWatchers.lock.Lock()
	resourceWatchers.metaWatchersMap[DeploymentResource] = metaWatcher
	addEventHandlersToWatcher(metaWatcher, resourceWatchers)
	resourceWatchers.lock.Unlock()

	e := buildTestMetadataEnricherWithFuncs(
		"state_deployment",
		DeploymentResource,
		resourceWatchers,
		config,
		func(resource kubernetes.Resource) map[string]mapstr.M {
			//nolint:errcheck // we know the type
			deployment := resource.(*kubernetes.Deployment)
			return map[string]mapstr.M{
				join(deployment.Namespace, deployment.Name): {
					"kubernetes": mapstr.M{"labels": mapstr.M{"generation": "current"}},
				},
			}
		},
		func(resource kubernetes.Resource) []string {
			deployment := resource.(*kubernetes.Deployment) //nolint:errcheck // It's a test, let it panic
			return []string{join(deployment.Namespace, deployment.Name)}
		},
		func(event mapstr.M) string {
			return join(getString(event, mb.ModuleDataKey+".namespace"), getString(event, "name"))
		},
		log,
	)
	require.NoError(t, watcher.Store().Add(resource), "deployment must be added to the mock watcher store")
	assert.True(t, e.Start(resourceWatchers), "Start() must return true")

	var workers sync.WaitGroup
	workers.Go(func() {
		for range 100 {
			watcher.handler.OnUpdate(resource)
		}
	})
	workers.Go(func() {
		for range 100 {
			e.Enrich([]mapstr.M{{
				"name":           resource.Name,
				mb.ModuleDataKey: mapstr.M{"namespace": resource.Namespace},
			}})
		}
	})

	workers.Wait()
	e.Stop(resourceWatchers)
}

func TestActiveWatcherLookupBlocksScopeReplacement(t *testing.T) {
	resourceWatchers := NewWatchers()
	lookupStarted := make(chan struct{})
	releaseLookup := make(chan struct{})
	active := newCoordinatedWatcher(&blockingStore{
		Store:         newMockWatcher().store,
		lookupStarted: lookupStarted,
		releaseLookup: releaseLookup,
	})
	replacement := newCoordinatedWatcher(newMockWatcher().store)
	activeResource := &kubernetes.Pod{ObjectMeta: metav1.ObjectMeta{
		Name:   "resource",
		Labels: map[string]string{"source": "active"},
	}}
	replacementResource := activeResource.DeepCopy()
	replacementResource.Labels = map[string]string{"source": "replacement"}
	require.NoError(t, active.Store().Add(activeResource), "active watcher store must contain the resource")
	require.NoError(t, replacement.Store().Add(replacementResource), "replacement watcher store must contain the resource")

	metaWatcher := &metaWatcher{
		watcher:            active,
		started:            true,
		users:              make(map[*enricher]watcherRegistration),
		enrichers:          make(map[*enricher]struct{}),
		nodeScope:          true,
		replacementWatcher: replacement,
	}
	resourceWatchers.metaWatchersMap[PodResource] = metaWatcher
	e := newWatcherLookupTestEnricher(t, resourceWatchers, metaWatcher)

	lookupDone := make(chan mapstr.M, 1)
	go func() {
		lookupDone <- e.getMetadata(mapstr.M{"name": activeResource.Name})
	}()
	<-lookupStarted

	startDone := make(chan struct{})
	go func() {
		e.Start(resourceWatchers)
		close(startDone)
	}()
	<-replacement.started
	requireActiveWatcherWriterPending(t, metaWatcher, "scope replacement must wait for the in-progress lookup")
	select {
	case <-active.stopped:
		t.Fatal("active watcher stopped while its store lookup was in progress")
	default:
	}

	close(releaseLookup)
	require.Equal(t, "active", (<-lookupDone)["source"], "in-progress lookup must finish against the active watcher")
	<-startDone
	<-active.stopped

	e.Lock()
	e.metadataCache = make(map[string]mapstr.M)
	e.Unlock()
	require.Equal(t, "replacement", e.getMetadata(mapstr.M{"name": activeResource.Name})["source"], "later lookup must use the replacement watcher")

	e.Stop(resourceWatchers)
}

// TestScopeUpgradeKeepsOldWatcherActiveDuringCacheSync verifies that during a
// scope upgrade the old node-scoped watcher W remains the active watcher while
// the replacement cluster-scoped watcher R's WaitForCacheSync runs. The swap to
// R happens only after R.Start() returns, so metadata lookups never see an empty
// store (no blackout window).
func TestScopeUpgradeKeepsOldWatcherActiveDuringCacheSync(t *testing.T) {
	resourceWatchers := NewWatchers()

	// W: node-scoped watcher with a pod already in its store.
	oldWatcher := newMockWatcher()
	pod := &kubernetes.Pod{ObjectMeta: metav1.ObjectMeta{
		Name:      "my-pod",
		Namespace: "default",
		Labels:    map[string]string{"source": "old"},
	}}
	require.NoError(t, oldWatcher.Store().Add(pod))

	// R: replacement cluster-scoped watcher; empty store; blocks in Start()
	// until Stop() is called (simulating RBAC-gated WaitForCacheSync).
	replacementW := newCancelErrorWatcher()

	mw := &metaWatcher{
		watcher:            oldWatcher,
		started:            true,
		nodeScope:          true,
		replacementWatcher: replacementW,
		users:              make(map[*enricher]watcherRegistration),
		enrichers:          make(map[*enricher]struct{}),
	}
	resourceWatchers.metaWatchersMap[PodResource] = mw
	e := newWatcherLookupTestEnricher(t, resourceWatchers, mw)

	// Before the upgrade the old watcher serves "my-pod".
	got := e.getMetadata(mapstr.M{"name": pod.Name})
	require.NotNil(t, got, "old watcher must serve my-pod before the scope upgrade")
	e.Lock()
	e.metadataCache = make(map[string]mapstr.M) // clear so the next lookup hits the store
	e.Unlock()

	// Start() sets pendingReplacement=R and releases the lock, then blocks in R.Start().
	// It deliberately leaves starting alone (that flag tracks W's own startup), and
	// W remains the active watcher for the whole of R's WaitForCacheSync.
	startDone := make(chan struct{})
	go func() {
		e.Start(resourceWatchers)
		close(startDone)
	}()
	<-replacementW.startCalled // R.Start() is in progress; W is still the active watcher

	// During R's WaitForCacheSync, W stays active: metadata lookups must not
	// return nil. The swap to R happens only after R.Start() succeeds.
	got = e.getMetadata(mapstr.M{"name": pod.Name})
	assert.NotNil(t, got, "W must remain active during R's WaitForCacheSync; scope upgrade must not create a metadata blackout")

	// Capture the attempt's done channel before Stop(); we must wait for the
	// claimReplacement goroutine to close it before the test ends (goroutine races the
	// testing.T cleanup). The goroutine will not log: ownership guard prevents it.
	resourceWatchers.lock.RLock()
	attemptDone := mw.pendingReplacement.done
	resourceWatchers.lock.RUnlock()

	// Stop() withdraws W, stops R via pendingReplacement (unblocking R.Start()).
	e.Stop(resourceWatchers)
	select {
	case <-startDone:
	case <-time.After(5 * time.Second):
		t.Fatal("e.Start() goroutine did not finish after Stop()")
	}
	select {
	case <-attemptDone:
	case <-time.After(5 * time.Second):
		t.Fatal("claimReplacement goroutine did not finish after Stop()")
	}
}

// TestUpgradeCancelledWhenClusterScopedOwnerStops verifies the
// else-if-!hasClusterScopedUser branch of releaseWatcherOwnership: when a scope
// upgrade is in flight (pendingReplacement != nil) and the only cluster-scoped owner
// stops, Stop() cancels the upgrade by stopping R, which unblocks R.Start().
// nodeScope stays true throughout because W was never swapped out, and starting
// is left alone because it belongs to W's own startup goroutine.
func TestUpgradeCancelledWhenClusterScopedOwnerStops(t *testing.T) {
	resourceWatchers := NewWatchers()

	// W: already-started node-scoped watcher.
	w := newMockWatcher()
	pod := &kubernetes.Pod{ObjectMeta: metav1.ObjectMeta{Name: "my-pod", Namespace: "default"}}
	require.NoError(t, w.Store().Add(pod))

	// R: pending restart; blocks in Start() until Stop() is called.
	r := newBlockingMockWatcher()

	mw := &metaWatcher{
		watcher:            w,
		started:            true,
		nodeScope:          true, // W is still the active watcher, so still node-scoped
		pendingReplacement: &watcherStart{watcher: r, done: make(chan struct{})},
		users:              make(map[*enricher]watcherRegistration),
		enrichers:          make(map[*enricher]struct{}),
	}
	resourceWatchers.metaWatchersMap[PodResource] = mw

	config := &kubernetesConfig{AddResourceMetadata: &metadata.AddResourceMetadataConfig{}}
	log := logptest.NewTestingLogger(t, selector)

	// E1: node-scoped, stays alive after E2 stops.
	e1 := newMetadataEnricher("pod", PodResource, config, log)
	e1.watcher = mw
	e1.watchedResources = []string{PodResource}
	mw.users[e1] = watcherRegistration{committed: true, nodeScope: true}
	mw.enrichers[e1] = struct{}{}

	// E2: the only cluster-scoped owner; its Stop() triggers the cancel path.
	e2 := newMetadataEnricher("state_pod", PodResource, config, log)
	e2.watcher = mw
	e2.watchedResources = []string{PodResource}
	mw.users[e2] = watcherRegistration{committed: true, nodeScope: false}
	mw.enrichers[e2] = struct{}{}

	// A goroutine holds R.Start() open; it unblocks when R.Stop() is called.
	rStartDone := make(chan struct{})
	go func() {
		_ = r.Start()
		close(rStartDone)
	}()
	<-r.startCalled

	// E2 stops. It is the only cluster-scoped user, so releaseWatcherOwnership
	// must cancel the upgrade by stopping R, which unblocks rStartDone.
	e2.Stop(resourceWatchers)

	select {
	case <-rStartDone:
	case <-time.After(5 * time.Second):
		t.Fatal("R.Start() did not unblock after E2.Stop(); Stop() must have called R.Stop()")
	}

	resourceWatchers.lock.RLock()
	assert.Nil(t, mw.starting, "starting must remain nil; cancel-upgrade does not touch it")
	assert.True(t, mw.nodeScope, "nodeScope must stay true; W was never swapped out")
	assert.Nil(t, mw.pendingReplacement, "pendingReplacement must be nil after upgrade cancellation")
	assert.True(t, mw.started, "W must still be started; only E2 (non-final) stopped")
	resourceWatchers.lock.RUnlock()

	e1.Stop(resourceWatchers)
}

// TestUpgradeCancelDoesNotClearConcurrentActiveStart verifies that cancelling
// a scope upgrade does not clear an unrelated startup claim for the active
// watcher.
func TestUpgradeCancelDoesNotClearConcurrentActiveStart(t *testing.T) {
	resourceWatchers := NewWatchers()

	// The active watcher blocks inside Start until it is stopped.
	w := newBlockingMockWatcher()
	// The in-flight replacement also blocks inside Start until it is stopped.
	r := newBlockingMockWatcher()

	mw := &metaWatcher{
		watcher:            w,
		started:            false,
		nodeScope:          true,
		pendingReplacement: &watcherStart{watcher: r, done: make(chan struct{})},
		users:              make(map[*enricher]watcherRegistration),
		enrichers:          make(map[*enricher]struct{}),
	}
	resourceWatchers.metaWatchersMap[PodResource] = mw

	config := &kubernetesConfig{AddResourceMetadata: &metadata.AddResourceMetadataConfig{}}
	log := logptest.NewTestingLogger(t, selector)

	// The node-scoped owner starts the active watcher.
	e1 := newMetadataEnricher("pod", PodResource, config, log)
	e1.watcher = mw
	e1.watchedResources = []string{PodResource}
	mw.users[e1] = watcherRegistration{committed: true, nodeScope: true}
	mw.enrichers[e1] = struct{}{}

	// The cluster-scoped owner is the only owner that requires the upgrade.
	e2 := newMetadataEnricher("state_pod", PodResource, config, log)
	e2.watcher = mw
	e2.watchedResources = []string{PodResource}
	mw.users[e2] = watcherRegistration{committed: true, nodeScope: false}
	mw.enrichers[e2] = struct{}{}

	// Start the replacement attempt.
	rStartDone := make(chan struct{})
	go func() {
		_ = r.Start()
		close(rStartDone)
	}()
	<-r.startCalled

	// Start the active watcher and wait until its startup attempt is in flight.
	e1StartDone := make(chan struct{})
	go func() {
		e1.Start(resourceWatchers)
		close(e1StartDone)
	}()
	<-w.startCalled

	resourceWatchers.lock.RLock()
	require.NotNil(t, mw.starting, "Start() must record the active watcher startup attempt before blocking")
	resourceWatchers.lock.RUnlock()

	// Cancelling the upgrade must stop the replacement without clearing the
	// active watcher's startup claim.
	e2.Stop(resourceWatchers)

	select {
	case <-rStartDone:
	case <-time.After(5 * time.Second):
		t.Fatal("replacement Start() did not unblock; upgrade cancellation must stop it")
	}

	resourceWatchers.lock.RLock()
	starting := mw.starting
	assert.Nil(t, mw.pendingReplacement, "pendingReplacement must be nil after upgrade cancellation")
	assert.True(t, mw.nodeScope, "nodeScope must stay true; the active watcher was never replaced")
	resourceWatchers.lock.RUnlock()
	require.NotNil(t, starting, "the active watcher startup claim must survive upgrade cancellation")
	startAttemptDone := starting.done

	// Releasing the final owner detaches and stops the active startup attempt.
	// launchStart then skips the stale commit and closes the attempt's done channel.
	e1.Stop(resourceWatchers)

	select {
	case <-e1StartDone:
	case <-time.After(5 * time.Second):
		t.Fatal("active watcher Start() did not unblock after the final Stop()")
	}
	select {
	case <-startAttemptDone:
	case <-time.After(5 * time.Second):
		t.Fatal("launchStart goroutine did not finish after Stop()")
	}
}

func TestActiveWatcherLookupCompletesBeforeFinalOwnerStop(t *testing.T) {
	resourceWatchers := NewWatchers()
	lookupStarted := make(chan struct{})
	releaseLookup := make(chan struct{})
	active := newCoordinatedWatcher(&blockingStore{
		Store:         newMockWatcher().store,
		lookupStarted: lookupStarted,
		releaseLookup: releaseLookup,
	})
	resource := &kubernetes.Pod{ObjectMeta: metav1.ObjectMeta{
		Name:   "resource",
		Labels: map[string]string{"source": "active"},
	}}
	require.NoError(t, active.Store().Add(resource), "active watcher store must contain the resource")

	metaWatcher := &metaWatcher{
		watcher:   active,
		started:   true,
		users:     make(map[*enricher]watcherRegistration),
		enrichers: make(map[*enricher]struct{}),
	}
	resourceWatchers.metaWatchersMap[PodResource] = metaWatcher
	e := newWatcherLookupTestEnricher(t, resourceWatchers, metaWatcher)

	lookupDone := make(chan mapstr.M, 1)
	go func() {
		lookupDone <- e.getMetadata(mapstr.M{"name": resource.Name})
	}()
	<-lookupStarted

	stopDone := make(chan struct{})
	go func() {
		e.Stop(resourceWatchers)
		close(stopDone)
	}()

	requireActiveWatcherWriterPending(t, metaWatcher, "final-owner stop must wait for the in-progress lookup")
	close(releaseLookup)
	require.Equal(t, "active", (<-lookupDone)["source"], "in-progress lookup must finish before final-owner stop")
	<-stopDone
	<-active.stopped

	e.Lock()
	e.metadataCache = make(map[string]mapstr.M)
	e.Unlock()
	require.Nil(t, e.getMetadata(mapstr.M{"name": resource.Name}), "lookup after withdrawal must not use the stopped watcher")
}

// TestReplacementWatcherOldWatcherStoppedAfterConcurrentStop covers the race where
// (a) ensurePrimaryReady → claimReplacement sets pendingReplacement=R, keeps W as the
//
//	active watcher, releases the lock, then the goroutine blocks in R.Start(), and
//
// (b) a concurrent Stop() withdraws W (stopping it) and stops R via pendingReplacement,
//
//	which unblocks R.Start() with an error, and
//
// (c) the claimReplacement goroutine acquires the lock, finds the registry entry gone
//
//	(Stop() was the final owner and deleted it), and closes attempt.done.
//	Stop() had already cleared pendingReplacement and stopped both W and R.
func TestReplacementWatcherOldWatcherStoppedAfterConcurrentStop(t *testing.T) {
	resourceWatchers := NewWatchers()
	oldWatcher := newCoordinatedWatcher(newMockWatcher().store)
	replacementWatcher := newCancelErrorWatcher()

	mw := &metaWatcher{
		watcher:            oldWatcher,
		started:            true,
		users:              make(map[*enricher]watcherRegistration),
		enrichers:          make(map[*enricher]struct{}),
		nodeScope:          true,
		replacementWatcher: replacementWatcher,
	}
	resourceWatchers.metaWatchersMap[PodResource] = mw
	e := newWatcherLookupTestEnricher(t, resourceWatchers, mw)

	startDone := make(chan struct{})
	go func() {
		e.Start(resourceWatchers)
		close(startDone)
	}()
	<-replacementWatcher.startCalled // Start() set pendingReplacement=R and released the lock.

	// Capture the attempt's done channel before Stop(); we must wait for the
	// claimReplacement goroutine to close it before the test ends (goroutine races the
	// testing.T cleanup). The goroutine will not log: ownership guard prevents it.
	resourceWatchers.lock.RLock()
	attemptDone := mw.pendingReplacement.done
	resourceWatchers.lock.RUnlock()

	// Stop() is the final owner release: it withdraws the active watcher (still W)
	// and separately stops pendingReplacement (R). Stopping R unblocks R.Start(), which
	// then returns an error.
	e.Stop(resourceWatchers)

	select {
	case <-startDone:
	case <-time.After(5 * time.Second):
		t.Fatal("e.Start() goroutine did not finish after Stop()")
	}
	select {
	case <-oldWatcher.stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("old watcher W was not stopped by Stop()")
	}
	select {
	case <-attemptDone:
	case <-time.After(5 * time.Second):
		t.Fatal("claimReplacement goroutine did not finish after Stop()")
	}
}

func TestBuildMetadataEnricher_EventHandler(t *testing.T) {
	resourceWatchers := NewWatchers()
	metricsRepo := NewMetricsRepo()

	resourceWatchers.lock.Lock()
	watcher := &metaWatcher{
		watcher:     newMockWatcher(),
		started:     false,
		users:       make(map[*enricher]watcherRegistration),
		enrichers:   make(map[*enricher]struct{}),
		metricsRepo: metricsRepo,
	}
	resourceWatchers.metaWatchersMap[PodResource] = watcher
	addEventHandlersToWatcher(watcher, resourceWatchers)
	resourceWatchers.lock.Unlock()

	funcs := mockFuncs{}
	resource := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			UID:  types.UID("mockuid"),
			Name: "enrich",
			Labels: map[string]string{
				"label": "value",
			},
			Namespace: "default",
		},
	}
	events := []mapstr.M{
		{"name": "unknown"},
		{"name": "enrich"},
	}

	config := &kubernetesConfig{
		Namespace:  "test-ns",
		SyncPeriod: time.Minute,
		Node:       "test-node",
		AddResourceMetadata: &metadata.AddResourceMetadataConfig{
			CronJob:    false,
			Deployment: false,
		},
	}

	metricset := "pod"
	log := logptest.NewTestingLogger(t, selector)

	enricher := buildTestMetadataEnricher(metricset, PodResource, resourceWatchers, config, &funcs, log)
	resourceWatchers.lock.Lock()
	wData := resourceWatchers.metaWatchersMap[PodResource]
	mockW, ok := wData.watcher.(*mockWatcher)
	require.True(t, ok)
	require.NotNil(t, mockW.handler)
	resourceWatchers.lock.Unlock()

	assert.True(t, enricher.Start(resourceWatchers), "Start() must return true after successful watcher setup")
	resourceWatchers.lock.Lock()
	require.True(t, watcher.started)
	resourceWatchers.lock.Unlock()

	mockW.handler.OnAdd(resource)
	err := mockW.Store().Add(resource)
	require.NoError(t, err)

	// Test enricher

	enricher.Enrich(events)

	require.Equal(t, []mapstr.M{
		{"name": "unknown"},
		{
			"name":    "enrich",
			"_module": mapstr.M{"label": "value", "pod": mapstr.M{"name": "enrich", "uid": "mockuid"}},
			"meta":    mapstr.M{"orchestrator": mapstr.M{"cluster": mapstr.M{"name": "gke-4242"}}},
		},
	}, events)

	require.Equal(t, resource, funcs.updated)

	// Enrich a pod (metadata goes in root level)
	events = []mapstr.M{
		{"name": "unknown"},
		{"name": "enrich"},
	}
	enricher.isPod = true
	enricher.Enrich(events)

	require.Equal(t, []mapstr.M{
		{"name": "unknown"},
		{
			"name":    "enrich",
			"uid":     "mockuid",
			"_module": mapstr.M{"label": "value"},
			"meta":    mapstr.M{"orchestrator": mapstr.M{"cluster": mapstr.M{"name": "gke-4242"}}},
		},
	}, events)

	// Emit delete event
	resourceWatchers.lock.Lock()
	wData = resourceWatchers.metaWatchersMap[PodResource]
	mockW, ok = wData.watcher.(*mockWatcher)
	require.True(t, ok)
	resourceWatchers.lock.Unlock()

	mockW.handler.OnDelete(resource)
	err = mockW.Store().Delete(resource)
	require.NoError(t, err)

	require.Equal(t, resource, funcs.deleted)

	events = []mapstr.M{
		{"name": "unknown"},
		{"name": "enrich"},
	}
	enricher.Enrich(events)

	require.Equal(t, []mapstr.M{
		{"name": "unknown"},
		{"name": "enrich"},
	}, events)

	enricher.Stop(resourceWatchers)
	resourceWatchers.lock.Lock()
	require.NotContains(t, resourceWatchers.metaWatchersMap, PodResource, "final owner must evict the watcher")
	resourceWatchers.lock.Unlock()
}

func TestBuildMetadataEnricher_PartialMetadata(t *testing.T) {
	resourceWatchers := NewWatchers()
	metricsRepo := NewMetricsRepo()

	resourceWatchers.lock.Lock()
	watcher := &metaWatcher{
		watcher: &mockWatcher{
			store: cache.NewStore(cache.MetaNamespaceKeyFunc),
		},
		started:     false,
		users:       make(map[*enricher]watcherRegistration),
		enrichers:   make(map[*enricher]struct{}),
		metricsRepo: metricsRepo,
	}
	resourceWatchers.metaWatchersMap[ReplicaSetResource] = watcher
	addEventHandlersToWatcher(watcher, resourceWatchers)
	resourceWatchers.lock.Unlock()

	isController := true
	resource := &metav1.PartialObjectMetadata{
		ObjectMeta: metav1.ObjectMeta{
			UID:  types.UID("mockuid"),
			Name: "enrich",
			Labels: map[string]string{
				"label": "value",
			},
			Namespace: "default",
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       "enrich_deployment",
					Controller: &isController,
				},
			},
		},
	}

	config := &kubernetesConfig{
		Namespace:  "test-ns",
		SyncPeriod: time.Minute,
		Node:       "test-node",
		AddResourceMetadata: &metadata.AddResourceMetadataConfig{
			CronJob:    false,
			Deployment: true,
		},
	}

	metricset := "replicaset"
	log := logptest.NewTestingLogger(t, selector)

	commonMetaConfig := metadata.Config{}
	commonConfig, _ := conf.NewConfigFrom(&commonMetaConfig)
	client := k8sfake.NewSimpleClientset()
	generalMetaGen := metadata.NewResourceMetadataGenerator(commonConfig, client)

	updateFunc := getEventMetadataFunc(log, generalMetaGen, nil)

	deleteFunc := func(r kubernetes.Resource) []string {
		accessor, _ := meta.Accessor(r)
		id := accessor.GetName()
		namespace := accessor.GetNamespace()
		if namespace != "" {
			id = join(namespace, id)
		}
		return []string{id}
	}

	indexFunc := func(e mapstr.M) string {
		name := getString(e, "name")
		namespace := getString(e, mb.ModuleDataKey+".namespace")
		var id string
		if name != "" && namespace != "" {
			id = join(namespace, name)
		} else if namespace != "" {
			id = namespace
		} else {
			id = name
		}
		return id
	}

	enricher := buildTestMetadataEnricherWithFuncs(
		metricset,
		ReplicaSetResource,
		resourceWatchers,
		config,
		updateFunc,
		deleteFunc,
		indexFunc,
		log,
	)

	assert.True(t, enricher.Start(resourceWatchers), "Start() must return true after successful watcher setup")
	resourceWatchers.lock.Lock()
	require.True(t, watcher.started)
	resourceWatchers.lock.Unlock()

	// manually run the transform function here, just like the actual informer
	transformed, err := transformReplicaSetMetadata(resource)
	require.NoError(t, err)
	watcher.watcher.GetEventHandler().OnAdd(transformed)
	err = watcher.watcher.Store().Add(transformed)
	require.NoError(t, err)

	// Test enricher
	events := []mapstr.M{
		// {"name": "unknown"},
		{"name": resource.Name, mb.ModuleDataKey + ".namespace": resource.Namespace},
	}
	enricher.Enrich(events)

	require.Equal(t, []mapstr.M{
		// {"name": "unknown"},
		{
			"name": "enrich",
			"_module": mapstr.M{
				"labels":     mapstr.M{"label": "value"},
				"replicaset": mapstr.M{"name": "enrich", "uid": "mockuid"},
				"namespace":  resource.Namespace,
				"deployment": mapstr.M{
					"name": "enrich_deployment",
				},
			},
			mb.ModuleDataKey + ".namespace": resource.Namespace,
			"meta":                          mapstr.M{},
		},
	}, events)

	watcher.watcher.GetEventHandler().OnDelete(resource)
	err = watcher.watcher.Store().Delete(resource)
	require.NoError(t, err)

	events = []mapstr.M{
		{"name": "enrich"},
	}
	enricher.Enrich(events)

	require.Equal(t, []mapstr.M{
		{"name": "enrich"},
	}, events)

	enricher.Stop(resourceWatchers)
	resourceWatchers.lock.Lock()
	require.NotContains(t, resourceWatchers.metaWatchersMap, ReplicaSetResource, "final owner must evict the watcher")
	resourceWatchers.lock.Unlock()
}

// TestBuildMetadataEnricher_StartStopDeadlock ensures that Stop() can unblock
// a Start() that is blocked inside watcher.Start() (e.g. WaitForCacheSync
// waiting for an informer that never syncs due to missing RBAC).
func TestBuildMetadataEnricher_StartStopDeadlock(t *testing.T) {
	resourceWatchers := NewWatchers()

	config := &kubernetesConfig{
		Namespace:  "test-ns",
		SyncPeriod: time.Minute,
		Node:       "test-node",
		AddResourceMetadata: &metadata.AddResourceMetadataConfig{
			CronJob:    false,
			Deployment: false,
		},
	}
	log := logptest.NewTestingLogger(t, selector)

	blocking := newBlockingMockWatcher()
	mw := &metaWatcher{
		watcher:   blocking,
		started:   false,
		users:     make(map[*enricher]watcherRegistration),
		enrichers: make(map[*enricher]struct{}),
	}
	enricherDeployment := newMetadataEnricher("state_deployment", DeploymentResource, config, log)
	enricherDeployment.watchedResources = []string{DeploymentResource}
	mw.users[enricherDeployment] = watcherRegistration{committed: true, nodeScope: true}

	resourceWatchers.lock.Lock()
	resourceWatchers.metaWatchersMap[DeploymentResource] = mw
	resourceWatchers.lock.Unlock()

	startDone := make(chan struct{})
	go func() {
		enricherDeployment.Start(resourceWatchers)
		close(startDone)
	}()

	// Wait until Start() is blocked inside watcher.Start().
	<-blocking.startCalled

	// Capture the goroutine's done channel before calling Stop() so we can
	// wait for the launchStart goroutine to finish after Start() returns.
	resourceWatchers.lock.RLock()
	attemptDone := mw.starting.done
	resourceWatchers.lock.RUnlock()

	stopDone := make(chan struct{})
	go func() {
		enricherDeployment.Stop(resourceWatchers)
		close(stopDone)
	}()

	select {
	case <-stopDone:
	case <-time.After(5 * time.Second):
		t.Fatal("Stop() did not return within 5 seconds")
	}
	select {
	case <-startDone:
	case <-time.After(5 * time.Second):
		t.Fatal("Start() did not unblock after Stop() was called")
	}
	select {
	case <-attemptDone:
	case <-time.After(5 * time.Second):
		t.Fatal("launchStart goroutine did not finish within 5 seconds")
	}

	resourceWatchers.lock.Lock()
	assert.False(t, mw.started)
	resourceWatchers.lock.Unlock()
}

// TestBuildMetadataEnricher_StartStopDeadlock_ExtraWatcher verifies that
// Stop() can unblock a Start() that is blocked inside an extra watcher's
// Start() (same deadlock as the main watcher path).
func TestBuildMetadataEnricher_StartStopDeadlock_ExtraWatcher(t *testing.T) {
	resourceWatchers := NewWatchers()

	blockingNamespace := newBlockingMockWatcher()
	namespaceConfig, err := conf.NewConfigFrom(map[string]any{"enabled": true})
	require.NoError(t, err)

	config := &kubernetesConfig{
		Namespace:  "test-ns",
		SyncPeriod: time.Minute,
		Node:       "test-node",
		AddResourceMetadata: &metadata.AddResourceMetadataConfig{
			CronJob:    false,
			Deployment: false,
			Namespace:  namespaceConfig,
		},
	}
	log := logptest.NewTestingLogger(t, selector)

	deployMW := &metaWatcher{
		watcher:   newMockWatcher(),
		started:   false,
		users:     make(map[*enricher]watcherRegistration),
		enrichers: make(map[*enricher]struct{}),
	}
	nsMW := &metaWatcher{
		watcher:   blockingNamespace,
		started:   false,
		users:     make(map[*enricher]watcherRegistration),
		enrichers: make(map[*enricher]struct{}),
	}
	enricherDeployment := newMetadataEnricher("state_deployment", DeploymentResource, config, log)
	// The enricher owns both the main watcher and the namespace extra watcher.
	enricherDeployment.watchedResources = []string{NamespaceResource, DeploymentResource}
	deployMW.users[enricherDeployment] = watcherRegistration{committed: true, nodeScope: true}
	nsMW.users[enricherDeployment] = watcherRegistration{committed: true, nodeScope: true}

	resourceWatchers.lock.Lock()
	resourceWatchers.metaWatchersMap[DeploymentResource] = deployMW
	resourceWatchers.metaWatchersMap[NamespaceResource] = nsMW
	resourceWatchers.lock.Unlock()

	startDone := make(chan struct{})
	go func() {
		enricherDeployment.Start(resourceWatchers)
		close(startDone)
	}()

	<-blockingNamespace.startCalled

	// Capture the goroutine's done channel before calling Stop().
	resourceWatchers.lock.RLock()
	attemptDone := nsMW.starting.done
	resourceWatchers.lock.RUnlock()

	stopDone := make(chan struct{})
	go func() {
		enricherDeployment.Stop(resourceWatchers)
		close(stopDone)
	}()

	select {
	case <-stopDone:
	case <-time.After(5 * time.Second):
		t.Fatal("Stop() did not return within 5 seconds")
	}
	select {
	case <-startDone:
	case <-time.After(5 * time.Second):
		t.Fatal("Start() did not unblock after Stop() was called")
	}
	select {
	case <-attemptDone:
	case <-time.After(5 * time.Second):
		t.Fatal("launchStart goroutine did not finish within 5 seconds")
	}

	resourceWatchers.lock.Lock()
	assert.False(t, nsMW.started)
	resourceWatchers.lock.Unlock()
}

// TestExtraWatcherStopCancelsJoinedWait verifies the Stop/cancel lifecycle for a
// second enricher sharing a namespace extra watcher: Stop() cancels e2's joined
// wait on the in-flight goroutine and Start() returns false.
func TestExtraWatcherStopCancelsJoinedWait(t *testing.T) {
	resourceWatchers := NewWatchers()
	blockingNamespace := newBlockingMockWatcher()
	namespaceConfig, err := conf.NewConfigFrom(map[string]any{"enabled": true})
	require.NoError(t, err)
	config := &kubernetesConfig{
		AddResourceMetadata: &metadata.AddResourceMetadataConfig{
			Namespace: namespaceConfig,
		},
	}
	log := logptest.NewTestingLogger(t, selector)

	podMW := &metaWatcher{
		watcher:   newMockWatcher(),
		started:   true, // already started; primary does not block
		users:     make(map[*enricher]watcherRegistration),
		enrichers: make(map[*enricher]struct{}),
	}
	nsMW := &metaWatcher{
		watcher:   blockingNamespace,
		users:     make(map[*enricher]watcherRegistration),
		enrichers: make(map[*enricher]struct{}),
	}

	e1 := newMetadataEnricher("pod", PodResource, config, log)
	e1.watchedResources = []string{NamespaceResource, PodResource}
	podMW.users[e1] = watcherRegistration{committed: true}
	podMW.enrichers[e1] = struct{}{}
	nsMW.users[e1] = watcherRegistration{committed: true}

	e2 := newMetadataEnricher("state_pod", PodResource, config, log)
	e2.watchedResources = []string{NamespaceResource, PodResource}
	podMW.users[e2] = watcherRegistration{committed: true}
	podMW.enrichers[e2] = struct{}{}
	nsMW.users[e2] = watcherRegistration{committed: true}

	resourceWatchers.lock.Lock()
	resourceWatchers.metaWatchersMap[PodResource] = podMW
	resourceWatchers.metaWatchersMap[NamespaceResource] = nsMW
	resourceWatchers.lock.Unlock()

	// e1 starts first; its goroutine calls blockingNamespace.Start() and blocks.
	e1StartDone := make(chan struct{})
	go func() {
		e1.Start(resourceWatchers)
		close(e1StartDone)
	}()
	<-blockingNamespace.startCalled

	// Capture the goroutine's done channel before either Stop() call.
	resourceWatchers.lock.RLock()
	attemptDone := nsMW.starting.done
	resourceWatchers.lock.RUnlock()

	// e2 also starts; Stop() must cancel its joined wait and Start() return false.
	e2StartDone := make(chan bool, 1)
	go func() { e2StartDone <- e2.Start(resourceWatchers) }()

	// Stop e2: its Start() must unblock with false.
	e2StopDone := make(chan struct{})
	go func() {
		e2.Stop(resourceWatchers)
		close(e2StopDone)
	}()
	select {
	case <-e2StopDone:
	case <-time.After(5 * time.Second):
		t.Fatal("e2.Stop() did not return within 5 seconds")
	}
	select {
	case result := <-e2StartDone:
		assert.False(t, result, "e2.Start() must return false after Stop()")
	case <-time.After(5 * time.Second):
		t.Fatal("e2.Start() did not return after e2.Stop()")
	}

	// Cleanup: stop e1 then wait for the goroutine.
	e1.Stop(resourceWatchers)
	select {
	case <-e1StartDone:
	case <-time.After(5 * time.Second):
		t.Fatal("e1.Start() did not return after e1.Stop()")
	}
	select {
	case <-attemptDone:
	case <-time.After(5 * time.Second):
		t.Fatal("launchStart goroutine did not finish within 5 seconds")
	}
}

// TestStartReturnsWhenExtrasReady verifies the positive path: Start() returns true
// after extras (node, namespace) and the primary (pod) watcher all start successfully.
// This is the path taken on every Fetch by pod/container metricsets with default config.
func TestStartReturnsWhenExtrasReady(t *testing.T) {
	resourceWatchers := NewWatchers()

	nodeConfig, err := conf.NewConfigFrom(map[string]any{"enabled": true})
	require.NoError(t, err)
	namespaceConfig, err := conf.NewConfigFrom(map[string]any{"enabled": true})
	require.NoError(t, err)
	config := &kubernetesConfig{
		AddResourceMetadata: &metadata.AddResourceMetadataConfig{
			Node:      nodeConfig,
			Namespace: namespaceConfig,
		},
	}
	log := logptest.NewTestingLogger(t, selector)

	nodeMock := newMockWatcher()
	nsMock := newMockWatcher()
	podMock := newMockWatcher()

	nodeMW := &metaWatcher{
		watcher:   nodeMock,
		users:     make(map[*enricher]watcherRegistration),
		enrichers: make(map[*enricher]struct{}),
	}
	nsMW := &metaWatcher{
		watcher:   nsMock,
		users:     make(map[*enricher]watcherRegistration),
		enrichers: make(map[*enricher]struct{}),
	}
	podMW := &metaWatcher{
		watcher:   podMock,
		users:     make(map[*enricher]watcherRegistration),
		enrichers: make(map[*enricher]struct{}),
	}

	e := newMetadataEnricher("pod", PodResource, config, log)
	e.watchedResources = []string{NodeResource, NamespaceResource, PodResource}
	nodeMW.users[e] = watcherRegistration{committed: true}
	nsMW.users[e] = watcherRegistration{committed: true}
	podMW.users[e] = watcherRegistration{committed: true}

	resourceWatchers.lock.Lock()
	resourceWatchers.metaWatchersMap[NodeResource] = nodeMW
	resourceWatchers.metaWatchersMap[NamespaceResource] = nsMW
	resourceWatchers.metaWatchersMap[PodResource] = podMW
	resourceWatchers.lock.Unlock()

	assert.True(t, e.Start(resourceWatchers), "Start() must return true when all watchers commit successfully")

	resourceWatchers.lock.RLock()
	assert.True(t, nodeMW.started, "node extra watcher must be started before Start() returns")
	assert.True(t, nsMW.started, "namespace extra watcher must be started before Start() returns")
	assert.True(t, podMW.started, "primary watcher must be started before Start() returns")
	resourceWatchers.lock.RUnlock()

	assert.Equal(t, 1, nodeMock.startCalls, "node watcher must be started exactly once")
	assert.Equal(t, 1, nsMock.startCalls, "namespace watcher must be started exactly once")
	assert.Equal(t, 1, podMock.startCalls, "primary watcher must be started exactly once")

	// Second Start(): all watchers already started — must hit the fast path and
	// return true without calling watcher.Start() again.
	assert.True(t, e.Start(resourceWatchers), "second Start() must return true via fast path")
	assert.Equal(t, 1, nodeMock.startCalls, "fast path must not call node watcher.Start() again")
	assert.Equal(t, 1, nsMock.startCalls, "fast path must not call namespace watcher.Start() again")
	assert.Equal(t, 1, podMock.startCalls, "fast path must not call primary watcher.Start() again")
}

// TestHang1NonFinalOwnerStopCancelsStart verifies the hang-1 fix: a non-final
// owner's Stop() cancels its own blocked Start() via stopCh without stopping the
// shared watcher, so the remaining owner's eventual Stop() can clean up normally.
func TestHang1NonFinalOwnerStopCancelsStart(t *testing.T) {
	resourceWatchers := NewWatchers()
	blocking := newBlockingMockWatcher()
	config := &kubernetesConfig{
		AddResourceMetadata: &metadata.AddResourceMetadataConfig{}, // no extras
	}
	log := logptest.NewTestingLogger(t, selector)

	mw := &metaWatcher{
		watcher:   blocking,
		users:     make(map[*enricher]watcherRegistration),
		enrichers: make(map[*enricher]struct{}),
	}

	e1 := newMetadataEnricher("pod", PodResource, config, log)
	e1.watchedResources = []string{PodResource}
	mw.users[e1] = watcherRegistration{committed: true}
	mw.enrichers[e1] = struct{}{}

	e2 := newMetadataEnricher("state_pod", PodResource, config, log)
	e2.watchedResources = []string{PodResource}
	mw.users[e2] = watcherRegistration{committed: true}
	mw.enrichers[e2] = struct{}{}

	resourceWatchers.lock.Lock()
	resourceWatchers.metaWatchersMap[PodResource] = mw
	resourceWatchers.lock.Unlock()

	// e1 starts; its goroutine blocks inside blocking.Start().
	e1StartDone := make(chan bool, 1)
	go func() {
		e1StartDone <- e1.Start(resourceWatchers)
	}()
	<-blocking.startCalled

	// Capture the goroutine's done channel before calling Stop().
	resourceWatchers.lock.RLock()
	attemptDone := mw.starting.done
	resourceWatchers.lock.RUnlock()

	// e1 is a non-final owner (e2 is still registered). Stop() must cancel e1's
	// wait (via stopCh) without stopping the shared watcher.
	e1StopDone := make(chan struct{})
	go func() {
		e1.Stop(resourceWatchers)
		close(e1StopDone)
	}()
	select {
	case <-e1StopDone:
	case <-time.After(5 * time.Second):
		t.Fatal("e1.Stop() did not return within 5 seconds")
	}
	result := <-e1StartDone
	assert.False(t, result, "e1.Start() must return false after e1.Stop()")

	// The watcher must still be registered for e2.
	resourceWatchers.lock.RLock()
	assert.NotNil(t, resourceWatchers.metaWatchersMap[PodResource], "watcher must remain in registry for the surviving owner e2")
	resourceWatchers.lock.RUnlock()

	// e2 is the final owner: its Stop() must stop the shared watcher, which
	// unblocks the launchStart goroutine.
	e2.Stop(resourceWatchers)
	select {
	case <-attemptDone:
	case <-time.After(5 * time.Second):
		t.Fatal("launchStart goroutine did not finish within 5 seconds after e2.Stop()")
	}
}

// TestMissingRequiredExtraWatcherBlocksStart verifies that collectExtraDependencies returns
// false when getExtraWatchers() requires an extra that is absent from the registry,
// causing Start() to return false rather than proceeding with incomplete enrichment.
func TestMissingRequiredExtraWatcherBlocksStart(t *testing.T) {
	namespaceConfig, err := conf.NewConfigFrom(map[string]any{"enabled": true})
	require.NoError(t, err)
	config := &kubernetesConfig{
		AddResourceMetadata: &metadata.AddResourceMetadataConfig{
			Namespace: namespaceConfig,
		},
	}
	log := logptest.NewTestingLogger(t, selector)
	resourceWatchers := NewWatchers()

	podMW := &metaWatcher{
		watcher:   newMockWatcher(),
		started:   true,
		users:     make(map[*enricher]watcherRegistration),
		enrichers: make(map[*enricher]struct{}),
	}
	e := newMetadataEnricher("pod", PodResource, config, log)
	e.watchedResources = []string{PodResource}
	podMW.users[e] = watcherRegistration{committed: true}
	podMW.enrichers[e] = struct{}{}

	resourceWatchers.lock.Lock()
	resourceWatchers.metaWatchersMap[PodResource] = podMW
	// NamespaceResource intentionally not registered (failed construction).
	resourceWatchers.lock.Unlock()

	result := e.Start(resourceWatchers)
	assert.False(t, result, "Start() must return false when a required extra watcher is missing from the registry")

	e.Stop(resourceWatchers)
}

// TestWaitForAttemptCancellation directly tests waitForAttempt: closing stopCh returns
// false whether done is closed concurrently or not, and closing done alone returns true.
func TestWaitForAttemptCancellation(t *testing.T) {
	t.Run("stopCh wins over open done", func(t *testing.T) {
		attempt := &watcherStart{done: make(chan struct{})}
		stopCh := make(chan struct{})
		close(stopCh)
		assert.False(t, waitForAttempt(attempt, stopCh), "closed stopCh must return false")
	})

	t.Run("done wins when stopCh is open", func(t *testing.T) {
		attempt := &watcherStart{done: make(chan struct{})}
		stopCh := make(chan struct{})
		close(attempt.done)
		assert.True(t, waitForAttempt(attempt, stopCh), "closed done must return true")
	})

	t.Run("both closed returns false via stopCh re-check", func(t *testing.T) {
		attempt := &watcherStart{done: make(chan struct{})}
		stopCh := make(chan struct{})
		close(attempt.done)
		close(stopCh)
		// Outer select picks either branch; inner re-check on stopCh forces false.
		for range 200 {
			assert.False(t, waitForAttempt(attempt, stopCh), "both closed must return false")
		}
	})
}

// TestClaimStartJoinReturnsSameAttempt verifies that a second call to claimStart
// while a start is already in progress returns the existing attempt, not a new one.
// This is the unit-level proof of the join-on-concurrent-start invariant; it does
// not rely on goroutine scheduling or timing.
func TestClaimStartJoinReturnsSameAttempt(t *testing.T) {
	mw := &metaWatcher{
		watcher:   newMockWatcher(),
		users:     make(map[*enricher]watcherRegistration),
		enrichers: make(map[*enricher]struct{}),
	}

	first, claimed := claimStart(mw)
	require.True(t, claimed, "first claimStart must claim the attempt")
	require.NotNil(t, first)

	second, claimed := claimStart(mw)
	assert.False(t, claimed, "second claimStart must join, not claim")
	assert.Same(t, first, second, "second caller must receive the same attempt pointer")
}

// TestClaimReplacementJoinReturnsSameAttempt verifies that a second call to claimReplacement
// while a restart is already pending returns the existing attempt, not a new one.
// This is the unit-level proof of the join-on-scope-replacement invariant.
func TestClaimReplacementJoinReturnsSameAttempt(t *testing.T) {
	rw := NewWatchers()
	r := newReleasableWatcher()
	log := logptest.NewTestingLogger(t, selector)

	mw := &metaWatcher{
		watcher:            newMockWatcher(),
		started:            true,
		nodeScope:          true,
		replacementWatcher: r,
		users:              make(map[*enricher]watcherRegistration),
		enrichers:          make(map[*enricher]struct{}),
	}
	rw.metaWatchersMap[PodResource] = mw

	rw.lock.Lock()
	first, ok := claimReplacement(rw, log, PodResource, mw)
	require.True(t, ok, "first claimReplacement must succeed")
	require.NotNil(t, first)

	second, ok := claimReplacement(rw, log, PodResource, mw)
	rw.lock.Unlock()

	require.True(t, ok, "second claimReplacement must join the pending attempt")
	assert.Same(t, first, second, "second caller must receive the same attempt pointer")

	r.Release()
	<-first.done
}

// TestStartErrTerminalGuard verifies that a watcher failure records a terminal
// startErr and prevents any further watcher.Start() calls: the watcher failed
// once, so the metricset must not keep retrying on every Fetch.
func TestStartErrTerminalGuard(t *testing.T) {
	config := &kubernetesConfig{AddResourceMetadata: &metadata.AddResourceMetadataConfig{}}
	log := logptest.NewTestingLogger(t, selector)

	rw := NewWatchers()
	failingMock := newMockWatcher()
	failingMock.startErr = fmt.Errorf("watcher failed")
	mw := &metaWatcher{
		watcher:   failingMock,
		users:     make(map[*enricher]watcherRegistration),
		enrichers: make(map[*enricher]struct{}),
	}
	e := newMetadataEnricher("pod", PodResource, config, log)
	e.watchedResources = []string{PodResource}
	mw.users[e] = watcherRegistration{committed: true}
	mw.enrichers[e] = struct{}{}
	rw.metaWatchersMap[PodResource] = mw

	assert.False(t, e.Start(rw), "Start() must return false when the watcher fails to start")
	assert.Equal(t, 1, failingMock.startCalls, "watcher.Start() must be called exactly once")

	rw.lock.RLock()
	assert.Nil(t, mw.starting, "commitStart must clear starting after the attempt")
	assert.Error(t, mw.startErr, "commitStart must record the terminal error")
	rw.lock.RUnlock()

	assert.False(t, e.Start(rw), "second Start() must return false without retrying")
	assert.Equal(t, 1, failingMock.startCalls, "terminal startErr must prevent further watcher.Start() calls")
}

// TestStartInitialGuardAndIdempotency verifies the initial stopped() check in Start()
// and that repeated calls after Stop() consistently return false.
func TestStartInitialGuardAndIdempotency(t *testing.T) {
	resourceWatchers := NewWatchers()
	config := &kubernetesConfig{AddResourceMetadata: &metadata.AddResourceMetadataConfig{}}
	log := logptest.NewTestingLogger(t, selector)

	mw := &metaWatcher{
		watcher:   newMockWatcher(),
		started:   true,
		users:     make(map[*enricher]watcherRegistration),
		enrichers: make(map[*enricher]struct{}),
	}
	resourceWatchers.metaWatchersMap[PodResource] = mw

	e := newMetadataEnricher("pod", PodResource, config, log)
	e.watchedResources = []string{PodResource}
	mw.users[e] = watcherRegistration{committed: true}
	mw.enrichers[e] = struct{}{}

	// Stop() before Start() must always produce false.
	e.Stop(resourceWatchers)
	assert.False(t, e.Start(resourceWatchers), "Start() after Stop() must return false")

	// Second call: idempotent.
	assert.False(t, e.Start(resourceWatchers), "repeated Start() after Stop() must return false")
}

// TestHang2NonFinalClusterScopedOwnerStopCancelsReplacement verifies the hang-2 fix:
// a non-final cluster-scoped owner's Stop() must cancel its own wait for the
// scope-replacement goroutine (via stopCh) without stopping R or disturbing
// the surviving cluster-scoped owner. EC2 returns true once R succeeds.
func TestHang2NonFinalClusterScopedOwnerStopCancelsReplacement(t *testing.T) {
	resourceWatchers := NewWatchers()
	w := newMockWatcher()
	r := newReleasableWatcher()
	config := &kubernetesConfig{AddResourceMetadata: &metadata.AddResourceMetadataConfig{}}
	log := logptest.NewTestingLogger(t, selector)

	mw := &metaWatcher{
		watcher:            w,
		started:            true,
		nodeScope:          true,
		replacementWatcher: r,
		users:              make(map[*enricher]watcherRegistration),
		enrichers:          make(map[*enricher]struct{}),
	}
	resourceWatchers.metaWatchersMap[PodResource] = mw

	ec1 := newMetadataEnricher("state_pod", PodResource, config, log)
	ec1.watchedResources = []string{PodResource}
	mw.users[ec1] = watcherRegistration{committed: true, nodeScope: false}
	mw.enrichers[ec1] = struct{}{}

	ec2 := newMetadataEnricher("state_container", PodResource, config, log)
	ec2.watchedResources = []string{PodResource}
	mw.users[ec2] = watcherRegistration{committed: true, nodeScope: false}
	mw.enrichers[ec2] = struct{}{}

	// EC1 starts: claimReplacement launches R's goroutine; EC1 blocks on attempt.done.
	ec1StartDone := make(chan bool, 1)
	go func() { ec1StartDone <- ec1.Start(resourceWatchers) }()
	<-r.startCalled

	// EC1 is non-final: Stop() must cancel EC1's wait without stopping R.
	ec1StopDone := make(chan struct{})
	go func() {
		ec1.Stop(resourceWatchers)
		close(ec1StopDone)
	}()
	select {
	case <-ec1StopDone:
	case <-time.After(5 * time.Second):
		t.Fatal("ec1.Stop() did not return within 5 seconds; must not block waiting for R")
	}
	assert.False(t, <-ec1StartDone, "ec1.Start() must return false after ec1.Stop()")

	// R must still be in flight: pendingReplacement preserved for the surviving EC2.
	resourceWatchers.lock.RLock()
	assert.NotNil(t, mw.pendingReplacement, "pendingReplacement must remain: R is still starting for EC2")
	resourceWatchers.lock.RUnlock()

	// EC2 starts: joins the in-flight attempt; then R succeeds.
	ec2StartDone := make(chan bool, 1)
	go func() { ec2StartDone <- ec2.Start(resourceWatchers) }()

	r.Release()

	select {
	case result := <-ec2StartDone:
		assert.True(t, result, "ec2.Start() must return true after R succeeds")
	case <-time.After(5 * time.Second):
		t.Fatal("ec2.Start() did not return within 5 seconds after R succeeded")
	}
	assert.EqualValues(t, 1, r.startCount.Load(), "R.Start() must be called exactly once")
}

// TestScopeReplacementSecondClusterOwnerJoinsExistingAttempt verifies that two
// concurrent cluster-scoped owners both waiting for scope replacement join a
// single shared goroutine: R.Start() is called exactly once.
func TestScopeReplacementSecondClusterOwnerJoinsExistingAttempt(t *testing.T) {
	resourceWatchers := NewWatchers()
	w := newMockWatcher()
	r := newReleasableWatcher()
	config := &kubernetesConfig{AddResourceMetadata: &metadata.AddResourceMetadataConfig{}}
	log := logptest.NewTestingLogger(t, selector)

	mw := &metaWatcher{
		watcher:            w,
		started:            true,
		nodeScope:          true,
		replacementWatcher: r,
		users:              make(map[*enricher]watcherRegistration),
		enrichers:          make(map[*enricher]struct{}),
	}
	resourceWatchers.metaWatchersMap[PodResource] = mw

	ec1 := newMetadataEnricher("state_pod", PodResource, config, log)
	ec1.watchedResources = []string{PodResource}
	mw.users[ec1] = watcherRegistration{committed: true, nodeScope: false}
	mw.enrichers[ec1] = struct{}{}

	ec2 := newMetadataEnricher("state_container", PodResource, config, log)
	ec2.watchedResources = []string{PodResource}
	mw.users[ec2] = watcherRegistration{committed: true, nodeScope: false}
	mw.enrichers[ec2] = struct{}{}

	ec1StartDone := make(chan bool, 1)
	ec2StartDone := make(chan bool, 1)
	go func() { ec1StartDone <- ec1.Start(resourceWatchers) }()
	<-r.startCalled // R is in Start(); EC1 is blocking

	// EC2 also starts; both must return true after R succeeds, R.Start() called once.
	go func() { ec2StartDone <- ec2.Start(resourceWatchers) }()

	r.Release()

	select {
	case result := <-ec1StartDone:
		assert.True(t, result, "ec1.Start() must return true after R succeeds")
	case <-time.After(5 * time.Second):
		t.Fatal("ec1.Start() did not return within 5 seconds")
	}
	select {
	case result := <-ec2StartDone:
		assert.True(t, result, "ec2.Start() must return true after R succeeds")
	case <-time.After(5 * time.Second):
		t.Fatal("ec2.Start() did not return within 5 seconds")
	}
	assert.EqualValues(t, 1, r.startCount.Load(), "R.Start() must have been called exactly once total")
}

// TestScopeReplacementSuccessSwapsWatcher verifies that when R succeeds,
// the active watcher is swapped from W to R, W is stopped, and both cluster-scoped
// owners return true.
func TestScopeReplacementSuccessSwapsWatcher(t *testing.T) {
	resourceWatchers := NewWatchers()
	w := newBlockingMockWatcher()
	r := newReleasableWatcher()
	config := &kubernetesConfig{AddResourceMetadata: &metadata.AddResourceMetadataConfig{}}
	log := logptest.NewTestingLogger(t, selector)

	mw := &metaWatcher{
		watcher:            w,
		started:            true,
		nodeScope:          true,
		replacementWatcher: r,
		users:              make(map[*enricher]watcherRegistration),
		enrichers:          make(map[*enricher]struct{}),
	}
	resourceWatchers.metaWatchersMap[PodResource] = mw

	ec := newMetadataEnricher("state_pod", PodResource, config, log)
	ec.watchedResources = []string{PodResource}
	mw.users[ec] = watcherRegistration{committed: true, nodeScope: false}
	mw.enrichers[ec] = struct{}{}

	ecStartDone := make(chan bool, 1)
	go func() { ecStartDone <- ec.Start(resourceWatchers) }()
	<-r.startCalled

	r.Release()

	select {
	case result := <-ecStartDone:
		assert.True(t, result, "ec.Start() must return true after R succeeds")
	case <-time.After(5 * time.Second):
		t.Fatal("ec.Start() did not return within 5 seconds")
	}

	resourceWatchers.lock.RLock()
	assert.False(t, mw.nodeScope, "nodeScope must be false after R commits")
	assert.True(t, mw.started, "started must be true after R commits")
	assert.Nil(t, mw.pendingReplacement, "pendingReplacement must be nil after R commits")
	resourceWatchers.lock.RUnlock()

	// W must have been stopped by the replacement commit.
	select {
	case <-w.stopCalled:
	case <-time.After(5 * time.Second):
		t.Fatal("W.Stop() was not called after R committed")
	}

	ec.Stop(resourceWatchers)
}

// TestScopeReplacementFailureRestoresFactory verifies that when R fails,
// W remains active, the factory is restored for retry, and the caller returns false.
func TestScopeReplacementFailureRestoresFactory(t *testing.T) {
	resourceWatchers := NewWatchers()
	w := newMockWatcher()
	r := newReleasableWatcher()
	factoryCalled := 0
	factory := func() (kubernetes.Watcher, error) {
		factoryCalled++
		return r, nil
	}
	config := &kubernetesConfig{AddResourceMetadata: &metadata.AddResourceMetadataConfig{}}
	log := logptest.NewTestingLogger(t, selector)

	mw := &metaWatcher{
		watcher:                   w,
		started:                   true,
		nodeScope:                 true,
		replacementWatcherFactory: factory,
		users:                     make(map[*enricher]watcherRegistration),
		enrichers:                 make(map[*enricher]struct{}),
	}
	resourceWatchers.metaWatchersMap[PodResource] = mw

	ec := newMetadataEnricher("state_pod", PodResource, config, log)
	ec.watchedResources = []string{PodResource}
	mw.users[ec] = watcherRegistration{committed: true, nodeScope: false}
	mw.enrichers[ec] = struct{}{}

	ecStartDone := make(chan bool, 1)
	go func() { ecStartDone <- ec.Start(resourceWatchers) }()
	<-r.startCalled

	// R fails: stop it.
	r.Stop()

	select {
	case result := <-ecStartDone:
		assert.False(t, result, "ec.Start() must return false when R fails")
	case <-time.After(5 * time.Second):
		t.Fatal("ec.Start() did not return within 5 seconds after R failed")
	}

	resourceWatchers.lock.RLock()
	assert.True(t, mw.nodeScope, "nodeScope must stay true; W was never swapped out")
	assert.True(t, mw.started, "W must still be started")
	assert.Nil(t, mw.pendingReplacement, "pendingReplacement must be nil after R failed")
	assert.NotNil(t, mw.replacementWatcherFactory, "factory must be restored so the next Start() can retry")
	resourceWatchers.lock.RUnlock()
	assert.Equal(t, 1, factoryCalled, "factory must have been called exactly once to create R")

	ec.Stop(resourceWatchers)
}

// TestNodeScopedOwnerReturnsTrueWhenReplacementCommits pins the commit-before-stop
// ordering in claimReplacement: R must set m.started=true under the registry lock before
// calling W.Stop(), so that EN's waitAndVerify (triggered when W's goroutine closes
// W.attempt.done) observes the committed started state and returns true.
func TestNodeScopedOwnerReturnsTrueWhenReplacementCommits(t *testing.T) {
	resourceWatchers := NewWatchers()
	w := newBlockingMockWatcher() // W: node-scoped, not yet started
	r := newReleasableWatcher()   // R: cluster-scoped replacement
	config := &kubernetesConfig{AddResourceMetadata: &metadata.AddResourceMetadataConfig{}}
	log := logptest.NewTestingLogger(t, selector)

	mw := &metaWatcher{
		watcher:            w,
		started:            false,
		nodeScope:          true,
		replacementWatcher: r,
		users:              make(map[*enricher]watcherRegistration),
		enrichers:          make(map[*enricher]struct{}),
	}
	resourceWatchers.metaWatchersMap[PodResource] = mw

	// EN: node-scoped owner. Its Start() waits on W's startup goroutine.
	en := newMetadataEnricher("pod", PodResource, config, log)
	en.watchedResources = []string{PodResource}
	mw.users[en] = watcherRegistration{committed: true, nodeScope: true}
	mw.enrichers[en] = struct{}{}

	// EC: cluster-scoped owner. Its Start() triggers claimReplacement and waits for R.
	ec := newMetadataEnricher("state_pod", PodResource, config, log)
	ec.watchedResources = []string{PodResource}
	mw.users[ec] = watcherRegistration{committed: true, nodeScope: false}
	mw.enrichers[ec] = struct{}{}

	// EN starts; its goroutine blocks in W.Start() (not started yet, so ensureWatcherReady runs).
	enStartDone := make(chan bool, 1)
	go func() { enStartDone <- en.Start(resourceWatchers) }()
	<-w.startCalled

	// EC starts; claimReplacement launches R's goroutine; EC blocks on R's attempt.done.
	ecStartDone := make(chan bool, 1)
	go func() { ecStartDone <- ec.Start(resourceWatchers) }()
	<-r.startCalled

	// Release R: R commits m.started=true, m.starting=nil under the lock, then
	// stops W outside the lock. W.Start() returns error, W's goroutine closes
	// W.attempt.done. EN's waitAndVerify re-reads m.started=true and returns true.
	r.Release()

	select {
	case result := <-enStartDone:
		assert.True(t, result, "EN must return true: R committed m.started=true before W's done closed")
	case <-time.After(5 * time.Second):
		t.Fatal("EN.Start() did not return within 5 seconds after R committed")
	}
	select {
	case result := <-ecStartDone:
		assert.True(t, result, "EC must return true after R succeeds")
	case <-time.After(5 * time.Second):
		t.Fatal("EC.Start() did not return within 5 seconds after R committed")
	}

	en.Stop(resourceWatchers)
	ec.Stop(resourceWatchers)
}

func requireActiveWatcherWriterPending(t *testing.T, metaWatcher *metaWatcher, message string) {
	t.Helper()

	require.Eventually(t, func() bool {
		if metaWatcher.activeWatcherLock.TryRLock() {
			metaWatcher.activeWatcherLock.RUnlock()
			return false
		}
		return true
	}, time.Second, time.Millisecond, message)
}

// blockingMockWatcher blocks in Start() until Stop() is called.
type blockingMockWatcher struct {
	startCalled chan struct{}
	stopCalled  chan struct{}
	store       cache.Store
}

func newBlockingMockWatcher() *blockingMockWatcher {
	return &blockingMockWatcher{
		startCalled: make(chan struct{}),
		stopCalled:  make(chan struct{}),
		store: cache.NewStore(func(obj any) (string, error) {
			objName, err := cache.ObjectToName(obj)
			if err != nil {
				return "", err
			}
			return objName.Name, nil
		}),
	}
}

func (m *blockingMockWatcher) Start() error {
	select {
	case <-m.startCalled:
	default:
		close(m.startCalled)
	}
	<-m.stopCalled
	return fmt.Errorf("watcher stopped before cache sync")
}

func (m *blockingMockWatcher) Stop() {
	// Close only once; a second Stop() call is a no-op.
	select {
	case <-m.stopCalled:
	default:
		close(m.stopCalled)
	}
}

func (m *blockingMockWatcher) GetEventHandler() kubernetes.ResourceEventHandler  { return nil }
func (m *blockingMockWatcher) AddEventHandler(r kubernetes.ResourceEventHandler) {}
func (m *blockingMockWatcher) Store() cache.Store                                { return m.store }
func (m *blockingMockWatcher) Client() k8s.Interface                             { return nil }
func (m *blockingMockWatcher) CachedObject() runtime.Object                      { return nil }

func TestGetWatcherStoreKeyFromMetadataKey(t *testing.T) {
	t.Run("global resource", func(t *testing.T) {
		assert.Equal(t, "name", getWatcherStoreKeyFromMetadataKey("name"))
	})
	t.Run("namespaced resource", func(t *testing.T) {
		assert.Equal(t, "namespace/name", getWatcherStoreKeyFromMetadataKey("namespace/name"))
	})
	t.Run("container", func(t *testing.T) {
		assert.Equal(t, "namespace/pod", getWatcherStoreKeyFromMetadataKey("namespace/pod/container"))
	})
}

func resourceMetadataConfig(t *testing.T, node, namespace, deployment, cronJob bool) *metadata.AddResourceMetadataConfig {
	t.Helper()
	nodeConfig, err := conf.NewConfigFrom(map[string]any{"enabled": node})
	require.NoError(t, err, "node metadata config must be valid")
	namespaceConfig, err := conf.NewConfigFrom(map[string]any{"enabled": namespace})
	require.NoError(t, err, "namespace metadata config must be valid")
	return &metadata.AddResourceMetadataConfig{
		Node:       nodeConfig,
		Namespace:  namespaceConfig,
		Deployment: deployment,
		CronJob:    cronJob,
	}
}

func informerTestPod(name, generation string) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
			UID:       types.UID(name),
			Labels:    map[string]string{"generation": generation},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{{Name: informerTestContainerName}},
		},
	}
}

func configureRealInformerTestEnricher(e *enricher, container bool) {
	e.Lock()
	defer e.Unlock()

	e.updateFunc = func(resource kubernetes.Resource) map[string]mapstr.M {
		pod := resource.(*kubernetes.Pod) //nolint:errcheck // It's a test, let it panic
		eventMetadata := func() mapstr.M {
			return mapstr.M{
				"kubernetes": mapstr.M{
					"labels": mapstr.M{"generation": pod.Labels["generation"]},
					"pod": mapstr.M{
						"name": pod.Name,
						"uid":  string(pod.UID),
					},
				},
			}
		}

		if container {
			result := make(map[string]mapstr.M, len(pod.Spec.Containers))
			for _, podContainer := range pod.Spec.Containers {
				result[join(pod.Namespace, pod.Name, podContainer.Name)] = eventMetadata()
			}
			return result
		}
		return map[string]mapstr.M{join(pod.Namespace, pod.Name): eventMetadata()}
	}
	e.deleteFunc = func(resource kubernetes.Resource) []string {
		pod := resource.(*kubernetes.Pod) //nolint:errcheck // It's a test, let it panic
		if container {
			ids := make([]string, 0, len(pod.Spec.Containers))
			for _, podContainer := range pod.Spec.Containers {
				ids = append(ids, join(pod.Namespace, pod.Name, podContainer.Name))
			}
			return ids
		}
		return []string{join(pod.Namespace, pod.Name)}
	}
	if container {
		e.index = func(event mapstr.M) string {
			return join(
				getString(event, mb.ModuleDataKey+".namespace"),
				getString(event, mb.ModuleDataKey+".pod.name"),
				getString(event, "name"),
			)
		}
	} else {
		e.index = func(event mapstr.M) string {
			return join(getString(event, mb.ModuleDataKey+".namespace"), getString(event, "name"))
		}
		e.isPod = true
	}
}

func newFailingStateServiceEnricher(
	t *testing.T,
	metricsRepo *MetricsRepo,
	resourceWatchers *Watchers,
	nodeScope bool,
) Enricher {
	t.Helper()

	const moduleName = "constructor_rollback_test"
	registry := mb.NewRegister()
	require.NoError(t, registry.AddModule(moduleName, mb.DefaultModuleFactory))

	var enricher Enricher
	registry.MustAddMetricSet(moduleName, "state_service", func(base mb.BaseMetricSet) (mb.MetricSet, error) {
		enricher = NewResourceMetadataEnricher(base, metricsRepo, resourceWatchers, nodeScope)
		return &constructorRollbackMetricSet{BaseMetricSet: base}, nil
	})

	mbtest.NewMetricSetWithRegistry(t, map[string]any{
		"module":       moduleName,
		"metricsets":   []string{"state_service"},
		"add_metadata": true,
		"kube_config":  writeTestKubeConfig(t),
		"add_resource_metadata": map[string]any{
			"namespace": map[string]any{"enabled": false},
		},
	}, registry)
	return enricher
}

func writeTestKubeConfig(t *testing.T) string {
	path := filepath.Join(t.TempDir(), "kubeconfig")
	require.NoError(t, os.WriteFile(path, []byte(`
apiVersion: v1
kind: Config
clusters:
  - name: test
    cluster:
      server: https://127.0.0.1:1
      insecure-skip-tls-verify: true
contexts:
  - name: test
    context:
      cluster: test
      user: test
current-context: test
users:
  - name: test
    user:
      token: test
`), 0o600))
	return path
}

func buildTestMetadataEnricher(
	metricsetName string,
	resourceName string,
	resourceWatchers *Watchers,
	config *kubernetesConfig,
	funcs *mockFuncs,
	log *logp.Logger,
) *enricher {
	return buildTestMetadataEnricherWithScope(
		metricsetName,
		resourceName,
		resourceWatchers,
		config,
		funcs,
		log,
		false,
	)
}

func buildTestMetadataEnricherWithScope(
	metricsetName string,
	resourceName string,
	resourceWatchers *Watchers,
	config *kubernetesConfig,
	funcs *mockFuncs,
	log *logp.Logger,
	nodeScope bool,
) *enricher {
	return buildTestMetadataEnricherWithFuncsAndScope(
		metricsetName,
		resourceName,
		resourceWatchers,
		config,
		funcs.update,
		funcs.delete,
		funcs.index,
		log,
		nodeScope,
	)
}

func buildTestMetadataEnricherWithFuncs(
	metricsetName string,
	resourceName string,
	resourceWatchers *Watchers,
	config *kubernetesConfig,
	updateFunc func(kubernetes.Resource) map[string]mapstr.M,
	deleteFunc func(kubernetes.Resource) []string,
	indexFunc func(mapstr.M) string,
	log *logp.Logger,
) *enricher {
	return buildTestMetadataEnricherWithFuncsAndScope(
		metricsetName,
		resourceName,
		resourceWatchers,
		config,
		updateFunc,
		deleteFunc,
		indexFunc,
		log,
		false,
	)
}

func buildTestMetadataEnricherWithFuncsAndScope(
	metricsetName string,
	resourceName string,
	resourceWatchers *Watchers,
	config *kubernetesConfig,
	updateFunc func(kubernetes.Resource) map[string]mapstr.M,
	deleteFunc func(kubernetes.Resource) []string,
	indexFunc func(mapstr.M) string,
	log *logp.Logger,
	nodeScope bool,
) *enricher {
	e := newMetadataEnricher(metricsetName, resourceName, config, log)
	e.updateFunc = updateFunc
	e.deleteFunc = deleteFunc
	e.index = indexFunc

	resourceWatchers.lock.Lock()
	metaWatcher := resourceWatchers.metaWatchersMap[resourceName]
	registerWatcherUser(resourceName, metaWatcher, e, true, nodeScope)
	resourceWatchers.lock.Unlock()
	commitWatcherOwnership(e, resourceWatchers)
	return e
}

func newWatcherLookupTestEnricher(t *testing.T, resourceWatchers *Watchers, metaWatcher *metaWatcher) *enricher {
	t.Helper()

	e := newMetadataEnricher(
		"pod",
		PodResource,
		&kubernetesConfig{AddResourceMetadata: &metadata.AddResourceMetadataConfig{}},
		logptest.NewTestingLogger(t, selector),
	)
	e.index = func(event mapstr.M) string {
		return event["name"].(string) //nolint:errcheck // test event always has a name
	}
	e.updateFunc = func(resource kubernetes.Resource) map[string]mapstr.M {
		pod := resource.(*kubernetes.Pod) //nolint:errcheck // test watcher stores Pods
		return map[string]mapstr.M{
			pod.Name: {"source": pod.Labels["source"]},
		}
	}

	resourceWatchers.lock.Lock()
	registerWatcherUser(PodResource, metaWatcher, e, true, false)
	resourceWatchers.lock.Unlock()
	commitWatcherOwnership(e, resourceWatchers)
	return e
}

type mockFuncs struct {
	updated kubernetes.Resource
	deleted kubernetes.Resource
	indexed mapstr.M
}

func (f *mockFuncs) update(obj kubernetes.Resource) map[string]mapstr.M {
	accessor, _ := meta.Accessor(obj)
	f.updated = obj
	meta := mapstr.M{
		"kubernetes": mapstr.M{
			"pod": mapstr.M{
				"name": accessor.GetName(),
				"uid":  string(accessor.GetUID()),
			},
		},
	}
	logger := logp.NewLogger("kubernetes") //nolint:forbidigo // test helper
	for k, v := range accessor.GetLabels() {
		kubernetes2.ShouldPut(meta, fmt.Sprintf("kubernetes.%v", k), v, logger)
	}
	kubernetes2.ShouldPut(meta, "orchestrator.cluster.name", "gke-4242", logger)
	id := accessor.GetName()
	return map[string]mapstr.M{id: meta}
}

func (f *mockFuncs) delete(obj kubernetes.Resource) []string {
	accessor, _ := meta.Accessor(obj)
	f.deleted = obj
	return []string{accessor.GetName()}
}

func (f *mockFuncs) index(m mapstr.M) string {
	f.indexed = m
	return m["name"].(string) //nolint:errcheck // test mock
}

type mockWatcher struct {
	handler    kubernetes.ResourceEventHandler
	store      cache.Store
	startCalls int
	stopCalls  int
	startErr   error
}

func newMockWatcher() *mockWatcher {
	return &mockWatcher{
		store: cache.NewStore(func(obj any) (string, error) {
			objName, err := cache.ObjectToName(obj)
			if err != nil {
				return "", err
			}
			return objName.Name, nil
		}),
	}
}

func (m *mockWatcher) GetEventHandler() kubernetes.ResourceEventHandler {
	return m.handler
}

func (m *mockWatcher) Start() error {
	m.startCalls++
	return m.startErr
}

func (m *mockWatcher) Stop() {
	m.stopCalls++
}

func (m *mockWatcher) AddEventHandler(r kubernetes.ResourceEventHandler) {
	m.handler = r
}

func (m *mockWatcher) Store() cache.Store {
	return m.store
}

func (m *mockWatcher) Client() k8s.Interface {
	return nil
}

func (m *mockWatcher) CachedObject() runtime.Object {
	return nil
}

type blockingStore struct {
	cache.Store
	lookupStarted chan<- struct{}
	releaseLookup <-chan struct{}
	lookupOnce    sync.Once
}

func (s *blockingStore) GetByKey(key string) (any, bool, error) {
	s.lookupOnce.Do(func() {
		close(s.lookupStarted)
	})
	<-s.releaseLookup
	return s.Store.GetByKey(key)
}

type coordinatedWatcher struct {
	store   cache.Store
	handler kubernetes.ResourceEventHandler

	started chan struct{}
	stopped chan struct{}

	startOnce sync.Once
	stopOnce  sync.Once
}

func newCoordinatedWatcher(store cache.Store) *coordinatedWatcher {
	return &coordinatedWatcher{
		store:   store,
		started: make(chan struct{}),
		stopped: make(chan struct{}),
	}
}

func (m *coordinatedWatcher) GetEventHandler() kubernetes.ResourceEventHandler {
	return m.handler
}

func (m *coordinatedWatcher) Start() error {
	m.startOnce.Do(func() {
		close(m.started)
	})
	return nil
}

func (m *coordinatedWatcher) Stop() {
	m.stopOnce.Do(func() {
		close(m.stopped)
	})
}

func (m *coordinatedWatcher) AddEventHandler(handler kubernetes.ResourceEventHandler) {
	m.handler = handler
}

func (m *coordinatedWatcher) Store() cache.Store {
	return m.store
}

func (*coordinatedWatcher) Client() k8s.Interface {
	return nil
}

func (*coordinatedWatcher) CachedObject() runtime.Object {
	return nil
}

// cancelErrorWatcher blocks in Start() until Stop() is called, then returns an
// error. Used to simulate a watcher whose Start() is cancelled mid-flight.
type cancelErrorWatcher struct {
	store       cache.Store
	startCalled chan struct{}
	cancel      chan struct{}
	startOnce   sync.Once
	stopOnce    sync.Once
}

func newCancelErrorWatcher() *cancelErrorWatcher {
	return &cancelErrorWatcher{
		store:       newMockWatcher().store,
		startCalled: make(chan struct{}),
		cancel:      make(chan struct{}),
	}
}

func (w *cancelErrorWatcher) Start() error {
	w.startOnce.Do(func() { close(w.startCalled) })
	<-w.cancel
	return fmt.Errorf("start cancelled")
}

func (w *cancelErrorWatcher) Stop() {
	w.stopOnce.Do(func() { close(w.cancel) })
}

func (w *cancelErrorWatcher) Store() cache.Store                              { return w.store }
func (w *cancelErrorWatcher) AddEventHandler(kubernetes.ResourceEventHandler) {}
func (*cancelErrorWatcher) GetEventHandler() kubernetes.ResourceEventHandler  { return nil }
func (*cancelErrorWatcher) Client() k8s.Interface                             { return nil }
func (*cancelErrorWatcher) CachedObject() runtime.Object                      { return nil }

// releasableWatcher blocks in Start() until explicitly released (→ nil) or stopped (→ error).
// startCount is incremented on every entry to Start(), letting tests assert that a shared
// blocking watcher was started exactly once across concurrent callers.
type releasableWatcher struct {
	store       cache.Store
	startCalled chan struct{}
	released    chan struct{}
	stopped     chan struct{}
	startOnce   sync.Once
	releaseOnce sync.Once
	stopOnce    sync.Once
	startCount  atomic.Int64
}

func newReleasableWatcher() *releasableWatcher {
	return &releasableWatcher{
		store:       newMockWatcher().store,
		startCalled: make(chan struct{}),
		released:    make(chan struct{}),
		stopped:     make(chan struct{}),
	}
}

func (w *releasableWatcher) Start() error {
	w.startCount.Add(1)
	w.startOnce.Do(func() { close(w.startCalled) })
	select {
	case <-w.released:
		return nil
	case <-w.stopped:
		return fmt.Errorf("watcher stopped before cache sync")
	}
}

func (w *releasableWatcher) Release() { w.releaseOnce.Do(func() { close(w.released) }) }

func (w *releasableWatcher) Stop() { w.stopOnce.Do(func() { close(w.stopped) }) }

func (w *releasableWatcher) Store() cache.Store                              { return w.store }
func (w *releasableWatcher) AddEventHandler(kubernetes.ResourceEventHandler) {}
func (*releasableWatcher) GetEventHandler() kubernetes.ResourceEventHandler  { return nil }
func (*releasableWatcher) Client() k8s.Interface                             { return nil }
func (*releasableWatcher) CachedObject() runtime.Object                      { return nil }
