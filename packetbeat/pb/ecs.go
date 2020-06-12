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

package pb

import "time"

type ecsEvent struct {
	ID   string `ecs:"id"`
	Code string `ecs:"code"`
	Kind string `ecs:"kind"`
	// overridden because this needs to be an array
	Category []string `ecs:"category"`
	Action   string   `ecs:"action"`
	Outcome  string   `ecs:"outcome"`
	// overridden because this needs to be an array
	Type          []string      `ecs:"type"`
	Module        string        `ecs:"module"`
	Dataset       string        `ecs:"dataset"`
	Provider      string        `ecs:"provider"`
	Severity      int64         `ecs:"severity"`
	Original      string        `ecs:"original"`
	Hash          string        `ecs:"hash"`
	Duration      time.Duration `ecs:"duration"`
	Sequence      int64         `ecs:"sequence"`
	Timezone      string        `ecs:"timezone"`
	Created       time.Time     `ecs:"created"`
	Start         time.Time     `ecs:"start"`
	End           time.Time     `ecs:"end"`
	RiskScore     float64       `ecs:"risk_score"`
	RiskScoreNorm float64       `ecs:"risk_score_norm"`
	Ingested      time.Time     `ecs:"ingested"`
	Reference     string        `ecs:"reference"`
	Url           string        `ecs:"url"`
}

type ecsRelated struct {
	IP   []string `ecs:"ip"`
	User []string `ecs:"user"`
	Hash []string `ecs:"hash"`

	// for de-dup
	ipSet map[string]struct{}
}
