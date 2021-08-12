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
	"github.com/elastic/beats/v7/libbeat/common"
)

func genOrchestratorFields(kubemeta common.MapStr, kind string) common.MapStr {
	fields := common.MapStr{
		"type": "kubernetes",
	}
	resource := common.MapStr{
		"type": kind,
	}
	namespace, err := kubemeta.GetValue("namespace")
	if err == nil {
		fields.Put("namespace", namespace)
	}

	nodeMap, err := kubemeta.GetValue(kind)
	if err == nil {
		node, _ := nodeMap.(common.MapStr)
		name, err := node.GetValue("name")
		if err == nil {
			resource.Put("name", name)
		}
	}
	fields.Put("resource", resource)
	return fields
}
