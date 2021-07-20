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
	"log"

	"io/ioutil"

	// "github.com/prashantv/gostub"

	// "io/ioutil"
	"os"
	//"github.com/pkg/errors"

	// "strings"
	"errors"
	"fmt"
	"testing"

	//"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	// restclient "k8s.io/client-go/rest"
	// "k8s.io/client-go/tools/clientcmd"
	// clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	// "github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/logp"
)

func TestDiscoverKubernetesNode(t *testing.T) {
	client := k8sfake.NewSimpleClientset()
	logger := logp.NewLogger("autodiscover.node")

	tests := []struct {
		host      string
		inCluster bool
		node      string
		err       error
		name      string
		setEnv    bool
	}{
		{
			name:      "test value from config",
			host:      "worker-1",
			inCluster: false,
			node:      "worker-1",
			err:       nil,
			setEnv:    false,
		},
		{
			name:      "test value with env var",
			host:      "",
			inCluster: false,
			node:      "worker-2",
			err:       nil,
			setEnv:    true,
		},
		{
			name:      "test value with env var not set",
			host:      "",
			inCluster: false,
			node:      "",
			err:       errors.New("kubernetes: Couldn't collect info from any of the files in /etc/machine-id /var/lib/dbus/machine-id: kubernetes: NODE_NAME environment variable was not set"),
			setEnv:    false,
		},
		{
			name:      "test value with inCluster and env var not set",
			host:      "",
			inCluster: true,
			node:      "",
			err:       errors.New("kubernetes: Couldn't get namespace when beat is in cluster with error: open /var/run/secrets/kubernetes.io/serviceaccount/namespace: no such file or directory: kubernetes: NODE_NAME environment variable was not set"),
			setEnv:    false,
		},
		{
			name:      "test value with inCluster, pod not found and env var not set",
			host:      "",
			inCluster: true,
			node:      "",
			err:       errors.New("kubernetes: Querying for pod failed with error: pods \"test-pod\" not found: kubernetes: NODE_NAME environment variable was not set"),
			setEnv:    false,
		},
		{
			name:      "test value with inCluster, pod found and env var not set",
			host:      "",
			inCluster: true,
			node:      "test-node",
			err:       nil,
			setEnv:    false,
		},
		{
			name:      "test value with inCluster, pod found and env var set",
			host:      "",
			inCluster: true,
			node:      "worker-2",
			err:       nil,
			setEnv:    true,
		},
		{
			name:      "test value without inCluster, machine-id empty and env var not set",
			host:      "",
			inCluster: false,
			node:      "",
			err:       errors.New("kubernetes: Couldn't collect info from any of the files in /etc/machine-id /var/lib/dbus/machine-id: kubernetes: NODE_NAME environment variable was not set"),
			setEnv:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			os.Unsetenv("NODE_NAME")
			if test.setEnv {
				os.Setenv("NODE_NAME", "worker-2")
			}
			if test.name == "test value with inCluster, pod not found and env var not set" {
				createNsandHostname()
				defer deleteNamespace()
			}

			if test.name == "test value with inCluster, pod found and env var not set" {
				createNsandHostname()
				defer deleteNamespace()
				err := createPod(client)
				if err != nil {
					t.Fatal(err)
				}
			}

			nodeName, error := DiscoverKubernetesNode(logger, test.host, test.inCluster, client)
			assert.Equal(t, test.node, nodeName)
			if error != nil {
				assert.Equal(t, test.err.Error(), error.Error())
			} else {
				assert.Equal(t, test.err, error)
			}
		})
	}
}

func mockCreateNamespace(namespaceFilePath string) {

	_, err := os.Create(namespaceFilePath)
	if err != nil {
		panic(err)
	}
	d1 := []byte("default")
	err = ioutil.WriteFile(namespaceFilePath, d1, 0644)
	if err != nil {
		panic(err)
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

func createNsandHostname() {
	namespaceFilePath = "test_namespace"
	osHostname = mockHostname
	mockCreateNamespace(namespaceFilePath)
}

func deleteNamespace() {
	e := os.Remove("test_namespace")
	if e != nil {
		log.Fatal(e)
	}
}

func mockHostname() (string, error) {
	return "test-pod", nil
}
