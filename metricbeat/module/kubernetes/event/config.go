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
	"errors"
	"time"
)

type kubeEventsConfig struct {
	InCluster        bool          `config:"in_cluster"`
	KubeConfig       string        `config:"kube_config"`
	Namespace        string        `config:"namespace"`
	SyncPeriod       time.Duration `config:"sync_period"`
	LabelsDedot      bool          `config:"labels.dedot"`
	AnnotationsDedot bool          `config:"annotations.dedot"`
}

type Enabled struct {
	Enabled bool `config:"enabled"`
}

func defaultKubernetesEventsConfig() kubeEventsConfig {
	return kubeEventsConfig{
		InCluster:        true,
		SyncPeriod:       1 * time.Second,
		LabelsDedot:      true,
		AnnotationsDedot: true,
	}
}

func (c kubeEventsConfig) Validate() error {
	if !c.InCluster && c.KubeConfig == "" {
		return errors.New("`kube_config` path can't be empty when in_cluster is set to false")
	}
	return nil
}
