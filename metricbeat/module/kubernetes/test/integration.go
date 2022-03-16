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

package test

import (
	"testing"
)

// GetAPIServerConfig function returns configuration for talking to Kubernetes API server.
func GetAPIServerConfig(t *testing.T, metricSetName string) map[string]interface{} {
	t.Helper()
	return map[string]interface{}{
		"module":            "kubernetes",
		"metricsets":        []string{metricSetName},
		"host":              "${NODE_NAME}",
		"hosts":             []string{"https://${KUBERNETES_SERVICE_HOST}:${KUBERNETES_SERVICE_PORT}"},
		"bearer_token_file": "/var/run/secrets/kubernetes.io/serviceaccount/token",
		"ssl": map[string]interface{}{
			"certificate_authorities": []string{
				"/var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
			},
		},
	}
}

// GetKubeStateMetricsConfig function returns configuration for talking to kube-state-metrics.
func GetKubeStateMetricsConfig(t *testing.T, metricSetName string) map[string]interface{} {
	t.Helper()
	return map[string]interface{}{
		"module":     "kubernetes",
		"metricsets": []string{metricSetName},
		"host":       "${NODE_NAME}",
		"hosts":      []string{"kube-state-metrics:8080"},
	}
}

// GetKubeletConfig function returns configuration for talking to Kubelet API.
func GetKubeletConfig(t *testing.T, metricSetName string) map[string]interface{} {
	t.Helper()
	return map[string]interface{}{
		"module":            "kubernetes",
		"metricsets":        []string{metricSetName},
		"host":              "${NODE_NAME}",
		"hosts":             []string{"https://localhost:10250"},
		"bearer_token_file": "/var/run/secrets/kubernetes.io/serviceaccount/token",
		"ssl": map[string]interface{}{
			"verification_mode": "none",
		},
	}
}

// GetKubeProxyConfig function returns configuration for talking to kube-proxy.
func GetKubeProxyConfig(t *testing.T, metricSetName string) map[string]interface{} {
	t.Helper()
	return map[string]interface{}{
		"module":     "kubernetes",
		"metricsets": []string{metricSetName},
		"host":       "${NODE_NAME}",
		"hosts":      []string{"localhost:10249"},
	}
}

// GetSchedulerConfig function returns configuration for talking to kube-scheduler.
func GetSchedulerConfig(t *testing.T, metricSetName string) map[string]interface{} {
	t.Helper()
	return map[string]interface{}{
		"module":            "kubernetes",
		"metricsets":        []string{metricSetName},
		"host":              "${NODE_NAME}",
		"hosts":             []string{"https://0.0.0.0:10259"},
		"bearer_token_file": "/var/run/secrets/kubernetes.io/serviceaccount/token",
		"ssl": map[string]interface{}{
			"verification_mode": "none",
		},
	}
}

// GetControllerManagerConfig function returns configuration for talking to kube-controller-manager.
func GetControllerManagerConfig(t *testing.T, metricSetName string) map[string]interface{} {
	t.Helper()
	return map[string]interface{}{
		"module":            "kubernetes",
		"metricsets":        []string{metricSetName},
		"host":              "${NODE_NAME}",
		"hosts":             []string{"https://0.0.0.0:10257"},
		"bearer_token_file": "/var/run/secrets/kubernetes.io/serviceaccount/token",
		"ssl": map[string]interface{}{
			"verification_mode": "none",
		},
	}
}
