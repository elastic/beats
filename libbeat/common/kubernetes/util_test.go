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
	"fmt"
	"os"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/menderesk/beats/v7/libbeat/logp"
)

func TestDiscoverKubernetesNode(t *testing.T) {
	client := k8sfake.NewSimpleClientset()
	logger := logp.NewLogger("autodiscover.node")
	ge := errors.New("kubernetes: Node could not be discovered with any known method. Consider setting env var NODE_NAME")
	tests := []struct {
		host        string
		node        string
		err         error
		name        string
		setEnv      bool
		isInCluster bool
		machineid   string
		podname     string
		namespace   string
		init        func(*testing.T, kubernetes.Interface)
	}{
		{
			name:        "test value from config",
			host:        "worker-1",
			node:        "worker-1",
			err:         nil,
			setEnv:      false,
			isInCluster: true,
			machineid:   "",
			podname:     "",
			namespace:   "",
		},
		{
			name:        "test value with env var",
			host:        "",
			node:        "worker-2",
			err:         nil,
			setEnv:      true,
			isInCluster: false,
			machineid:   "",
			podname:     "",
			namespace:   "",
		},
		{
			name:        "test value with not incluster, machine ID not retrieved, env var not set",
			host:        "",
			node:        "",
			err:         ge,
			setEnv:      false,
			isInCluster: false,
			machineid:   "",
			podname:     "",
			namespace:   "",
		},
		{
			name:        "test value with inCluster , serviceaccount namespace not found and env var not set",
			host:        "",
			node:        "",
			err:         ge,
			setEnv:      false,
			isInCluster: true,
			machineid:   "",
			podname:     "",
			namespace:   "none",
		},
		{
			name:        "test value with inCluster, pod not found and env var not set",
			host:        "",
			isInCluster: true,
			node:        "",
			err:         ge,
			setEnv:      false,
			machineid:   "",
			podname:     "test-pod",
			namespace:   "default",
		},
		{
			name:        "test value with inCluster, pod found and env var not set",
			host:        "",
			isInCluster: true,
			node:        "test-node",
			err:         nil,
			setEnv:      false,
			machineid:   "",
			podname:     "test-pod",
			namespace:   "default",
			init:        createResources,
		},
		{
			name:        "test value with inCluster, pod found and env var set",
			host:        "",
			isInCluster: true,
			node:        "test-node",
			err:         nil,
			setEnv:      true,
			machineid:   "",
			podname:     "test-pod",
			namespace:   "default",
			init:        createResources,
		},
		{
			name:        "test value without inCluster, machine-ID empty and env var not set",
			host:        "",
			isInCluster: false,
			node:        "",
			err:         ge,
			setEnv:      false,
			machineid:   "",
			podname:     "",
			namespace:   "",
		},
		{
			name:        "test value without inCluster, machine-ID set, node not found and env var not set",
			host:        "",
			isInCluster: false,
			node:        "",
			err:         ge,
			setEnv:      false,
			machineid:   "worker-2",
			podname:     "",
			namespace:   "",
		},
		{
			name:        "test value without inCluster, machine-ID set, node found and env var not set",
			host:        "",
			isInCluster: false,
			node:        "worker-2",
			err:         nil,
			setEnv:      false,
			machineid:   "worker-2",
			podname:     "",
			namespace:   "",
			init:        createResources,
		},
		{
			name:        "test value without inCluster, machine-ID set, node not found and env var set",
			host:        "",
			isInCluster: false,
			node:        "worker-2",
			err:         nil,
			setEnv:      true,
			machineid:   "worker-2",
			podname:     "",
			namespace:   "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			if test.setEnv {

				if err := os.Setenv("NODE_NAME", "worker-2"); err != nil {
					t.Fatal(err)
				}
				defer func() {
					if err := os.Unsetenv("NODE_NAME"); err != nil {
						t.Fatal(err)
					}
				}()
			}
			mdu := createMockdu(test.namespace, test.podname, test.machineid)
			if test.init != nil {
				test.init(t, client)
			}

			var nodeName string
			var error error
			nd := &DiscoverKubernetesNodeParams{ConfigHost: test.host, Client: client, IsInCluster: test.isInCluster, HostUtils: mdu}
			nodeName, error = DiscoverKubernetesNode(logger, nd)

			assert.Equal(t, test.node, nodeName)
			if error != nil {
				assert.Equal(t, test.err.Error(), error.Error())
			} else {
				assert.Equal(t, test.err, error)
			}

		})
	}
}

func createPod(client kubernetes.Interface) error {
	pod := getPodObject()

	_, err := client.CoreV1().Pods(pod.Namespace).Create(context.Background(), pod, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create k8s pod: %v", err)
	}
	return nil
}

func createResources(t *testing.T, client kubernetes.Interface) {
	err := createPod(client)
	if err != nil {
		t.Fatal(err)
	}

	err = createNode(client, "worker-2")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		pod := "test-pod"
		err := client.CoreV1().Pods("default").Delete(context.Background(), pod, metav1.DeleteOptions{})
		if err != nil {
			t.Fatalf("failed to delete k8s pod: %v", err)
		}

		err = client.CoreV1().Nodes().Delete(context.Background(), "worker-2", metav1.DeleteOptions{})
		if err != nil {
			t.Fatalf("failed to delete k8s node: %v", err)
		}

	})
}

func createNode(client kubernetes.Interface, name string) error {
	node := getNodeObject(name)

	_, err := client.CoreV1().Nodes().Create(context.Background(), node, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create k8s node: %v", err)
	}
	return nil
}

func getPodObject() *core.Pod {
	return &core.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			Labels: map[string]string{
				"app": "demo",
			},
		},
		Spec: core.PodSpec{
			NodeName: "test-node",
			Containers: []core.Container{
				{
					Name:            "busybox",
					Image:           "busybox",
					ImagePullPolicy: core.PullIfNotPresent,
					Command: []string{
						"sleep",
						"3600",
					},
				},
			},
		},
	}
}

func getNodeObject(name string) *core.Node {
	return &core.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"name": name,
			},
		},
		Spec:   core.NodeSpec{},
		Status: core.NodeStatus{NodeInfo: core.NodeSystemInfo{MachineID: name}},
	}
}

func createMockdu(namespace, podname, machineid string) *mockDiscoveryUtils {
	return &mockDiscoveryUtils{namespace: namespace, podname: podname, machineid: machineid}
}

type mockDiscoveryUtils struct {
	namespace string
	podname   string
	machineid string
}

func (hd *mockDiscoveryUtils) GetMachineID() string {
	return hd.machineid
}

func (hd *mockDiscoveryUtils) GetNamespace() (string, error) {
	var error error
	if hd.namespace == "none" {
		error = errors.New("open /var/run/secrets/kubernetes.io/serviceaccount/namespace: no such file or directory")
	}
	return hd.namespace, error
}

func (hd *mockDiscoveryUtils) GetPodName() (string, error) {
	return hd.podname, nil
}
