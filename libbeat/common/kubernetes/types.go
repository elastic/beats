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

	"github.com/ericchiang/k8s"
	appsv1 "github.com/ericchiang/k8s/apis/apps/v1beta1"
	"github.com/ericchiang/k8s/apis/core/v1"
	extv1 "github.com/ericchiang/k8s/apis/extensions/v1beta1"
	metav1 "github.com/ericchiang/k8s/apis/meta/v1"
)

func init() {
	k8s.Register("", "v1", "events", true, &v1.Event{})
	k8s.RegisterList("", "v1", "events", true, &v1.EventList{})
}

// Resource data
type Resource = k8s.Resource

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
type ReplicaSet = extv1.ReplicaSet

// StatefulSet data
type StatefulSet = appsv1.StatefulSet

// Time extracts time from k8s.Time type
func Time(t *metav1.Time) time.Time {
	return time.Unix(t.GetSeconds(), int64(t.GetNanos()))
}

// ContainerID parses the container ID to get the actual ID string
func ContainerID(s *PodContainerStatus) string {
	cID, _ := ContainerIDWithRuntime(s)
	return cID
}

// ContainerIDWithRuntime parses the container ID to get the actual ID string
func ContainerIDWithRuntime(s *PodContainerStatus) (string, string) {
	cID := s.GetContainerID()
	if cID != "" {
		parts := strings.Split(cID, "://")
		if len(parts) == 2 {
			return parts[1], parts[0]
		}
	}
	return "", ""
}
