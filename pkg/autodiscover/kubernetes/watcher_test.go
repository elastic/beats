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
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	cachetest "k8s.io/client-go/tools/cache/testing"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
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

// TestWatcherStopPreventsRestart reproduces the shared-informer lifecycle bug:
// a watcher that has been started and stopped must never be restarted, because
// the underlying one-shot SharedInformer would refuse to run again while
// WaitForCacheSync still reports success from the first run. Start must fail
// with ErrWatcherStopped instead of silently returning a frozen watcher.
func TestWatcherStopPreventsRestart(t *testing.T) {
	client := fake.NewSimpleClientset()
	listWatch := cachetest.NewFakeControllerSource()
	resource := &Pod{}
	informer := cache.NewSharedInformer(listWatch, resource, 0)
	w, err := NewNamedWatcherWithInformer(
		"test",
		client,
		resource,
		informer,
		logptest.NewTestingLogger(t, ""),
		WatchOptions{})
	require.NoError(t, err)

	require.NoError(t, w.Start())
	w.Stop()

	// Give the informer's Run loop time to observe the cancelled context and
	// mark itself stopped so IsStopped() reflects the terminal state.
	assert.Eventually(t, func() bool {
		//nolint:errcheck // It's a test, it can panic on failure
		return w.(*watcher).informer.IsStopped()
	}, time.Second*5, time.Millisecond)

	// Try starting watcher again and check for the correct error returned
	require.ErrorIs(t, w.Start(), ErrWatcherStopped, "expected ErrWatcherStopped, got: %v", err)
}

// TestWatcherStartRejectsInvalidLifecycleStates verifies that Start refuses to
// run when any single terminal lifecycle signal is present, even if the others
// look healthy.
func TestWatcherStartRejectsInvalidLifecycleStates(t *testing.T) {
	newWatcher := func(t *testing.T) *watcher {
		client := fake.NewSimpleClientset()
		listWatch := cachetest.NewFakeControllerSource()
		resource := &Pod{}
		informer := cache.NewSharedInformer(listWatch, resource, 0)
		w, err := NewNamedWatcherWithInformer(
			"test",
			client,
			resource,
			informer,
			logptest.NewTestingLogger(t, ""),
			WatchOptions{})
		require.NoError(t, err)
		//nolint:errcheck // It's a test, we know the underlying type
		return w.(*watcher)
	}

	t.Run("queue shutting down", func(t *testing.T) {
		w := newWatcher(t)
		w.queue.ShutDown()
		require.False(t, w.informer.IsStopped())
		require.NoError(t, w.ctx.Err())

		require.ErrorIs(t, w.Start(), ErrWatcherStopped, "expected ErrWatcherStopped")
	})

	t.Run("context cancelled", func(t *testing.T) {
		w := newWatcher(t)
		w.stop() // context cancel func
		require.False(t, w.informer.IsStopped())
		require.False(t, w.queue.ShuttingDown())
		require.ErrorIs(t, w.ctx.Err(), context.Canceled)

		require.ErrorIs(t, w.Start(), ErrWatcherStopped, "expected ErrWatcherStopped")
	})

	t.Run("informer stopped", func(t *testing.T) {
		w := newWatcher(t)
		stopCh := make(chan struct{})

		go w.informer.Run(stopCh)
		require.Eventually(t, w.informer.HasSynced, time.Second*5, time.Millisecond)

		close(stopCh)

		require.Eventually(t, func() bool {
			return w.informer.IsStopped()
		}, time.Second*5, time.Millisecond)

		require.False(t, w.queue.ShuttingDown())
		require.NoError(t, w.ctx.Err())

		require.ErrorIs(t, w.Start(), ErrWatcherStopped, "expected ErrWatcherStopped")
	})
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
		AddFunc: func(obj any) {
			added = true
		},
		UpdateFunc: func(obj any) {
			updated = true
		},
		DeleteFunc: func(obj any) {
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
		WatchOptions{IsUpdated: func(old, new any) bool {
			return true
		}})
	require.NoError(t, err)

	var updated bool

	watcher.AddEventHandler(ResourceEventHandlerFuncs{
		UpdateFunc: func(obj any) {
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
