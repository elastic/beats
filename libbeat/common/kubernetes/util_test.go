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

	"github.com/elastic/beats/v7/libbeat/logp"
)

func TestDiscoverKubernetesNode(t *testing.T) {
	client := k8sfake.NewSimpleClientset()
	logger := logp.NewLogger("autodiscover.node")

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
			name:        "test value with env var not set",
			host:        "",
			node:        "",
			err:         errors.New("kubernetes: Couldn't collect info from any of the files in /etc/machine-id /var/lib/dbus/machine-id: kubernetes: NODE_NAME environment variable was not set"),
			setEnv:      false,
			isInCluster: false,
			machineid:   "",
			podname:     "",
			namespace:   "",
		},
		{
			name:        "test value with inCluster and env var not set",
			host:        "",
			node:        "",
			err:         errors.New("kubernetes: Couldn't get namespace when beat is in cluster with error: open /var/run/secrets/kubernetes.io/serviceaccount/namespace: no such file or directory: kubernetes: NODE_NAME environment variable was not set"),
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
			err:         errors.New("kubernetes: Querying for pod failed with error: pods \"test-pod\" not found: kubernetes: NODE_NAME environment variable was not set"),
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
		},
		{
			name:        "test value with inCluster, pod found and env var set",
			host:        "",
			isInCluster: true,
			node:        "worker-2",
			err:         nil,
			setEnv:      true,
			machineid:   "",
			podname:     "test-pod",
			namespace:   "default",
		},
		{
			name:        "test value without inCluster, machine-id empty and env var not set",
			host:        "",
			isInCluster: false,
			node:        "",
			err:         errors.New("kubernetes: Couldn't collect info from any of the files in /etc/machine-id /var/lib/dbus/machine-id: kubernetes: NODE_NAME environment variable was not set"),
			setEnv:      false,
			machineid:   "",
			podname:     "",
			namespace:   "",
		},
		{
			name:        "test value without inCluster, machine-id set, node not found and env var not set",
			host:        "",
			isInCluster: false,
			node:        "",
			err:         errors.New("kubernetes: Couldn't discover node worker-2: kubernetes: NODE_NAME environment variable was not set"),
			setEnv:      false,
			machineid:   "worker-2",
			podname:     "",
			namespace:   "",
		},
		{
			name:        "test value without inCluster, machine-id set, node found and env var not set",
			host:        "",
			isInCluster: false,
			node:        "worker-2",
			err:         nil,
			setEnv:      false,
			machineid:   "worker-2",
			podname:     "",
			namespace:   "",
		},
		{
			name:        "test value without inCluster, machine-id set, node not found and env var set",
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
				os.Setenv("NODE_NAME", "worker-2")
				defer os.Unsetenv("NODE_NAME")
			}
			mhd := createMockhd(test.namespace, test.podname, test.machineid)
			d := &discoveryUtils{eDisc: mhd, client: client, isInCluster: test.isInCluster}

			if test.name == "test value with inCluster, pod found and env var not set" {
				err := createPod(client)

				if err != nil {
					t.Fatal(err)
				}
				defer deletePod(client)
			}

			if test.name == "test value without inCluster, machine-id set, node found and env var not set" {
				err := createNode(client, "worker-2")
				if err != nil {
					t.Fatal(err)
				}
			}
			var nodeName string
			var error error
			if test.host != "" {
				nodeName, error = DiscoverKubernetesNode(logger, test.host, test.isInCluster, client)
			} else {
				nodeName, error = d.discoverKubernetesNode(logger)
			}

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

func deletePod(client kubernetes.Interface) error {
	pod := "test-pod"

	err := client.CoreV1().Pods("default").Delete(context.Background(), pod, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete k8s pod: %v", err)
	}
	return nil
}

func createNode(client kubernetes.Interface, name string) error {
	node := getNodeObject(name)

	_, err := client.CoreV1().Nodes().Create(context.Background(), node, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create k8s node: %v", err)
	}
	return nil
}

func deleteNode(client kubernetes.Interface, node string) error {

	err := client.CoreV1().Nodes().Delete(context.Background(), node, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete k8s node: %v", err)
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

func createMockhd(namespace, podname, machineid string) *mockHostDiscovery {
	return &mockHostDiscovery{namespace: namespace, podname: podname, machineid: machineid}
}

type mockHostDiscovery struct {
	namespace string
	podname   string
	machineid string
}

func (hd *mockHostDiscovery) GetMachineID() string {
	return hd.machineid
}

func (hd *mockHostDiscovery) GetNamespace() (string, error) {
	var error error
	if hd.namespace == "none" {
		error = errors.New("open /var/run/secrets/kubernetes.io/serviceaccount/namespace: no such file or directory")
	}
	return hd.namespace, error
}

func (hd *mockHostDiscovery) GetPodName() (string, error) {
	return hd.podname, nil
}
