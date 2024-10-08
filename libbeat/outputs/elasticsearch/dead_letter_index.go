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

package elasticsearch

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/elastic-agent-libs/config"
)

const (
	drop              = "drop"
	dead_letter_index = "dead_letter_index"
)

func deadLetterIndexForConfig(config *config.C) (string, error) {
	var indexConfig struct {
		Index string
	}
	err := config.Unpack(&indexConfig)
	if err != nil {
		return "", err
	}
	if indexConfig.Index == "" {
		return "", fmt.Errorf("%s policy requires an `index` to be specified", dead_letter_index)
	}
	return indexConfig.Index, nil
}

func deadLetterIndexForPolicy(configNamespace *config.Namespace) (string, error) {
	if configNamespace == nil || configNamespace.Name() == drop {
		return "", nil
	}
	if configNamespace.Name() == dead_letter_index {
		cfgwarn.Beta("The non_indexable_policy dead_letter_index is beta.")
		return deadLetterIndexForConfig(configNamespace.Config())
	}
	return "", fmt.Errorf("no such policy type: %s", configNamespace.Name())
}
