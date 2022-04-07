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

package event

import (
	"time"

	"github.com/elastic/beats/v8/libbeat/common/kubernetes"
)

type kubeEventsConfig struct {
	KubeConfig        string                       `config:"kube_config"`
	KubeClientOptions kubernetes.KubeClientOptions `config:"kube_client_options"`
	Namespace         string                       `config:"namespace"`
	SyncPeriod        time.Duration                `config:"sync_period"`
	LabelsDedot       bool                         `config:"labels.dedot"`
	AnnotationsDedot  bool                         `config:"annotations.dedot"`
	SkipOlder         bool                         `config:"skip_older"`
}

type Enabled struct {
	Enabled bool `config:"enabled"`
}

func defaultKubernetesEventsConfig() kubeEventsConfig {
	return kubeEventsConfig{
		SyncPeriod:       10 * time.Minute,
		LabelsDedot:      true,
		AnnotationsDedot: true,
		SkipOlder:        true,
	}
}
