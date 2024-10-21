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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	"github.com/elastic/elastic-agent/pkg/testing/define"
)

func TestKubernetesAgentService(t *testing.T) {
	info := define.Require(t, define.Requirements{
		Stack: &define.Stack{},
		Local: false,
		Sudo:  false,
		OS: []define.OS{
			// only test the service container
			{Type: define.Kubernetes, DockerVariant: "service"},
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

	hasher := sha256.New()
	hasher.Write([]byte(t.Name()))
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

			// has a unique entrypoint and command because its ran in the cloud
			// adjust the spec to run it correctly
			container.Command = []string{"elastic-agent"}
			container.Args = []string{"-c", "/etc/elastic-agent/agent.yml", "-e"}
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

	// update the configmap to only run the connectors input
	serviceAgentYAML, err := os.ReadFile(filepath.Join("testdata", "connectors.agent.yml"))
	require.NoError(t, err)
	for _, obj := range k8sObjects {
		switch objWithType := obj.(type) {
		case *corev1.ConfigMap:
			_, ok := objWithType.Data["agent.yml"]
			if ok {
				objWithType.Data["agent.yml"] = string(serviceAgentYAML)
			}
		}
	}

	ctx := context.Background()

	deployK8SAgent(t, ctx, client, k8sObjects, testNamespace, false, testLogsBasePath, map[string]bool{
		"connectors-py": true,
	})
}
