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

//go:build integration && !requirefips

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/kind/pkg/cluster"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
)

func TestHintsKubernetesInputAllowList(t *testing.T) {
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)

	kubeConfigPath, _ := createKindCluster(t, filebeat.TempDir())
	nodeName, _, filestreamContainerID := startFlogKubernetes(t, kubeConfigPath)
	startFlogKubernetesWithAnnotations(
		t,
		kubeConfigPath,
		map[string]string{
			"co.elastic.logs/raw": `[{"type":"httpjson","id":"disallowed-httpjson"}]`,
		},
	)

	cfgYAML := getConfig(
		t,
		map[string]any{
			"kubeConfig": kubeConfigPath,
			"nodeName":   nodeName,
		},
		"autodiscover",
		"k8s-input-allow-list.yml")
	filebeat.WriteConfigFile(cfgYAML)
	filebeat.Start()

	const (
		rejectionLogPrefix = "Rejecting autodiscover hints configuration."
		rejectionMessage   = `Rejecting autodiscover hints configuration. ` +
			`input.type: httpjson, reason: disallowed input type, allowed ` +
			`inputs are: log filestream container`
	)
	filestreamStart := fmt.Sprintf(
		`"message":"Input 'filestream' starting","service.name":"filebeat","id":"kubernetes-container-logs-%s"`,
		filestreamContainerID,
	)

	filebeat.WaitLogsContainsAnyOrder(
		[]string{filestreamStart, rejectionLogPrefix},
		30*time.Second,
		"default filestream input did not start or httpjson raw hints configuration was not rejected",
	)

	warningLine := filebeat.GetLogLine(rejectionLogPrefix)
	require.NotEmpty(t, warningLine, "rejection warning log line should be available from the beginning of the logs")

	var warning map[string]any
	require.NoError(t, json.Unmarshal([]byte(warningLine), &warning), "log entries must be valid JSON")
	assert.Equal(t, "warn", warning["log.level"], "rejection should be logged at warning level")
	assert.Equal(t, rejectionMessage, warning["message"], "rejection warning should describe the rejected input")

	filebeat.Stop()
	assert.Empty(
		t,
		filebeat.GetLogLine("Input 'httpjson' starting"),
		"the rejected httpjson input should not start",
	)
	assert.Empty(
		t,
		filebeat.GetLogLine("Auto discover config check failed for config"),
		"the rejected configuration should be filtered before CheckConfig",
	)
}

func createKindCluster(t *testing.T, workDir string) (kubeConfigPath, clusterName string) {
	t.Helper()

	clusterName = fmt.Sprintf("test-cluster-%s", uuid.Must(uuid.NewV4()))
	provider := cluster.NewProvider()
	if err := provider.Create(clusterName, cluster.CreateWithWaitForReady(30*time.Second)); err != nil {
		t.Fatalf("could not create cluster: %s", err)
	}
	t.Cleanup(func() {
		if err := provider.Delete(clusterName, ""); err != nil {
			t.Logf("could not delete Kubernetes cluster: %s", err)
		}
	})

	var kubeConfig string
	require.Eventually(t, func() bool {
		var err error
		kubeConfig, err = provider.KubeConfig(clusterName, false)
		return err == nil
	}, 30*time.Second, 100*time.Millisecond, "could not get Kubernetes configuration")

	kubeConfigPath = filepath.Join(workDir, "kube-config")
	if err := os.WriteFile(kubeConfigPath, []byte(kubeConfig), 0o666); err != nil {
		t.Fatalf("cannot write Kubernetes config file: %s", err)
	}

	return kubeConfigPath, clusterName
}

func startFlogKubernetes(t *testing.T, kubeConfigPath string) (nodeName, podName, containerID string) {
	t.Helper()
	return startFlogKubernetesWithAnnotations(t, kubeConfigPath, nil)
}

func startFlogKubernetesWithAnnotations(
	t *testing.T,
	kubeConfigPath string,
	annotations map[string]string,
) (nodeName, podName, containerID string) {
	t.Helper()

	clientset := newK8sClientsetFromKubeConfigPath(t, kubeConfigPath)
	podName = "flog-pod-" + uuid.Must(uuid.NewV4()).String()
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        podName,
			Namespace:   "default",
			Annotations: annotations,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:  "flog",
				Image: "mingrammer/flog",
				Args:  []string{"-s", "0.2", "-d", "0.2", "-l"},
			}},
		},
	}

	pod, err := clientset.CoreV1().Pods("default").Create(context.Background(), pod, metav1.CreateOptions{})
	require.NoError(t, err, "could not create Kubernetes pod")
	t.Cleanup(func() {
		_, err := clientset.CoreV1().Pods("default").Get(context.Background(), pod.Name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return
		}
		if err := clientset.CoreV1().Pods("default").Delete(context.Background(), pod.Name, metav1.DeleteOptions{}); err != nil {
			t.Logf("could not remove Kubernetes pod: %s", err)
		}
	})

	require.Eventually(t, func() bool {
		pod, err = clientset.CoreV1().Pods("default").Get(t.Context(), pod.Name, metav1.GetOptions{})
		if err != nil {
			return false
		}
		if pod.Status.Phase != corev1.PodRunning || len(pod.Status.ContainerStatuses) == 0 {
			return false
		}

		containerID = pod.Status.ContainerStatuses[0].ContainerID
		nodeName = pod.Spec.NodeName
		if separator := strings.Index(containerID, "://"); separator >= 0 {
			containerID = containerID[separator+3:]
		}
		return containerID != ""
	}, 60*time.Second, 100*time.Millisecond, "pod did not start within timeout")

	return nodeName, podName, containerID
}

func newK8sClientsetFromKubeConfigPath(t *testing.T, kubeConfigPath string) *kubernetes.Clientset {
	t.Helper()

	data, err := os.ReadFile(kubeConfigPath)
	require.NoError(t, err, "cannot read Kubernetes config")
	config, err := clientcmd.RESTConfigFromKubeConfig(data)
	require.NoError(t, err, "cannot build Kubernetes REST config")
	clientset, err := kubernetes.NewForConfig(config)
	require.NoError(t, err, "cannot create Kubernetes clientset")
	return clientset
}
