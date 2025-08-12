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

//This file was contributed to by generative AI

//go:build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/elastic/beats/v7/libbeat/tests/integration"
	"github.com/elastic/elastic-agent-autodiscover/docker"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/gofrs/uuid/v5"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/kind/pkg/apis/config/v1alpha4"
	"sigs.k8s.io/kind/pkg/cluster"
	"sigs.k8s.io/kind/pkg/cmd"
)

func TestHintsDocker(t *testing.T) {
	containerID := startFlogContainer(t)
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)

	cfgYAML := getConfig(t, nil, "autodiscover", "docker.yml")
	filebeat.WriteConfigFile(cfgYAML)
	filebeat.Start()

	// By ensuring the Filestream input started with the correct ID, we're
	// testing that the whole autodiscover + hints is working as expected.
	filebeat.WaitForLogs(
		fmt.Sprintf(
			`"message":"Input 'filestream' starting","service.name":"filebeat","id":"container-logs-%s"`,
			containerID,
		),
		30*time.Second,
		"Filestream did not start for the test container")
}

func TestHintsKubernetes(t *testing.T) {
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)

	kubeConfigPath, noneName, containerID := startFlogInKubernetes(t, filebeat.TempDir())

	cfgYAML := getConfig(
		t,
		map[string]any{
			"kubeConfig": kubeConfigPath,
			"nodeName":   noneName,
		},
		"autodiscover",
		"k8s.yml")
	filebeat.WriteConfigFile(cfgYAML)
	filebeat.Start()

	// By ensuring the Filestream input started with the correct ID, we're
	// testing that the whole autodiscover + hints is working as expected.
	filebeat.WaitForLogs(
		fmt.Sprintf(
			`"message":"Input 'filestream' starting","service.name":"filebeat","id":"container-logs-%s"`,
			containerID,
		),
		30*time.Second,
		"Filestream did not start for the test container")
}

func startFlogInKubernetes(t *testing.T, tempDir string) (string, string, string) {
	uid := uuid.Must(uuid.NewV4()).String()
	provider := cluster.NewProvider(
		cluster.ProviderWithLogger(cmd.NewLogger()),
	)

	clusterName := fmt.Sprintf("test-cluster-%s", uid)
	err := provider.Create(clusterName, cluster.CreateWithV1Alpha4Config(&v1alpha4.Cluster{}))
	if err != nil {
		t.Fatalf("could not create cluster: %s", err)
	}

	t.Cleanup(func() {
		if err := provider.Delete(clusterName, ""); err != nil {
			t.Logf("could not delete K8s cluster: %s", err)
		}
	})

	time.Sleep(30 * time.Second)

	kubeConfig, err := provider.KubeConfig(clusterName, false)
	if err != nil {
		t.Fatalf("could not get KubeConfig: %s", err)
	}

	kubeConfigPath := filepath.Join(tempDir, "kube-config")
	if err := os.WriteFile(kubeConfigPath, []byte(kubeConfig), 0666); err != nil {
		t.Fatalf("cannot write kube config file: %s", err)
	}

	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeConfig))
	if err != nil {
		t.Fatal(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		t.Fatal(err)
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "flog-pod-" + uid,
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "flog",
					Image: "mingrammer/flog",
					Args:  []string{"-s", "1", "-d", "1", "-l"},
				},
			},
		},
	}

	pod, err = clientset.CoreV1().Pods("default").Create(t.Context(), pod, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("could not create pod: %s", err)
	}

	t.Cleanup(func() {
		err := clientset.CoreV1().Pods("default").Delete(context.Background(), pod.Name, metav1.DeleteOptions{})
		if err != nil {
			t.Logf("could not remove pod: %s", err)
		}
	})

	// Wait for pod to be running
	for range 60 {
		pod, err = clientset.CoreV1().Pods("default").Get(context.Background(), pod.Name, metav1.GetOptions{})
		if err != nil {
			// We can probably ignore it and just time out
			t.Logf("could not get pod: %s", err)
		}

		if pod.Status.Phase == corev1.PodRunning && len(pod.Status.ContainerStatuses) > 0 {
			containerID := pod.Status.ContainerStatuses[0].ContainerID
			if containerID != "" {
				// Remove the runtime prefix (e.g., "containerd://")
				if idx := strings.Index(containerID, "://"); idx != -1 {
					return kubeConfigPath, pod.Spec.NodeName, containerID[idx+3:]
				}
				return kubeConfigPath, pod.Spec.NodeName, containerID
			}
		}

		time.Sleep(time.Second)
	}

	t.Fatalf("pod did not start within timeout")
	return kubeConfigPath, pod.Spec.NodeName, ""
}

// startFlogContainer starts a `mingrammer/flog` that logs one line every
// second. The container ID is returned and the container is stopped at the
// end of the test. On error the test fails by calling t.Fatalf
func startFlogContainer(t *testing.T) string {
	ctx := t.Context()
	img := "mingrammer/flog"
	cli, err := docker.NewClient(client.DefaultDockerHost, nil, nil, logp.NewNopLogger())
	if err != nil {
		t.Fatalf("cannot create Docker client: %s", err)
	}

	resp, err := cli.ContainerCreate(
		ctx,
		&container.Config{
			Image: img,
			Cmd:   []string{"-l", "-d", "1", "-s", "1"},
		}, nil, nil, nil, "")
	if err != nil {
		t.Fatalf("cannot create container for %q: %s", img, err)
	}

	err = cli.ContainerStart(ctx, resp.ID, container.StartOptions{})
	if err != nil {
		t.Fatalf("cannot start container: %s", err)
	}

	t.Cleanup(func() {
		ctx := context.Background()
		if err := cli.ContainerStop(ctx, resp.ID, container.StopOptions{}); err != nil {
			t.Errorf("cannot stop container: %s", err)
		}
		if err := cli.ContainerRemove(ctx, resp.ID, container.RemoveOptions{}); err != nil {
			t.Errorf("cannot remove container: %s", err)
		}
	})
	return resp.ID
}
