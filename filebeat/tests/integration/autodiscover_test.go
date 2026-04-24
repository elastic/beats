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
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/kind/pkg/apis/config/v1alpha4"
	"sigs.k8s.io/kind/pkg/cluster"
	"sigs.k8s.io/kind/pkg/cluster/nodeutils"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
	"github.com/elastic/beats/v7/libbeat/version"
	"github.com/elastic/elastic-agent-autodiscover/docker"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/testing/fs"
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

	kubeConfigPath, _ := createKindCluster(t, filebeat.TempDir())
	noneName, _, containerID := startFlogKubernetes(t, kubeConfigPath)

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
	filebeatImage := "docker.elastic.co/beats/filebeat-oss-wolfi" + ":" + version.GetDefaultVersion() + "-SNAPSHOT"

	workDir := fs.TempDir(t, "..", "..", "build", "integration-tests")

	kubeConfigPath, clusterName := createKindCluster(t, workDir,
		cluster.CreateWithV1Alpha4Config(&v1alpha4.Cluster{
			Nodes: []v1alpha4.Node{
				{
					Role: v1alpha4.ControlPlaneRole,
					ExtraMounts: []v1alpha4.Mount{
						{
							HostPath:      workDir,
							ContainerPath: workDir,
						},
					},
				},
			},
		}))
	nodeName, podName, _ := startFlogKubernetes(t, kubeConfigPath)

	loadDockerImageIntoKind(t, clusterName, filebeatImage)
	grantClusterAdminToDefaultServiceAccount(t, kubeConfigPath)

	filebeatPodName := "filebeat-pod-" + uuid.Must(uuid.NewV4()).String()
	logInputConfigPath := filepath.Join(workDir, "filebeat-log.yml")
	filestreamInputConfigPath := filepath.Join(workDir, "filebeat-filestream.yml")

	esURL := integration.GetESAdminURL(t, "http")
	esHost := fmt.Sprintf("%s://%s", esURL.Scheme, esURL.Host)
	if esURL.Hostname() == "localhost" || esURL.Hostname() == "127.0.0.1" {
		esHost = fmt.Sprintf("%s://%s:%s", esURL.Scheme, kindNodeGatewayIP(t, nodeName), esURL.Port())
	}
	esUser := esURL.User.Username()
	esPass, _ := esURL.User.Password()

	index := fmt.Sprintf("test-autodiscover-take-over-%s", uuid.Must(uuid.NewV4()).String())
	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("Elasticsearch index used: %q", index)
		}
	})

	tmplVars := map[string]any{
		"nodeName": nodeName,
		"podName":  podName,
		"esHost":   esHost,
		"esUser":   esUser,
		"esPass":   esPass,
		"index":    index,
	}

	writeFile(
		t,
		logInputConfigPath,
		getConfig(t, tmplVars, "autodiscover", "take-over-log-input-k8s.yml"),
	)

	startFilebeatPodForTakeOver(
		t,
		kubeConfigPath,
		nodeName,
		filebeatPodName,
		filebeatImage,
		workDir,
		logInputConfigPath,
	)

	// Wait until at least 5 events are ingested
	require.Eventually(
		t,
		func() bool { return countEventsInES(t, index, 1000) >= 5 },
		30*time.Second,
		200*time.Millisecond,
		"did not ingest the initial events")

	deletePodK8s(t, kubeConfigPath, filebeatPodName)
	logInputIngested := countEventsInES(t, index, 1000)

	// Re-Start Filebeat with Filestream and take_over enabled
	writeFile(
		t,
		filestreamInputConfigPath,
		getConfig(t, tmplVars, "autodiscover", "take-over-filestream-input-k8s.yml"),
	)

	startFilebeatPodForTakeOver(
		t,
		kubeConfigPath,
		nodeName,
		filebeatPodName,
		filebeatImage,
		workDir,
		filestreamInputConfigPath,
	)

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
	waitFilebeatLogContains(
		t,
		workDir,
		"File is inactive. Closing.",
		20*time.Second,
	)

	// Wait until at least all published events can be found in Elasticsearch
	require.Eventuallyf(
		t,
		func() bool { return countEventsInES(t, index, generatedEvents+100) >= generatedEvents },
		30*time.Second,
		200*time.Millisecond,
		"did not ingest all generated events from pod %s", podName,
	)

	deletePodK8s(t, kubeConfigPath, filebeatPodName)

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

func createKindCluster(
	t *testing.T,
	workDir string,
	options ...cluster.CreateOption,
) (kubeConfigPath, clusterName string) {

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

	clusterName = fmt.Sprintf("test-cluster-%s", uid)
	err := provider.Create(
		clusterName,
		append(
			[]cluster.CreateOption{cluster.CreateWithWaitForReady(30 * time.Second)},
			options...)...,
	)
	if err != nil {
		t.Fatalf("could not create cluster: %s", err)
	}
	t.Cleanup(func() {
		if err := provider.Delete(clusterName, ""); err != nil {
			t.Logf("could not delete K8s cluster: %s", err)
		}
	})

	var kubeConfig string
	require.Eventually(t, func() bool {
		kubeConfig, err = provider.KubeConfig(clusterName, false)
		return err == nil
	}, 30*time.Second, 100*time.Millisecond, "could not get kube config")

	kubeConfigPath = filepath.Join(workDir, "kube-config")
	if err = os.WriteFile(kubeConfigPath, []byte(kubeConfig), 0666); err != nil {
		t.Fatalf("cannot write kube config file: %s", err)
	}

	return kubeConfigPath, clusterName
}

func startFlogKubernetes(t *testing.T, kubeConfigPath string) (nodeName, podName, containerID string) {
	clientset := newK8sClientsetFromKubeConfigPath(t, kubeConfigPath)

	podName = "flog-pod-" + uuid.Must(uuid.NewV4()).String()
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
					Args:  []string{"-s", "0.2", "-d", "0.2", "-l"},
				},
			},
		},
	}

	pod, err := clientset.CoreV1().Pods("default").Create(context.Background(), pod, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("could not create pod: %s", err)
	}

	t.Cleanup(func() {
		_, err := clientset.CoreV1().Pods("default").Get(context.Background(), pod.Name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			// The pod has already been removed, return
			return
		}

		// by the time Cleanup runs, t.Context has been cancelled, so we need a new context here
		err = clientset.CoreV1().Pods("default").Delete(context.Background(), pod.Name, metav1.DeleteOptions{})
		if err != nil {
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

			if pod.Status.Phase == corev1.PodRunning && len(pod.Status.ContainerStatuses) > 0 {
				containerID = pod.Status.ContainerStatuses[0].ContainerID
				if containerID != "" {
					nodeName = pod.Spec.NodeName
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

	return nodeName, podName, containerID
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
	reader, err := cli.ImagePull(ctx, img, client.ImagePullOptions{})
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
		client.ContainerCreateOptions{
			Config: &container.Config{
				Image: img,
				Cmd:   []string{"-l", "-d", "1", "-s", "1"},
			},
		})
	if err != nil {
		t.Fatalf("cannot create container for %q: %s", img, err)
	}

	_, err = cli.ContainerStart(ctx, resp.ID, client.ContainerStartOptions{})
	if err != nil {
		t.Fatalf("cannot start container: %s", err)
	}

	t.Cleanup(func() {
		ctx := context.Background()
		if _, err := cli.ContainerStop(ctx, resp.ID, client.ContainerStopOptions{}); err != nil {
			t.Errorf("cannot stop container: %s", err)
		}
		if _, err := cli.ContainerRemove(ctx, resp.ID, client.ContainerRemoveOptions{}); err != nil {
			t.Errorf("cannot remove container: %s", err)
		}
	})
	return resp.ID
}

func loadDockerImageIntoKind(t *testing.T, clusterName, imageName string) {
	provider := cluster.NewProvider()
	nodes, err := provider.ListInternalNodes(clusterName)
	if err != nil {
		t.Fatalf(
			"cannot list nodes for kind cluster %q: %s",
			clusterName,
			err,
		)
	}
	if len(nodes) == 0 {
		t.Fatalf("no nodes found for kind cluster %q", clusterName)
	}

	cli, err := docker.NewClient(client.DefaultDockerHost, nil, nil, logp.NewNopLogger())
	if err != nil {
		t.Fatalf("cannot create Docker client: %s", err)
	}

	for _, node := range nodes {
		imageReader, err := cli.ImageSave(t.Context(), []string{imageName})
		if err != nil {
			t.Fatalf("cannot save image %q from local docker daemon: %s", imageName, err)
		}

		if err := nodeutils.LoadImageArchive(node, imageReader); err != nil {
			_ = imageReader.Close()
			t.Fatalf(
				"cannot load image %q into kind node %q: %s",
				imageName,
				node.String(),
				err,
			)
		}

		if err := imageReader.Close(); err != nil {
			t.Fatalf(
				"cannot close image stream for image %q on node %q: %s",
				imageName,
				node.String(),
				err,
			)
		}
	}
}

func kindNodeGatewayIP(t *testing.T, nodeName string) string {
	cli, err := docker.NewClient(client.DefaultDockerHost, nil, nil, logp.NewNopLogger())
	if err != nil {
		t.Fatalf("cannot create Docker client: %s", err)
	}

	inspectResult, err := cli.ContainerInspect(context.Background(), nodeName, client.ContainerInspectOptions{})
	if err != nil {
		t.Fatalf("cannot inspect Kind node %q: %s", nodeName, err)
	}

	if inspectResult.Container.NetworkSettings != nil {
		for _, networkSettings := range inspectResult.Container.NetworkSettings.Networks {
			if networkSettings != nil && networkSettings.Gateway != "" {
				return networkSettings.Gateway
			}
		}
	}

	t.Fatalf("cannot determine gateway IP for Kind node %q", nodeName)
	return ""
}

func grantClusterAdminToDefaultServiceAccount(t *testing.T, kubeConfigPath string) {
	cs := newK8sClientsetFromKubeConfigPath(t, kubeConfigPath)

	bindingName := "filebeat-autodiscover-admin-" + uuid.Must(uuid.NewV4()).String()
	_, err := cs.RbacV1().ClusterRoleBindings().Create(t.Context(), &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: bindingName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "default",
				Namespace: "default",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "cluster-admin",
		},
	}, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("cannot create cluster role binding for default service account: %s", err)
	}
}

func writeFile(t *testing.T, path, content string) {
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("cannot write file %q: %s", path, err)
	}
}

func startFilebeatPodForTakeOver(
	t *testing.T,
	kubeConfigPath,
	nodeName,
	podName,
	imageName,
	workDir,
	configPath string,
) {

	cs := newK8sClientsetFromKubeConfigPath(t, kubeConfigPath)
	hostPathDir := corev1.HostPathDirectory

	user, err := user.Current()
	if err != nil {
		t.Fatalf("cannot get current user: %s", err)
	}
	udi, err := strconv.Atoi(user.Uid)
	if err != nil {
		t.Fatalf("cannot convert UID from string to integer: %s", err)
	}

	// Small hack to make sure the test can read the files generated by the
	// container (Filebeat): run Filebeat (the container) with the same UID
	// as the current user. That is required for CI
	uid := int64(udi)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			NodeName:      nodeName,
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				{
					Name:            "filebeat",
					Image:           imageName,
					ImagePullPolicy: corev1.PullNever,
					Args: []string{
						"--strict.perms=false",
						"-c", configPath,
						"-E", fmt.Sprintf("path.home=%s", workDir),
					},
					SecurityContext: &corev1.SecurityContext{
						RunAsUser: &uid,
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "home-folder",
							MountPath: workDir,
						},
						{
							Name:      "varlogcontainers",
							MountPath: "/var/log/containers",
							ReadOnly:  true,
						},
						{
							Name:      "varlogpods",
							MountPath: "/var/log/pods",
							ReadOnly:  true,
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "home-folder",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: workDir,
							Type: &hostPathDir,
						},
					},
				},
				{
					Name: "varlogcontainers",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/var/log/containers",
							Type: &hostPathDir,
						},
					},
				},
				{
					Name: "varlogpods",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/var/log/pods",
							Type: &hostPathDir,
						},
					},
				},
			},
		},
	}

	if _, err := cs.CoreV1().Pods("default").Create(t.Context(), pod, metav1.CreateOptions{}); err != nil {
		t.Fatalf("could not create filebeat pod: %s", err)
	}

	require.Eventuallyf(
		t,
		func() bool {
			pod, err := cs.CoreV1().Pods("default").Get(t.Context(), podName, metav1.GetOptions{})
			if err != nil {
				return false
			}
			if pod.Status.Phase == corev1.PodFailed {
				t.Logf("filebeat pod failed: %v", pod.Status)
				return false
			}
			return pod.Status.Phase == corev1.PodRunning
		},
		60*time.Second,
		200*time.Millisecond,
		"filebeat pod %q did not start", podName,
	)
}

func waitFilebeatLogContains(t *testing.T, workDir, msg string, timeout time.Duration) {
	t.Helper()
	// Glob to match the date, it will stop working in about 1000 years
	paths, err := filepath.Glob(filepath.Join(workDir, "logs", "filebeat-2*.ndjson"))
	if err != nil {
		t.Fatalf("cannot resolve glob for log files: %s", err)
	}

	if len(paths) != 1 {
		t.Fatalf("There must be a single log file for Filebeat, found %d", len(paths))
	}

	f, err := os.Open(paths[0])
	if err != nil {
		t.Fatalf("cannot open Filebeat log file: %s", err)
	}
	defer f.Close()

	logFile := fs.LogFile{File: f}
	logFile.WaitLogsContains(t, msg, timeout, "Filebeat logs did not contain '%s'", msg)
}

func deletePodK8s(t *testing.T, kubeConfigPath, podName string) {
	cs := newK8sClientsetFromKubeConfigPath(t, kubeConfigPath)

	err := cs.CoreV1().Pods("default").Delete(context.Background(), podName, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		t.Fatalf("cannot delete pod %q: %s", podName, err)
	}

	require.Eventuallyf(
		t,
		func() bool {
			_, err := cs.CoreV1().Pods("default").Get(context.Background(), podName, metav1.GetOptions{})
			return apierrors.IsNotFound(err)
		},
		30*time.Second,
		100*time.Millisecond,
		"pod %q was not deleted", podName,
	)
}

// stopPodK8sAndCountLogs returns the number of log lines generated by the pod
// and then deletes it.
func stopPodK8sAndCountLogs(t *testing.T, kubeConfigPath, podName string) int {
	cs := newK8sClientsetFromKubeConfigPath(t, kubeConfigPath)

	logsReader, err := cs.CoreV1().Pods("default").GetLogs(podName, &corev1.PodLogOptions{
		Container: "flog",
		Follow:    true,
	}).Stream(t.Context())
	if err != nil {
		t.Fatalf("cannot get logs for pod %q: %s", podName, err)
	}
	defer logsReader.Close()

	if err := cs.CoreV1().Pods("default").Delete(context.Background(), podName, metav1.DeleteOptions{}); err != nil {
		t.Fatalf("cannot delete pod %q: %s", podName, err)
	}

	logLines := countReaderLines(t, logsReader)

	return logLines
}

func newK8sClientsetFromKubeConfigPath(t *testing.T, kubeConfigPath string) *kubernetes.Clientset {
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
	return cs
}

func countEventsInES(t *testing.T, index string, size int) int {
	return len(integration.GetEventsMsgFromES(t, index, size))
}

func countReaderLines(t *testing.T, r io.Reader) int {
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
