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
	"sync"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/elastic/beats/v7/metricbeat/mb"

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
	"github.com/elastic/elastic-agent-autodiscover/kubernetes/metadata"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/stretchr/testify/require"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8smetafake "k8s.io/client-go/metadata/fake"

	"github.com/elastic/elastic-agent-autodiscover/kubernetes"
)

const (
	podBName                  = "pod-b"
	informerTestContainerName = "container"
)

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
	require.Equal(t, 1, len(resourceWatchers.metaWatchersMap))
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
	require.Equal(t, 1, len(resourceWatchers.metaWatchersMap))
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
	require.Equal(t, 2, len(resourceWatchers.metaWatchersMap))
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
	assert.Equal(t, 0.1, metrics.CoresLimit.Value)
	assert.Equal(t, 100*1024*1024.0, metrics.MemoryLimit.Value)

	// modify the limit and verify the new value is present
	pod.Spec.Containers[0].Resources.Limits[v1.ResourceCPU] = resource.MustParse("200m")
	watcher.handler.OnUpdate(pod)
	metrics = containerStore.GetContainerMetrics()
	require.NotNil(t, metrics)
	assert.Equal(t, 0.2, metrics.CoresLimit.Value)

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
	assert.Equal(t, 0.1, metrics.CoresAllocatable.Value)
	assert.Equal(t, 100*1024*1024.0, metrics.MemoryAllocatable.Value)

	// modify the limit and verify the new value is present
	node.Status.Allocatable[v1.ResourceCPU] = resource.MustParse("200m")
	watcher.handler.OnUpdate(node)
	metrics = nodeStore.GetNodeMetrics()
	require.NotNil(t, metrics)
	assert.Equal(t, 0.2, metrics.CoresAllocatable.Value)

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
	require.Equal(t, 0, len(resourceWatchers.metaWatchersMap))
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
	require.Equal(t, len(extras)+1, len(resourceWatchers.metaWatchersMap))
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

	namespaceConfig, err := conf.NewConfigFrom(map[string]interface{}{
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
		AddResourceMetadata: metadata.GetDefaultResourceMetadataConfig(),
	}
	log := logptest.NewTestingLogger(t, selector)
	first := buildTestMetadataEnricher("pod", PodResource, resourceWatchers, config, &funcs, log)
	second := buildTestMetadataEnricher("pod", PodResource, resourceWatchers, config, &funcs, log)

	resourceWatchers.lock.Lock()
	require.Len(t, resourceWatchers.metaWatchersMap[PodResource].users, 2, "same-name enrichers must both own the watcher")
	require.Len(t, resourceWatchers.metaWatchersMap[PodResource].enrichers, 2, "same-name enrichers must both receive invalidation")
	resourceWatchers.lock.Unlock()

	first.Start(resourceWatchers)
	resourceWatchers.lock.Lock()
	require.True(t, resourceWatchers.metaWatchersMap[PodResource].started, "watcher must start")
	resourceWatchers.lock.Unlock()

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

	config := &kubernetesConfig{AddResourceMetadata: metadata.GetDefaultResourceMetadataConfig()}
	log := logptest.NewTestingLogger(t, selector)
	funcs := mockFuncs{}
	pod := buildTestMetadataEnricher("pod", PodResource, resourceWatchers, config, &funcs, log)
	container := buildTestMetadataEnricher("container", PodResource, resourceWatchers, config, &funcs, log)

	pod.Start(resourceWatchers)
	container.Stop(resourceWatchers)
	resourceWatchers.lock.RLock()
	require.Contains(t, resourceWatchers.metaWatchersMap, PodResource, "pod pointer must retain the shared watcher")
	require.Len(t, resourceWatchers.metaWatchersMap[PodResource].users, 1, "only pod ownership remains")
	resourceWatchers.lock.RUnlock()
	require.Equal(t, 0, watcher.stopCalls, "container release must not stop pod's watcher")

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

func TestEnricherConstructorRollbackReleasesRegisteredWatchers(t *testing.T) {
	resourceWatchers := NewWatchers()
	metricsRepo := NewMetricsRepo()
	client := k8sfake.NewSimpleClientset()
	metadataClient := k8smetafake.NewSimpleMetadataClient(k8smetafake.NewTestScheme())
	log := logptest.NewTestingLogger(t, selector)
	config := &kubernetesConfig{
		SyncPeriod:          time.Second,
		AddResourceMetadata: resourceMetadataConfig(t, false, false, false, false),
	}
	e := newMetadataEnricher("state_service", ServiceResource, config, log)

	require.NoError(
		t,
		createAllWatchers(client, metadataClient, e, false, config, log, resourceWatchers, metricsRepo),
		"primary watcher registration must succeed before the simulated constructor failure",
	)
	commonConfig, err := conf.NewConfigFrom(&metadata.Config{})
	require.NoError(t, err, "common metadata config must be valid")
	_, err = createMetadataGenSpecific(client, commonConfig, config.AddResourceMetadata, ServiceResource, resourceWatchers, nil)
	require.Error(t, err, "service metadata generator must fail without a namespace watcher")

	releaseWatcherOwnership(e, resourceWatchers)
	resourceWatchers.lock.RLock()
	require.Empty(t, resourceWatchers.metaWatchersMap, "constructor rollback must evict an unstarted final-owner watcher")
	resourceWatchers.lock.RUnlock()
	require.Empty(t, e.watchedResources, "constructor rollback must clear recorded ownership")
}

func TestClusterScopedConstructorRollbackDoesNotUpgradeNodeScopedOwner(t *testing.T) {
	resourceWatchers := NewWatchers()
	metricsRepo := NewMetricsRepo()
	client := k8sfake.NewSimpleClientset()
	metadataClient := k8smetafake.NewSimpleMetadataClient(k8smetafake.NewTestScheme())
	active := newMockWatcher()
	active.AddEventHandler(kubernetes.ResourceEventHandlerFuncs{})
	metaWatcher := &metaWatcher{
		watcher:   active,
		started:   true,
		users:     make(map[*enricher]watcherRegistration),
		enrichers: make(map[*enricher]struct{}),
		nodeScope: true,
	}
	resourceWatchers.metaWatchersMap[PodResource] = metaWatcher

	config := &kubernetesConfig{
		SyncPeriod:          time.Second,
		AddResourceMetadata: resourceMetadataConfig(t, false, false, false, false),
	}
	log := logptest.NewTestingLogger(t, selector)
	funcs := mockFuncs{}
	nodeScoped := buildTestMetadataEnricherWithScope("pod", PodResource, resourceWatchers, config, &funcs, log, true)
	provisional := newMetadataEnricher("state_pod", PodResource, config, log)
	require.NoError(
		t,
		createAllWatchers(client, metadataClient, provisional, false, config, log, resourceWatchers, metricsRepo),
		"provisional cluster-scoped watcher registration must succeed",
	)
	resourceWatchers.lock.RLock()
	require.NotNil(t, metaWatcher.restartWatcher, "provisional cluster-scoped registration must prepare a replacement")
	require.False(t, metaWatcher.users[provisional].committed, "constructor registration must remain provisional before initialization succeeds")
	resourceWatchers.lock.RUnlock()

	releaseWatcherOwnership(provisional, resourceWatchers)

	resourceWatchers.lock.RLock()
	require.Nil(t, metaWatcher.restartWatcher, "rolling back the last cluster-scoped registration must discard its pending replacement")
	require.True(t, metaWatcher.nodeScope, "rollback must preserve the active node-scoped watcher's scope")
	resourceWatchers.lock.RUnlock()

	nodeScoped.Start(resourceWatchers)
	require.Equal(t, 0, active.stopCalls, "remaining node-scoped owner must not stop the active watcher")
	require.Same(t, active, metaWatcher.watcher, "remaining node-scoped owner must keep the active watcher")

	nodeScoped.Stop(resourceWatchers)
}

func TestPendingScopeUpgradeRetainedForCommittedClusterScopedOwner(t *testing.T) {
	resourceWatchers := NewWatchers()
	active := newMockWatcher()
	pending := newMockWatcher()
	metaWatcher := &metaWatcher{
		watcher:        active,
		started:        true,
		users:          make(map[*enricher]watcherRegistration),
		enrichers:      make(map[*enricher]struct{}),
		nodeScope:      true,
		restartWatcher: pending,
	}
	resourceWatchers.metaWatchersMap[PodResource] = metaWatcher

	config := &kubernetesConfig{AddResourceMetadata: metadata.GetDefaultResourceMetadataConfig()}
	log := logptest.NewTestingLogger(t, selector)
	funcs := mockFuncs{}
	nodeScoped := buildTestMetadataEnricherWithScope("pod", PodResource, resourceWatchers, config, &funcs, log, true)
	firstClusterScoped := buildTestMetadataEnricher("state_pod", PodResource, resourceWatchers, config, &funcs, log)
	secondClusterScoped := buildTestMetadataEnricher("state_container", PodResource, resourceWatchers, config, &funcs, log)

	firstClusterScoped.Stop(resourceWatchers)
	resourceWatchers.lock.RLock()
	require.Same(t, pending, metaWatcher.restartWatcher, "another committed cluster-scoped owner must retain the pending replacement")
	resourceWatchers.lock.RUnlock()

	secondClusterScoped.Stop(resourceWatchers)
	resourceWatchers.lock.RLock()
	require.Nil(t, metaWatcher.restartWatcher, "the last cluster-scoped owner must discard the pending replacement")
	require.True(t, metaWatcher.nodeScope, "discarding a pending replacement must preserve the active watcher scope")
	resourceWatchers.lock.RUnlock()

	nodeScoped.Stop(resourceWatchers)
}

func TestConcurrentNodeScopedStartDoesNotApplyProvisionalScopeUpgrade(t *testing.T) {
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

	config := &kubernetesConfig{AddResourceMetadata: metadata.GetDefaultResourceMetadataConfig()}
	log := logptest.NewTestingLogger(t, selector)
	funcs := mockFuncs{}
	nodeScoped := buildTestMetadataEnricherWithScope("pod", PodResource, resourceWatchers, config, &funcs, log, true)
	provisional := newMetadataEnricher("state_pod", PodResource, config, log)

	resourceWatchers.lock.Lock()
	metaWatcher.restartWatcher = pending
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
	require.Same(t, pending, metaWatcher.restartWatcher, "the provisional owner's replacement must remain available during construction")

	releaseWatcherOwnership(provisional, resourceWatchers)
	nodeScoped.Stop(resourceWatchers)
}

func TestFailedScopeUpgradeLeavesActiveWatcherRunning(t *testing.T) {
	resourceWatchers := NewWatchers()
	active := newMockWatcher()
	pending := newMockWatcher()
	pending.startErr = fmt.Errorf("replacement start failed")
	metaWatcher := &metaWatcher{
		watcher:        active,
		started:        true,
		users:          make(map[*enricher]watcherRegistration),
		enrichers:      make(map[*enricher]struct{}),
		nodeScope:      true,
		restartWatcher: pending,
	}
	resourceWatchers.metaWatchersMap[PodResource] = metaWatcher

	config := &kubernetesConfig{AddResourceMetadata: metadata.GetDefaultResourceMetadataConfig()}
	log := logptest.NewTestingLogger(t, selector)
	funcs := mockFuncs{}
	nodeScoped := buildTestMetadataEnricherWithScope("pod", PodResource, resourceWatchers, config, &funcs, log, true)
	clusterScoped := buildTestMetadataEnricher("state_pod", PodResource, resourceWatchers, config, &funcs, log)

	clusterScoped.Start(resourceWatchers)
	require.Equal(t, 1, pending.startCalls, "pending replacement must be attempted")
	require.Equal(t, 0, active.stopCalls, "failed replacement must not stop the active watcher")
	require.Same(t, active, metaWatcher.watcher, "failed replacement must not replace the active watcher")
	require.Same(t, pending, metaWatcher.restartWatcher, "failed replacement must remain pending")
	require.True(t, metaWatcher.nodeScope, "failed replacement must preserve the active watcher scope")
	require.True(t, metaWatcher.started, "failed replacement must preserve active watcher state")

	clusterScoped.Stop(resourceWatchers)
	nodeScoped.Stop(resourceWatchers)
}

func TestNodeScopeRestartWatcherLifecycle(t *testing.T) {
	resourceWatchers := NewWatchers()
	active := newMockWatcher()
	replacement := newMockWatcher()
	metaWatcher := &metaWatcher{
		watcher:        active,
		started:        true,
		users:          make(map[*enricher]watcherRegistration),
		enrichers:      make(map[*enricher]struct{}),
		nodeScope:      true,
		restartWatcher: replacement,
	}
	resourceWatchers.metaWatchersMap[PodResource] = metaWatcher

	config := &kubernetesConfig{AddResourceMetadata: metadata.GetDefaultResourceMetadataConfig()}
	log := logptest.NewTestingLogger(t, selector)
	funcs := mockFuncs{}
	nodeScoped := buildTestMetadataEnricherWithScope("pod", PodResource, resourceWatchers, config, &funcs, log, true)
	clusterScoped := buildTestMetadataEnricher("state_pod", PodResource, resourceWatchers, config, &funcs, log)

	clusterScoped.Start(resourceWatchers)
	require.Equal(t, 1, active.stopCalls, "scope upgrade must stop the old active watcher")
	require.Equal(t, 1, replacement.startCalls, "scope upgrade must start the pending cluster-scoped watcher")
	require.Same(t, replacement, metaWatcher.watcher, "replacement must become the active watcher")
	require.Nil(t, metaWatcher.restartWatcher, "successful scope upgrade must clear the pending watcher")

	clusterScoped.Stop(resourceWatchers)
	require.Equal(t, 0, replacement.stopCalls, "node-scoped owner must retain the replacement watcher")
	nodeScoped.Stop(resourceWatchers)
	require.Equal(t, 1, replacement.stopCalls, "final owner must stop the active replacement watcher")
	resourceWatchers.lock.RLock()
	require.NotContains(t, resourceWatchers.metaWatchersMap, PodResource, "final owner must evict the scope-upgraded watcher")
	resourceWatchers.lock.RUnlock()
}

func TestFinalOwnerEvictionDiscardsPendingRestartWatcher(t *testing.T) {
	resourceWatchers := NewWatchers()
	active := newMockWatcher()
	pending := newMockWatcher()
	metaWatcher := &metaWatcher{
		watcher:        active,
		started:        true,
		users:          make(map[*enricher]watcherRegistration),
		enrichers:      make(map[*enricher]struct{}),
		nodeScope:      true,
		restartWatcher: pending,
	}
	resourceWatchers.metaWatchersMap[PodResource] = metaWatcher

	config := &kubernetesConfig{AddResourceMetadata: metadata.GetDefaultResourceMetadataConfig()}
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
	podA.Start(resourceWatchers)
	containerA.Start(resourceWatchers)
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
	podB.Start(resourceWatchers)
	containerB.Start(resourceWatchers)
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

func TestConcurrentInvalidationAndEnrichmentAcrossGenerations(t *testing.T) {
	resourceWatchers := NewWatchers()
	log := logptest.NewTestingLogger(t, selector)
	config := &kubernetesConfig{AddResourceMetadata: metadata.GetDefaultResourceMetadataConfig()}
	resource := &kubernetes.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deployment",
			Namespace: "default",
		},
	}

	createGeneration := func() (*enricher, *mockWatcher) {
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
				deployment := resource.(*kubernetes.Deployment)
				return map[string]mapstr.M{
					join(deployment.Namespace, deployment.Name): {
						"kubernetes": mapstr.M{"labels": mapstr.M{"generation": "current"}},
					},
				}
			},
			func(resource kubernetes.Resource) []string {
				deployment := resource.(*kubernetes.Deployment)
				return []string{join(deployment.Namespace, deployment.Name)}
			},
			func(event mapstr.M) string {
				return join(getString(event, mb.ModuleDataKey+".namespace"), getString(event, "name"))
			},
			log,
		)
		require.NoError(t, watcher.Store().Add(resource), "deployment must be added to the mock watcher store")
		e.Start(resourceWatchers)
		return e, watcher
	}

	first, firstWatcher := createGeneration()
	var workers sync.WaitGroup
	workers.Add(2)
	go func() {
		defer workers.Done()
		for range 100 {
			firstWatcher.handler.OnUpdate(resource)
		}
	}()
	go func() {
		defer workers.Done()
		for range 100 {
			first.Enrich([]mapstr.M{{
				"name":           resource.Name,
				mb.ModuleDataKey: mapstr.M{"namespace": resource.Namespace},
			}})
		}
	}()
	workers.Wait()
	first.Stop(resourceWatchers)

	second, secondWatcher := createGeneration()
	require.NotSame(t, firstWatcher, secondWatcher, "next generation must use an independent watcher")
	second.Enrich([]mapstr.M{{
		"name":           resource.Name,
		mb.ModuleDataKey: mapstr.M{"namespace": resource.Namespace},
	}})
	second.Stop(resourceWatchers)
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

	enricher.Start(resourceWatchers)
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

	enricher.Start(resourceWatchers)
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
	nodeConfig, err := conf.NewConfigFrom(map[string]interface{}{"enabled": node})
	require.NoError(t, err, "node metadata config must be valid")
	namespaceConfig, err := conf.NewConfigFrom(map[string]interface{}{"enabled": namespace})
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
		pod := resource.(*kubernetes.Pod)
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
		pod := resource.(*kubernetes.Pod)
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
	logger := logp.NewLogger("kubernetes")
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
	return m["name"].(string)
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
		store: cache.NewStore(func(obj interface{}) (string, error) {
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
