// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License 2.0;
// you may not use this file except in compliance with the Elastic License 2.0.

//go:build integration

package integration

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/go-elasticsearch/v8"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/e2e-framework/klient"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/kyaml/filesys"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"

	aclient "github.com/elastic/elastic-agent/pkg/control/v2/client"
	atesting "github.com/elastic/elastic-agent/pkg/testing"
	"github.com/elastic/elastic-agent/pkg/testing/define"
	"github.com/elastic/elastic-agent/pkg/testing/tools/fleettools"
)

const (
	agentK8SKustomize = "../../deploy/kubernetes/elastic-agent-kustomize/default/elastic-agent-standalone"
	agentK8SHelm      = "../../deploy/helm/elastic-agent"
)

var noSpecialCharsRegexp = regexp.MustCompile("[^a-zA-Z0-9]+")

func TestKubernetesAgentStandaloneKustomize(t *testing.T) {
	info := define.Require(t, define.Requirements{
		Stack: &define.Stack{},
		Local: false,
		Sudo:  false,
		OS: []define.OS{
			// only test the basic and the wolfi container with otel
			{Type: define.Kubernetes, DockerVariant: "basic"},
			{Type: define.Kubernetes, DockerVariant: "wolfi"},
		},
		Group: define.Kubernetes,
	})

	agentImage := os.Getenv("AGENT_IMAGE")
	require.NotEmpty(t, agentImage, "AGENT_IMAGE must be set")

	client, err := info.KubeClient()
	require.NoError(t, err)
	require.NotNil(t, client)

	testLogsBasePath := os.Getenv("K8S_TESTS_POD_LOGS_BASE")
	require.NotEmpty(t, testLogsBasePath, "K8S_TESTS_POD_LOGS_BASE must be set")

	err = os.MkdirAll(filepath.Join(testLogsBasePath, t.Name()), 0755)
	require.NoError(t, err, "failed to create test logs directory")

	namespace := info.Namespace

	esHost := os.Getenv("ELASTICSEARCH_HOST")
	require.NotEmpty(t, esHost, "ELASTICSEARCH_HOST must be set")

	esAPIKey, err := generateESAPIKey(info.ESClient, namespace)
	require.NoError(t, err, "failed to generate ES API key")
	require.NotEmpty(t, esAPIKey, "failed to generate ES API key")

	renderedManifest, err := renderKustomize(agentK8SKustomize)
	require.NoError(t, err, "failed to render kustomize")

	testCases := []struct {
		name             string
		runUser          *int64
		runGroup         *int64
		capabilitiesDrop []corev1.Capability
		capabilitiesAdd  []corev1.Capability
		runK8SInnerTests bool
		skipReason       string
	}{
		{
			"default deployment - rootful agent",
			nil,
			nil,
			nil,
			nil,
			false,
			"",
		},
		{
			"drop ALL capabilities - rootful agent",
			int64Ptr(0),
			nil,
			[]corev1.Capability{"ALL"},
			[]corev1.Capability{},
			false,
			"",
		},
		{
			"drop ALL add CHOWN, SETPCAP capabilities - rootful agent",
			int64Ptr(0),
			nil,
			[]corev1.Capability{"ALL"},
			[]corev1.Capability{"CHOWN", "SETPCAP"},
			true,
			"",
		},
		{
			"drop ALL add CHOWN, SETPCAP capabilities - rootless agent",
			int64Ptr(1000), // elastic-agent uid
			nil,
			[]corev1.Capability{"ALL"},
			[]corev1.Capability{"CHOWN", "SETPCAP", "DAC_READ_SEARCH", "SYS_PTRACE"},
			true,
			"",
		},
		{
			"drop ALL add CHOWN, SETPCAP capabilities - rootless agent random uid:gid",
			int64Ptr(500),
			int64Ptr(500),
			[]corev1.Capability{"ALL"},
			[]corev1.Capability{"CHOWN", "SETPCAP", "DAC_READ_SEARCH", "SYS_PTRACE"},
			true,
			"",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if tc.skipReason != "" {
				t.Skip(tc.skipReason)
			}

			hasher := sha256.New()
			hasher.Write([]byte(tc.name))
			testNamespace := strings.ToLower(base64.URLEncoding.EncodeToString(hasher.Sum(nil)))
			testNamespace = noSpecialCharsRegexp.ReplaceAllString(testNamespace, "")

			k8sObjects, err := yamlToK8SObjects(bufio.NewReader(bytes.NewReader(renderedManifest)))
			require.NoError(t, err, "failed to convert yaml to k8s objects")

			adjustK8SAgentManifests(k8sObjects, testNamespace, "elastic-agent-standalone",
				func(container *corev1.Container) {
					// set agent image
					container.Image = agentImage
					// set ImagePullPolicy to "Never" to avoid pulling the image
					// as the image is already loaded by the kubernetes provisioner
					container.ImagePullPolicy = "Never"

					container.Resources.Limits = corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("800Mi"),
					}

					if tc.capabilitiesDrop != nil || tc.capabilitiesAdd != nil || tc.runUser != nil || tc.runGroup != nil {
						// set security context
						container.SecurityContext = &corev1.SecurityContext{
							Capabilities: &corev1.Capabilities{
								Drop: tc.capabilitiesDrop,
								Add:  tc.capabilitiesAdd,
							},
							RunAsUser:  tc.runUser,
							RunAsGroup: tc.runGroup,
						}

					}
					// set Elasticsearch host and API key
					for idx, env := range container.Env {
						if env.Name == "ES_HOST" {
							container.Env[idx].Value = esHost
							container.Env[idx].ValueFrom = nil
						}
						if env.Name == "API_KEY" {
							container.Env[idx].Value = esAPIKey
							container.Env[idx].ValueFrom = nil
						}
					}
				},
				func(pod *corev1.PodSpec) {
					for volumeIdx, volume := range pod.Volumes {
						// need to update the volume path of the state directory
						// to match the test namespace
						if volume.Name == "elastic-agent-state" {
							hostPathType := corev1.HostPathDirectoryOrCreate
							pod.Volumes[volumeIdx].VolumeSource.HostPath = &corev1.HostPathVolumeSource{
								Type: &hostPathType,
								Path: fmt.Sprintf("/var/lib/elastic-agent-standalone/%s/state", testNamespace),
							}
						}
					}
				})

			ctx := context.Background()

			deployK8SAgent(t, ctx, client, k8sObjects, testNamespace, tc.runK8SInnerTests, testLogsBasePath, nil)
		})
	}

}

func TestKubernetesAgentOtel(t *testing.T) {
	info := define.Require(t, define.Requirements{
		Stack: &define.Stack{},
		Local: false,
		Sudo:  false,
		OS: []define.OS{
			// only test the basic and the wolfi container with otel
			{Type: define.Kubernetes, DockerVariant: "basic"},
			{Type: define.Kubernetes, DockerVariant: "wolfi"},
		},
		Group: define.Kubernetes,
	})

	agentImage := os.Getenv("AGENT_IMAGE")
	require.NotEmpty(t, agentImage, "AGENT_IMAGE must be set")

	client, err := info.KubeClient()
	require.NoError(t, err)
	require.NotNil(t, client)

	testLogsBasePath := os.Getenv("K8S_TESTS_POD_LOGS_BASE")
	require.NotEmpty(t, testLogsBasePath, "K8S_TESTS_POD_LOGS_BASE must be set")

	err = os.MkdirAll(filepath.Join(testLogsBasePath, t.Name()), 0755)
	require.NoError(t, err, "failed to create test logs directory")

	namespace := info.Namespace

	esHost := os.Getenv("ELASTICSEARCH_HOST")
	require.NotEmpty(t, esHost, "ELASTICSEARCH_HOST must be set")

	esAPIKey, err := generateESAPIKey(info.ESClient, namespace)
	require.NoError(t, err, "failed to generate ES API key")
	require.NotEmpty(t, esAPIKey, "failed to generate ES API key")

	renderedManifest, err := renderKustomize(agentK8SKustomize)
	require.NoError(t, err, "failed to render kustomize")

	testCases := []struct {
		name              string
		envAdd            []corev1.EnvVar
		runK8SInnerTests  bool
		componentPresence map[string]bool
	}{

		{
			"run agent in otel mode",
			[]corev1.EnvVar{
				{Name: "ELASTIC_AGENT_OTEL", Value: "true"},
			},
			false,
			map[string]bool{
				"beat/metrics-monitoring": false,
				"filestream-monitoring":   false,
				"system/metrics-default":  false,
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			hasher := sha256.New()
			hasher.Write([]byte(tc.name))
			testNamespace := strings.ToLower(base64.URLEncoding.EncodeToString(hasher.Sum(nil)))
			testNamespace = noSpecialCharsRegexp.ReplaceAllString(testNamespace, "")

			k8sObjects, err := yamlToK8SObjects(bufio.NewReader(bytes.NewReader(renderedManifest)))
			require.NoError(t, err, "failed to convert yaml to k8s objects")

			adjustK8SAgentManifests(k8sObjects, testNamespace, "elastic-agent-standalone",
				func(container *corev1.Container) {
					// set agent image
					container.Image = agentImage
					// set ImagePullPolicy to "Never" to avoid pulling the image
					// as the image is already loaded by the kubernetes provisioner
					container.ImagePullPolicy = "Never"

					// set Elasticsearch host and API key
					for idx, env := range container.Env {
						if env.Name == "ES_HOST" {
							container.Env[idx].Value = esHost
							container.Env[idx].ValueFrom = nil
						}
						if env.Name == "API_KEY" {
							container.Env[idx].Value = esAPIKey
							container.Env[idx].ValueFrom = nil
						}
					}

					if len(tc.envAdd) > 0 {
						container.Env = append(container.Env, tc.envAdd...)
					}

					// drop arguments overriding default config
					container.Args = []string{}
				},
				func(pod *corev1.PodSpec) {
					for volumeIdx, volume := range pod.Volumes {
						// need to update the volume path of the state directory
						// to match the test namespace
						if volume.Name == "elastic-agent-state" {
							hostPathType := corev1.HostPathDirectoryOrCreate
							pod.Volumes[volumeIdx].VolumeSource.HostPath = &corev1.HostPathVolumeSource{
								Type: &hostPathType,
								Path: fmt.Sprintf("/var/lib/elastic-agent-standalone/%s/state", testNamespace),
							}
						}
					}
				})

			ctx := context.Background()

			deployK8SAgent(t, ctx, client, k8sObjects, testNamespace, tc.runK8SInnerTests, testLogsBasePath, tc.componentPresence)
		})
	}
}

func TestKubernetesAgentHelm(t *testing.T) {
	info := define.Require(t, define.Requirements{
		Stack: &define.Stack{},
		Local: false,
		Sudo:  false,
		OS: []define.OS{
			// only test the basic and the wolfi container with otel
			{Type: define.Kubernetes, DockerVariant: "basic"},
			{Type: define.Kubernetes, DockerVariant: "wolfi"},
		},
		Group: define.Kubernetes,
	})

	agentImage := os.Getenv("AGENT_IMAGE")
	require.NotEmpty(t, agentImage, "AGENT_IMAGE must be set")

	agentImageParts := strings.SplitN(agentImage, ":", 2)
	require.Len(t, agentImageParts, 2, "AGENT_IMAGE must be in the form '<repository>:<version>'")
	agentImageRepo := agentImageParts[0]
	agentImageTag := agentImageParts[1]

	client, err := info.KubeClient()
	require.NoError(t, err)
	require.NotNil(t, client)

	testLogsBasePath := os.Getenv("K8S_TESTS_POD_LOGS_BASE")
	require.NotEmpty(t, testLogsBasePath, "K8S_TESTS_POD_LOGS_BASE must be set")

	err = os.MkdirAll(filepath.Join(testLogsBasePath, t.Name()), 0755)
	require.NoError(t, err, "failed to create test logs directory")

	namespace := info.Namespace

	esHost := os.Getenv("ELASTICSEARCH_HOST")
	require.NotEmpty(t, esHost, "ELASTICSEARCH_HOST must be set")

	esAPIKey, err := generateESAPIKey(info.ESClient, namespace)
	require.NoError(t, err, "failed to generate ES API key")
	require.NotEmpty(t, esAPIKey, "failed to generate ES API key")

	enrollParams, err := fleettools.NewEnrollParams(context.Background(), info.KibanaClient)
	require.NoError(t, err, "failed to create fleet enroll params")

	testCases := []struct {
		name                       string
		values                     map[string]any
		atLeastValidatedPodsNumber int
		runK8SInnerTests           bool
	}{
		{
			name: "helm standalone agent default kubernetes privileged",
			values: map[string]any{
				"kubernetes": map[string]any{
					"enabled": true,
				},
				"agent": map[string]any{
					"unprivileged": false,
					"image": map[string]any{
						"repository": agentImageRepo,
						"tag":        agentImageTag,
						"pullPolicy": "Never",
					},
				},
				"outputs": map[string]any{
					"default": map[string]any{
						"type":    "ESPlainAuthAPI",
						"url":     esHost,
						"api_key": esAPIKey,
					},
				},
			},
			runK8SInnerTests: true,
			// - perNode Daemonset (at least 1 agent pod)
			// - clusterWide Deployment  (1 agent pod)
			// - ksmSharded Statefulset  (1 agent pod)
			atLeastValidatedPodsNumber: 3,
		},
		{
			name: "helm standalone agent default kubernetes unprivileged",
			values: map[string]any{
				"kubernetes": map[string]any{
					"enabled": true,
				},
				"agent": map[string]any{
					"unprivileged": true,
					"image": map[string]any{
						"repository": agentImageRepo,
						"tag":        agentImageTag,
						"pullPolicy": "Never",
					},
				},
				"outputs": map[string]any{
					"default": map[string]any{
						"type":    "ESPlainAuthAPI",
						"url":     esHost,
						"api_key": esAPIKey,
					},
				},
			},
			runK8SInnerTests: true,
			// - perNode Daemonset (at least 1 agent pod)
			// - clusterWide Deployment  (1 agent pod)
			// - ksmSharded Statefulset  (1 agent pod)
			atLeastValidatedPodsNumber: 3,
		},
		{
			name: "helm managed agent default kubernetes privileged",
			values: map[string]any{
				"agent": map[string]any{
					"unprivileged": false,
					"image": map[string]any{
						"repository": agentImageRepo,
						"tag":        agentImageTag,
						"pullPolicy": "Never",
					},
					"fleet": map[string]any{
						"enabled": true,
						"url":     enrollParams.FleetURL,
						"token":   enrollParams.EnrollmentToken,
						"preset":  "perNode",
					},
				},
			},
			runK8SInnerTests: true,
			// - perNode Daemonset (at least 1 agent pod)
			atLeastValidatedPodsNumber: 1,
		},
		{
			name: "helm managed agent default kubernetes unprivileged",
			values: map[string]any{
				"agent": map[string]any{
					"unprivileged": true,
					"image": map[string]any{
						"repository": agentImageRepo,
						"tag":        agentImageTag,
						"pullPolicy": "Never",
					},
					"fleet": map[string]any{
						"enabled": true,
						"url":     enrollParams.FleetURL,
						"token":   enrollParams.EnrollmentToken,
						"preset":  "perNode",
					},
				},
			},
			runK8SInnerTests: true,
			// - perNode Daemonset (at least 1 agent pod)
			atLeastValidatedPodsNumber: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			hasher := sha256.New()
			hasher.Write([]byte(tc.name))
			testNamespace := strings.ToLower(base64.URLEncoding.EncodeToString(hasher.Sum(nil)))
			testNamespace = noSpecialCharsRegexp.ReplaceAllString(testNamespace, "")

			settings := cli.New()
			settings.SetNamespace(testNamespace)
			actionConfig := &action.Configuration{}

			helmChart, err := loader.Load(agentK8SHelm)
			require.NoError(t, err, "failed to load helm chart")

			err = actionConfig.Init(settings.RESTClientGetter(), settings.Namespace(), "",
				func(format string, v ...interface{}) {})
			require.NoError(t, err, "failed to init helm action config")

			helmValues := tc.values

			t.Cleanup(func() {
				if t.Failed() {
					dumpLogs(t, ctx, client, testNamespace, testLogsBasePath)
				}

				uninstallAction := action.NewUninstall(actionConfig)
				uninstallAction.Wait = true

				_, err = uninstallAction.Run("helm-agent")
				if err != nil {
					require.NoError(t, err, "failed to uninstall helm chart")
				}
			})

			installAction := action.NewInstall(actionConfig)
			installAction.Namespace = testNamespace
			installAction.CreateNamespace = true
			installAction.UseReleaseName = true
			installAction.ReleaseName = "helm-agent"
			installAction.Timeout = 2 * time.Minute
			installAction.Wait = true
			installAction.WaitForJobs = true
			_, err = installAction.Run(helmChart, helmValues)
			require.NoError(t, err, "failed to install helm chart")

			podList := &corev1.PodList{}
			err = client.Resources(testNamespace).List(ctx, podList)
			require.NoError(t, err, fmt.Sprintf("failed to list pods in namespace %s", testNamespace))

			checkedAgentContainers := 0

			for _, pod := range podList.Items {
				if !strings.HasPrefix(pod.GetName(), "agent-") {
					continue
				}

				command := []string{"elastic-agent", "status"}
				var stdout, stderr bytes.Buffer
				var agentHealthyErr error
				// we will wait maximum 120 seconds for the agent to report healthy
				for i := 0; i < 120; i++ {
					stdout.Reset()
					stderr.Reset()
					agentHealthyErr = client.Resources().ExecInPod(ctx, testNamespace, pod.Name, "agent", command, &stdout, &stderr)
					if agentHealthyErr == nil {
						break
					}
					time.Sleep(time.Second * 1)
				}

				statusString := stdout.String()
				if agentHealthyErr != nil {
					t.Errorf("elastic-agent never reported healthy: %v", agentHealthyErr)
					t.Logf("stdout: %s\n", statusString)
					t.Logf("stderr: %s\n", stderr.String())
					t.FailNow()
					return
				}

				stdout.Reset()
				stderr.Reset()

				if tc.runK8SInnerTests {
					err := client.Resources().ExecInPod(ctx, testNamespace, pod.Name, "agent",
						[]string{"/usr/share/elastic-agent/k8s-inner-tests", "-test.v"}, &stdout, &stderr)
					t.Log(stdout.String())
					if err != nil {
						t.Log(stderr.String())
					}
					require.NoError(t, err, "error at k8s inner tests execution")
				}

				checkedAgentContainers++
			}

			require.GreaterOrEqual(t, checkedAgentContainers, tc.atLeastValidatedPodsNumber,
				fmt.Sprintf("at least %d agent containers should be checked", tc.atLeastValidatedPodsNumber))
		})
	}
}

// deployK8SAgent is a helper function to deploy the elastic-agent in k8s and invoke the inner k8s tests if
// runK8SInnerTests is true
func deployK8SAgent(t *testing.T, ctx context.Context, client klient.Client, objects []k8s.Object, namespace string,
	runInnerK8STests bool, testLogsBasePath string, componentPresence map[string]bool) {

	objects = append([]k8s.Object{&corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}}, objects...)

	t.Cleanup(func() {
		if t.Failed() {
			dumpLogs(t, ctx, client, namespace, testLogsBasePath)
		}

		// need to delete all k8s objects and wait for it as elastic-agent
		// in k8s creates cluster-wide roles and having multiple of them at
		// the same time isn't allowed
		deleteK8SObjects(t, ctx, client, objects, true)
	})

	// Create the objects
	for _, obj := range objects {
		obj.SetNamespace(namespace)
		err := client.Resources(namespace).Create(ctx, obj)
		require.NoError(t, err, fmt.Sprintf("failed to create object %s", obj.GetName()))
	}

	var agentPodName string
	// Wait for pods to be ready
	require.Eventually(t, func() bool {
		podList := &corev1.PodList{}
		err := client.Resources(namespace).List(ctx, podList)
		require.NoError(t, err, fmt.Sprintf("failed to list pods in namespace %s", namespace))

		for _, pod := range podList.Items {
			if agentPodName == "" && strings.HasPrefix(pod.GetName(), "elastic-agent-standalone") {
				agentPodName = pod.Name
			}

			for _, containerStatus := range pod.Status.ContainerStatuses {
				if containerStatus.RestartCount > 0 {
					return false
				}
			}

			for _, cond := range pod.Status.Conditions {
				if cond.Type != corev1.PodReady {
					continue
				}

				if cond.Status != corev1.ConditionTrue {
					return false
				}
			}
		}

		return true
	}, time.Second*100, time.Second*1, fmt.Sprintf("pods in namespace %s never became ready", namespace))

	require.NotEmpty(t, agentPodName, "agent pod name is empty")

	command := []string{"elastic-agent", "status", "--output=json"}
	var status atesting.AgentStatusOutput
	var stdout, stderr bytes.Buffer
	var agentHealthyErr error
	// we will wait maximum 120 seconds for the agent to report healthy
	for i := 0; i < 120; i++ {
		status = atesting.AgentStatusOutput{} // clear status output
		stdout.Reset()
		stderr.Reset()
		agentHealthyErr = client.Resources().ExecInPod(ctx, namespace, agentPodName, "elastic-agent-standalone", command, &stdout, &stderr)
		if agentHealthyErr == nil {
			if uerr := json.Unmarshal(stdout.Bytes(), &status); uerr == nil {
				if status.State == int(aclient.Healthy) {
					// agent is healthy innner tests should now pass
					if runInnerK8STests {
						err := client.Resources().ExecInPod(ctx, namespace, agentPodName, "elastic-agent-standalone",
							[]string{"/usr/share/elastic-agent/k8s-inner-tests", "-test.v"}, &stdout, &stderr)
						t.Log(stdout.String())
						if err != nil {
							t.Log(stderr.String())
						}
						require.NoError(t, err, "error at k8s inner tests execution")
					}

					// validate that the components defined are also healthy if they should exist
					componentsCorrect := true
					for component, shouldBePresent := range componentPresence {
						compState, ok := getComponentState(status, component)
						if shouldBePresent {
							if !ok {
								// doesn't exist
								componentsCorrect = false
							} else if compState != int(aclient.Healthy) {
								// not healthy
								componentsCorrect = false
							}
						} else if ok {
							// should not be present
							// break instantly and fail (as it existing should never happen)
							break
						}
					}
					if componentsCorrect {
						// agent health and components are correct
						return
					}
				}
			}
		}
		time.Sleep(time.Second * 1)
	}

	t.Errorf("elastic-agent never reported healthy: %+v", status)
	t.Logf("stdout: %s\n", stdout.String())
	t.Logf("stderr: %s\n", stderr.String())
	t.FailNow()
}

func getComponentState(status atesting.AgentStatusOutput, componentName string) (int, bool) {
	for _, comp := range status.Components {
		if comp.Name == componentName {
			return comp.State, true
		}
	}
	return -1, false
}

// dumpLogs dumps the logs of all pods in the given namespace to the given target directory
func dumpLogs(t *testing.T, ctx context.Context, client klient.Client, namespace string, targetDir string) {

	podList := &corev1.PodList{}

	clientSet, err := kubernetes.NewForConfig(client.RESTConfig())
	if err != nil {
		t.Logf("Error creating clientset: %v\n", err)
		return
	}

	err = client.Resources(namespace).List(ctx, podList)
	if err != nil {
		t.Logf("Error listing pods: %v\n", err)
		return
	}

	for _, pod := range podList.Items {

		previous := false
		for _, containerStatus := range pod.Status.ContainerStatuses {
			if containerStatus.RestartCount > 0 {
				previous = true
				break
			}
		}

		for _, container := range pod.Spec.Containers {
			logFilePath := filepath.Join(targetDir, fmt.Sprintf("%s-%s-%s.log", t.Name(), pod.Name, container.Name))
			logFile, err := os.Create(logFilePath)
			if err != nil {
				t.Logf("Error creating log file: %v\n", err)
				continue
			}

			req := clientSet.CoreV1().Pods(namespace).GetLogs(pod.Name, &corev1.PodLogOptions{
				Container: container.Name,
				Previous:  previous,
			})
			podLogsStream, err := req.Stream(context.TODO())
			if err != nil {
				t.Logf("Error getting container %s of pod %s logs: %v\n", container.Name, pod.Name, err)
				continue
			}

			_, err = io.Copy(logFile, podLogsStream)
			if err != nil {
				t.Logf("Error writing container %s of pod %s logs: %v\n", container.Name, pod.Name, err)
			} else {
				t.Logf("Wrote container %s of pod %s logs to %s\n", container.Name, pod.Name, logFilePath)
			}

			_ = podLogsStream.Close()
		}
	}
}

// adjustK8SAgentManifests adjusts the namespace of given k8s objects and calls the given callbacks for the containers and the pod
func adjustK8SAgentManifests(objects []k8s.Object, namespace string, containerName string, cbContainer func(container *corev1.Container), cbPod func(pod *corev1.PodSpec)) {
	// Update the agent image and image pull policy as it is already loaded in kind cluster
	for _, obj := range objects {
		obj.SetNamespace(namespace)
		var podSpec *corev1.PodSpec
		switch objWithType := obj.(type) {
		case *appsv1.DaemonSet:
			podSpec = &objWithType.Spec.Template.Spec
		case *appsv1.StatefulSet:
			podSpec = &objWithType.Spec.Template.Spec
		case *appsv1.Deployment:
			podSpec = &objWithType.Spec.Template.Spec
		case *appsv1.ReplicaSet:
			podSpec = &objWithType.Spec.Template.Spec
		case *batchv1.Job:
			podSpec = &objWithType.Spec.Template.Spec
		case *batchv1.CronJob:
			podSpec = &objWithType.Spec.JobTemplate.Spec.Template.Spec
		case *rbacv1.ClusterRoleBinding:
			for idx, subject := range objWithType.Subjects {
				if strings.HasPrefix(subject.Name, "elastic-agent") {
					objWithType.Subjects[idx].Namespace = namespace
				}
			}
			continue
		case *rbacv1.RoleBinding:
			for idx, subject := range objWithType.Subjects {
				if strings.HasPrefix(subject.Name, "elastic-agent") {
					objWithType.Subjects[idx].Namespace = namespace
				}
			}
			continue
		default:
			continue
		}

		for idx, container := range podSpec.Containers {
			if container.Name != containerName {
				continue
			}
			if cbContainer != nil {
				cbContainer(&podSpec.Containers[idx])
			}

			if cbPod != nil {
				cbPod(podSpec)
			}
		}

	}
}

// yamlToK8SObjects converts yaml to k8s objects
func yamlToK8SObjects(reader *bufio.Reader) ([]k8s.Object, error) {
	var objects []k8s.Object

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(rbacv1.SchemeGroupVersion, &rbacv1.ClusterRoleBinding{}, &rbacv1.ClusterRoleBindingList{})
	scheme.AddKnownTypes(rbacv1.SchemeGroupVersion, &rbacv1.ClusterRole{}, &rbacv1.ClusterRoleList{})
	scheme.AddKnownTypes(rbacv1.SchemeGroupVersion, &rbacv1.RoleBinding{}, &rbacv1.RoleBindingList{})
	scheme.AddKnownTypes(rbacv1.SchemeGroupVersion, &rbacv1.Role{}, &rbacv1.RoleList{})
	scheme.AddKnownTypes(corev1.SchemeGroupVersion, &corev1.ServiceAccount{}, &corev1.ServiceAccountList{})
	scheme.AddKnownTypes(corev1.SchemeGroupVersion, &corev1.Service{}, &corev1.ServiceList{})
	scheme.AddKnownTypes(appsv1.SchemeGroupVersion, &appsv1.DaemonSet{})
	scheme.AddKnownTypes(appsv1.SchemeGroupVersion, &appsv1.StatefulSet{})
	scheme.AddKnownTypes(appsv1.SchemeGroupVersion, &appsv1.Deployment{})
	scheme.AddKnownTypes(corev1.SchemeGroupVersion, &corev1.Secret{}, &corev1.ConfigMap{})
	decoder := serializer.NewCodecFactory(scheme).UniversalDeserializer()

	yamlReader := yaml.NewYAMLReader(reader)
	for {
		yamlBytes, err := yamlReader.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("failed to read YAML: %w", err)
		}
		obj, _, err := decoder.Decode(yamlBytes, nil, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to decode YAML: %w", err)
		}

		k8sObj, ok := obj.(k8s.Object)
		if !ok {
			return nil, fmt.Errorf("failed to cast object to k8s.Object: %v", obj)
		}

		objects = append(objects, k8sObj)
	}

	return objects, nil
}

// renderKustomize renders the given kustomize directory to YAML
func renderKustomize(kustomizePath string) ([]byte, error) {
	// Create a file system pointing to the kustomize directory
	fSys := filesys.MakeFsOnDisk()

	// Create a kustomizer
	k := krusty.MakeKustomizer(krusty.MakeDefaultOptions())

	// Run the kustomizer on the given directory
	resMap, err := k.Run(fSys, kustomizePath)
	if err != nil {
		return nil, err
	}

	// Convert the result to YAML
	renderedManifest, err := resMap.AsYaml()
	if err != nil {
		return nil, err
	}

	return renderedManifest, nil
}

// generateESAPIKey generates an API key for the given Elasticsearch.
func generateESAPIKey(esClient *elasticsearch.Client, keyName string) (string, error) {
	apiKeyReqBody := fmt.Sprintf(`{
		"name": "%s",
		"expiration": "1d"
	}`, keyName)

	resp, err := esClient.Security.CreateAPIKey(strings.NewReader(apiKeyReqBody))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	response := make(map[string]interface{})
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return "", err
	}

	keyToken := response["api_key"].(string)
	if keyToken == "" {
		return "", fmt.Errorf("key token is empty")
	}

	keyID := response["id"].(string)
	if keyID == "" {
		return "", fmt.Errorf("key ID is empty")
	}

	return fmt.Sprintf("%s:%s", keyID, keyToken), nil
}

// deleteK8SObjects deletes the given k8s objects and waits for them to be deleted if wait is true.
func deleteK8SObjects(t *testing.T, ctx context.Context, client klient.Client, objects []k8s.Object, wait bool) {
	for _, obj := range objects {
		_ = client.Resources().Delete(ctx, obj)
	}

	if !wait {
		return
	}

	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, 10*time.Second)
	defer timeoutCancel()

	for _, obj := range objects {
		if timeoutCtx.Err() != nil {
			break
		}

		for i := 0; i < 10; i++ {
			if timeoutCtx.Err() != nil {
				break
			}

			err := client.Resources().Get(timeoutCtx, obj.GetName(), obj.GetNamespace(), obj)
			if err != nil {
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
	}

	if timeoutCtx.Err() != nil {
		t.Log("Timeout waiting for k8s objects to be deleted")
	}
}

func int64Ptr(val int64) *int64 {
	valPtr := val
	return &valPtr
}
