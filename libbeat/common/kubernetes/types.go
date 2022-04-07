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
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Resource data
type Resource = runtime.Object

// ObjectMeta data
type ObjectMeta = metav1.ObjectMeta

// Pod data
type Pod = v1.Pod

// PodSpec data
type PodSpec = v1.PodSpec

// PodStatus data
type PodStatus = v1.PodStatus

// Node data
type Node = v1.Node

// Namespace data
type Namespace = v1.Namespace

// Container data
type Container = v1.Container

// ContainerPort data
type ContainerPort = v1.ContainerPort

// Event data
type Event = v1.Event

// PodContainerStatus data
type PodContainerStatus = v1.ContainerStatus

// Deployment data
type Deployment = appsv1.Deployment

// ReplicaSet data
type ReplicaSet = appsv1.ReplicaSet

// StatefulSet data
type StatefulSet = appsv1.StatefulSet

// Service data
type Service = v1.Service

// ServiceAccount data
type ServiceAccount = v1.ServiceAccount

// Job data
type Job = batchv1.Job

// CronJob data
type CronJob = batchv1.CronJob

// Role data
type Role = rbacv1.Role

// RoleBinding data
type RoleBinding = rbacv1.RoleBinding

// ClusterRole data
type ClusterRole = rbacv1.ClusterRole

// ClusterRoleBinding data
type ClusterRoleBinding = rbacv1.ClusterRoleBinding

// PodSecurityPolicy data
type PodSecurityPolicy = policyv1beta1.PodSecurityPolicy

// NetworkPolicy data
type NetworkPolicy = networkingv1.NetworkPolicy

const (
	// PodPending phase
	PodPending = v1.PodPending
	// PodRunning phase
	PodRunning = v1.PodRunning
	// PodSucceeded phase
	PodSucceeded = v1.PodSucceeded
	// PodFailed phase
	PodFailed = v1.PodFailed
	// PodUnknown phase
	PodUnknown = v1.PodUnknown
)

// Time extracts time from k8s.Time type
func Time(t *metav1.Time) time.Time {
	return t.Time
}

// MicroTime extracts time from k8s.MicroTime type
func MicroTime(t *metav1.MicroTime) time.Time {
	return t.Time
}

// ContainerID parses the container ID to get the actual ID string
func ContainerID(s PodContainerStatus) string {
	cID, _ := ContainerIDWithRuntime(s)
	return cID
}

// ContainerIDWithRuntime parses the container ID to get the actual ID string
func ContainerIDWithRuntime(s PodContainerStatus) (string, string) {
	cID := s.ContainerID
	if cID != "" {
		parts := strings.Split(cID, "://")
		if len(parts) == 2 {
			return parts[1], parts[0]
		}
	}
	return "", ""
}
