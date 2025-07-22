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

package kubernetes

import (
	"testing"
	"time"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	cachetest "k8s.io/client-go/tools/cache/testing"
)

func TestWatcherStartAndStop(t *testing.T) {
	client := fake.NewSimpleClientset()
	listWatch := cachetest.NewFakeControllerSource()
	resource := &Pod{}
	informer := cache.NewSharedInformer(listWatch, resource, 0)
	watcher, err := NewNamedWatcherWithInformer("test", client, resource, informer, logptest.NewTestingLogger(t, ""), WatchOptions{})
	require.NoError(t, err)
	require.NoError(t, watcher.Start())
	watcher.Stop()
}

func TestWatcherHandlers(t *testing.T) {
	client := fake.NewSimpleClientset()
	listWatch := cachetest.NewFakeControllerSource()
	resource := &Pod{}
	informer := cache.NewSharedInformer(listWatch, resource, 0)
	watcher, err := NewNamedWatcherWithInformer("test", client, resource, informer, logptest.NewTestingLogger(t, ""), WatchOptions{})
	require.NoError(t, err)

	var added, updated, deleted bool

	watcher.AddEventHandler(ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			added = true
		},
		UpdateFunc: func(obj interface{}) {
			updated = true
		},
		DeleteFunc: func(obj interface{}) {
			deleted = true
		},
	})

	require.NoError(t, watcher.Start())
	defer watcher.Stop()

	pod := &Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "test",
			UID:             types.UID("poduid"),
			Namespace:       "test",
			ResourceVersion: "1",
		},
	}
	// add a resource
	listWatch.Add(pod)
	assert.Eventually(t, func() bool {
		return added
	}, time.Second*5, time.Millisecond)

	// update the resource
	modifiedPod := pod.DeepCopy()
	modifiedPod.SetResourceVersion("2")
	listWatch.Modify(modifiedPod)
	assert.Eventually(t, func() bool {
		return updated
	}, time.Second*5, time.Millisecond)

	// delete the resource
	listWatch.Delete(modifiedPod)
	assert.Eventually(t, func() bool {
		return deleted
	}, time.Second*5, time.Millisecond)
}

func TestWatcherIsUpdated(t *testing.T) {
	client := fake.NewSimpleClientset()
	listWatch := cachetest.NewFakeControllerSource()
	resource := &Pod{}
	informer := cache.NewSharedInformer(listWatch, resource, 0)
	// set a custom IsUpdated that always returns true
	watcher, err := NewNamedWatcherWithInformer("test", client, resource, informer,
		logptest.NewTestingLogger(t, ""),
		WatchOptions{IsUpdated: func(old, new interface{}) bool {
			return true
		}})
	require.NoError(t, err)

	var updated bool

	watcher.AddEventHandler(ResourceEventHandlerFuncs{
		UpdateFunc: func(obj interface{}) {
			updated = true
		},
	})

	require.NoError(t, watcher.Start())
	defer watcher.Stop()

	pod := &Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			UID:       types.UID("poduid"),
			Namespace: "test",
		},
	}
	listWatch.Add(pod)

	// update the resource, but don't actually change it
	// with the default IsUpdated, our handler wouldn't be called, but with our custom one, it will
	modifiedPod := pod.DeepCopy()
	listWatch.Modify(modifiedPod)
	assert.Eventually(t, func() bool {
		return updated
	}, time.Second*5, time.Millisecond)

}

func TestCachedObject(t *testing.T) {
	t.Skip("Currently bugged, and not used anywhere")
	client := fake.NewSimpleClientset()
	listWatch := cachetest.NewFakeControllerSource()
	resource := &Namespace{}
	informer := cache.NewSharedInformer(listWatch, resource, 0)
	watcher, err := NewNamedWatcherWithInformer("test", client, resource, informer, logptest.NewTestingLogger(t, ""), WatchOptions{})
	require.NoError(t, err)

	require.NoError(t, watcher.Start())
	defer watcher.Stop()

	namespace := &Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "test",
			UID:             types.UID("poduid"),
			Namespace:       "test",
			ResourceVersion: "1",
		},
	}
	listWatch.Add(namespace)
	assert.EventuallyWithT(t, func(collectT *assert.CollectT) {
		assert.Equal(collectT, namespace, watcher.CachedObject())
	}, time.Second*5, time.Millisecond)
}
