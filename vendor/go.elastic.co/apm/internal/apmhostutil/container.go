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

package apmhostutil

import "go.elastic.co/apm/model"

// Container returns information about the container running the process, or an
// error the container information could not be determined.
func Container() (*model.Container, error) {
	return containerInfo()
}

// Kubernetes returns information about the Kubernetes node and pod running
// the process, or an error if they could not be determined. This information
// does not include the KUBERNETES_* environment variables that can be set via
// the Downward API.
func Kubernetes() (*model.Kubernetes, error) {
	return kubernetesInfo()
}
