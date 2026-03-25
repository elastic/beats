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

// This file was contributed to by generative AI

//go:build integration && !requirefips

package integration

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/kind/pkg/apis/config/v1alpha4"
	"sigs.k8s.io/kind/pkg/cluster"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
	"github.com/elastic/elastic-agent-autodiscover/docker"
	"github.com/elastic/elastic-agent-libs/logp"
)

func TestHintsDocker(t *testing.T) {
	containerID := startFlogDocker(t)
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
	filebeat.WaitLogsContains(
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

	kubeConfigPath, noneName, containerID := startFlogKubernetes(t, filebeat.TempDir())

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
	filebeat.WaitLogsContains(
		fmt.Sprintf(
			`"message":"Input 'filestream' starting","service.name":"filebeat","id":"container-logs-%s"`,
			containerID,
		),
		30*time.Second,
		"Filestream did not start for the test container")
}

func TestAutodiscoverFilestreamTakeOverDoesNotReingest(t *testing.T) {
	integration.EnsureESIsRunning(t)

	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)
	workDir := filebeat.TempDir()
	kubeConfigPath, nodeName, podName := startFlogKubernetesForTakeOver(t, workDir)
	esURL := integration.GetESAdminURL(t, "http")
	esUser := esURL.User.Username()
	esPass, _ := esURL.User.Password()
	index := fmt.Sprintf("test-autodiscover-take-over-%s", uuid.Must(uuid.NewV4()).String())

	tmplVars := map[string]any{
		"homeFolder": workDir,
		"kubeConfig": kubeConfigPath,
		"nodeName":   nodeName,
		"podName":    podName,
		"esHost":     fmt.Sprintf("%s://%s", esURL.Scheme, esURL.Host),
		"esUser":     esUser,
		"esPass":     esPass,
		"index":      index,
	}

	filebeat.WriteConfigFile(
		getConfig(t, tmplVars, "autodiscover", "take-over-log-input-k8s.yml"),
	)
	filebeat.Start()

	// Wait until at least 5 events are ingested
	require.Eventually(
		t,
		func() bool { return countEventsInES(t, index, 1000) >= 5 },
		30*time.Second,
		200*time.Millisecond,
		"did not ingest the initial events")

	filebeat.Stop()
	logInputIngested := countEventsInES(t, index, 1000)

	// Re-Start Filebeat with Filestream and take_over enabled
	filebeat.WriteConfigFile(
		getConfig(t, tmplVars, "autodiscover", "take-over-filestream-input-k8s.yml"),
	)
	filebeat.Start()

	// Wait for at least two extra events to be ingested
	require.EventuallyWithT(
		t,
		func(collect *assert.CollectT) {
			totalEvents := countEventsInES(t, index, 1000)
			if totalEvents <= logInputIngested+2 {
				collect.Errorf(
					"expecting more events in Elasticsearch than the %d from the first run",
					logInputIngested,
				)
			}
		},
		10*time.Second,
		time.Second,
		"No new events ingested")

	// Stop the pod and get the total number of events generated
	generatedEvents := stopPodK8sAndCountLogs(t, kubeConfigPath, podName)

	// Wait for Filebeat to fully ingest the file, we do it by waiting the file
	// to be closed due to inactivity.
	filebeat.WaitLogsContains(
		"File is inactive. Closing.",
		20*time.Second,
		"Filebeat did not close the log file due to inactivity")

	// Wait until at least all published events can be found in Elasticsearch
	require.Eventuallyf(
		t,
		func() bool { return countEventsInES(t, index, generatedEvents+100) >= generatedEvents },
		30*time.Second,
		200*time.Millisecond,
		"did not ingest all generated events from pod %s", podName,
	)

	filebeat.Stop()

	totalEventsIngested := countEventsInES(t, index, generatedEvents+100)

	require.Equalf(
		t,
		generatedEvents,
		totalEventsIngested,
		"file re-ingestion has occurred\n"+
			"Generated Events: %d\n"+
			"Events ingested by the Log input: %d\n"+
			"Total number of events ingested: %d",
		generatedEvents,
		logInputIngested,
		totalEventsIngested,
	)
}

func startFlogKubernetes(t *testing.T, tempDir string) (string, string, string) {
	uid := uuid.Must(uuid.NewV4()).String()

	defer func() {
		if t.Failed() {
			t.Log("To see the Kind logs search for 'cluster.ProviderWithLogger' and uncomment it.")
		}
	}()
	provider := cluster.NewProvider(
	// Uncomment the next line to have Kind logs written to stderr.
	// You will also have to import "sigs.k8s.io/kind/pkg/cmd"
	// cluster.ProviderWithLogger(cmd.NewLogger()),
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

	var kubeConfig string
	require.Eventually(t, func() bool {
		kubeConfig, err = provider.KubeConfig(clusterName, false)
		return err == nil
	}, 30*time.Second, 100*time.Millisecond, "could not get kube config")

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
		// by the time Cleanup runs, t.Context has been cancelled, so we need a new context here
		err := clientset.CoreV1().Pods("default").Delete(context.Background(), pod.Name, metav1.DeleteOptions{})
		if err != nil {
			t.Logf("could not remove pod: %s", err)
		}
	})

	var containerID string
	var podNodeName string
	require.Eventually(
		t,
		func() bool {
			pod, err = clientset.CoreV1().Pods("default").Get(t.Context(), pod.Name, metav1.GetOptions{})
			if err != nil {
				return false
			}

			if pod.Status.Phase == corev1.PodRunning && len(pod.Status.ContainerStatuses) > 0 {
				containerID = pod.Status.ContainerStatuses[0].ContainerID
				if containerID != "" {
					podNodeName = pod.Spec.NodeName
					// Remove the runtime prefix (e.g., "containerd://")
					if idx := strings.Index(containerID, "://"); idx != -1 {
						containerID = containerID[idx+3:]
						return true
					}
					return true
				}
			}

			return false
		},
		60*time.Second,
		100*time.Millisecond,
		"pod did not start within timeout",
	)

	return kubeConfigPath, podNodeName, containerID
}

// startFlogDocker starts a `mingrammer/flog` that logs one line every
// second. The container ID is returned and the container is stopped at the
// end of the test. On error the test fails by calling t.Fatalf
func startFlogDocker(t *testing.T) string {
	ctx := t.Context()
	img := "mingrammer/flog:0.4.3"
	cli, err := docker.NewClient(client.DefaultDockerHost, nil, nil, logp.NewNopLogger())
	if err != nil {
		t.Fatalf("cannot create Docker client: %s", err)
	}

	// Pull the image first
	reader, err := cli.ImagePull(ctx, img, image.PullOptions{})
	if err != nil {
		t.Fatalf("cannot pull image %q: %s", img, err)
	}
	defer reader.Close()

	// Wait for the pull to complete by reading the response
	_, err = io.Copy(io.Discard, reader)
	if err != nil {
		t.Fatalf("error while pulling image %q: %s", img, err)
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

// startFlogKubernetesForTakeOver creates a Kind cluster and starts a flog pod
// that writes one log line per second to stdout.
func startFlogKubernetesForTakeOver(t *testing.T, workDir string) (kubeConfigPath, nodeName, podName string) {
	uid := uuid.Must(uuid.NewV4()).String()

	defer func() {
		if t.Failed() {
			t.Log("To see the Kind logs search for 'cluster.ProviderWithLogger' and uncomment it.")
		}
	}()

	provider := cluster.NewProvider(
	// Uncomment the next line to have Kind logs written to stderr.
	// You will also have to import "sigs.k8s.io/kind/pkg/cmd"
	// cluster.ProviderWithLogger(cmd.NewLogger()),
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

	var kubeConfig string
	require.Eventually(t, func() bool {
		kubeConfig, err = provider.KubeConfig(clusterName, false)
		return err == nil
	}, 30*time.Second, 100*time.Millisecond, "could not get kube config")

	kubeConfigPath = filepath.Join(workDir, "kube-config")
	if err = os.WriteFile(kubeConfigPath, []byte(kubeConfig), 0666); err != nil {
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

	podName = "flog-pod-" + uid
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
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
		ctx := context.Background()
		if err := clientset.CoreV1().Pods("default").Delete(ctx, pod.Name, metav1.DeleteOptions{}); err != nil {
			t.Logf("could not remove pod: %s", err)
		}
	})

	require.Eventually(
		t,
		func() bool {
			pod, err = clientset.CoreV1().Pods("default").Get(t.Context(), pod.Name, metav1.GetOptions{})
			if err != nil {
				return false
			}
			if pod.Status.Phase == corev1.PodRunning &&
				len(pod.Status.ContainerStatuses) > 0 &&
				pod.Spec.NodeName != "" {
				nodeName = pod.Spec.NodeName
				return true
			}
			return false
		},
		60*time.Second,
		100*time.Millisecond,
		"pod did not start within timeout",
	)

	return
}

// stopPodK8sAndCountLogs returns the number of log lines generated by the pod
// and then deletes it.
func stopPodK8sAndCountLogs(t *testing.T, kubeConfigPath, podName string) int {
	data, err := os.ReadFile(kubeConfigPath)
	if err != nil {
		t.Fatalf("cannot read kube config: %s", err)
	}
	config, err := clientcmd.RESTConfigFromKubeConfig(data)
	if err != nil {
		t.Fatalf("cannot build REST config: %s", err)
	}
	cs, err := kubernetes.NewForConfig(config)
	if err != nil {
		t.Fatalf("cannot create clientset: %s", err)
	}

	logsReader, err := cs.CoreV1().Pods("default").GetLogs(podName, &corev1.PodLogOptions{
		Container: "flog",
		Follow:    true,
	}).Stream(t.Context())
	if err != nil {
		t.Fatalf("cannot get logs for pod %q: %s", podName, err)
	}

	t.Cleanup(func() { logsReader.Close() })

	if err := cs.CoreV1().Pods("default").Delete(t.Context(), podName, metav1.DeleteOptions{}); err != nil {
		t.Fatalf("cannot delete pod %q: %s", podName, err)
	}

	logLines := countReaderLines(t, logsReader)

	return logLines
}

func countEventsInES(t *testing.T, index string, size int) int {
	return len(integration.GetEventsMsgFromES(t, index, size))
}

func countReaderLines(t *testing.T, r io.Reader) int {
	t.Helper()
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	n := 0
	for s.Scan() {
		n++
	}
	if err := s.Err(); err != nil {
		t.Fatalf("cannot scan logs: %s", err)
	}
	return n
}
