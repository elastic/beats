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

package ilm

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

const pattern = "000001"

type ilmPolicyCfg struct {
	idxName    string
	policyName string
}

var (
	reVersion = regexp.MustCompile("%.*{.*beat.version.?}")
	reInvalid = regexp.MustCompile("%.*{.*}")
)

func newIlmPolicyCfg(idxName, policyName string, info beat.Info) *ilmPolicyCfg {
	idx := reVersion.ReplaceAllLiteralString(strings.ToLower(idxName), info.Version)
	if reInvalid.MatchString(idx) {
		logp.Warn("index %s is not a valid index name for ILM usage", idxName)
		return nil
	}
	return &ilmPolicyCfg{idxName: idx, policyName: policyName}
}

func (cfg *ilmPolicyCfg) buildILMTemplate() common.MapStr {
	return common.MapStr{
		"order":          2,
		"index_patterns": fmt.Sprintf("%s*", cfg.idxName),
		"settings": common.MapStr{
			"index": common.MapStr{
				"lifecycle": common.MapStr{
					"name":           cfg.policyName,
					"rollover_alias": cfg.idxName,
				},
			},
		},
	}
}
