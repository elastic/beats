// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package detector

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/markbates/pkger"
)

const embeddedRulesFile = "go.elastic.co/go-licence-detector:/assets/rules.json"

// rulesFile represents the structure of the rules file.
type rulesFile struct {
	Allowlist  []string `json:"allowlist"`
	Maybelist []string `json:"maybelist"`
}

// Rules holds rules for the detector.
type Rules struct {
	AllowList  map[string]struct{}
	Maybelist map[string]struct{}
}

// LoadRules loads rules from the given path. Embedded rules file is loaded if the path is empty.
func LoadRules(path string) (*Rules, error) {
	var f io.ReadCloser
	var err error

	if path == "" {
		f, err = pkger.Open(embeddedRulesFile)
	} else {
		f, err = os.Open(path)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to open rules file: %w", err)
	}
	defer f.Close()

	ruleBytes, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read rules: %w", err)
	}

	var rf rulesFile
	if err := json.Unmarshal(ruleBytes, &rf); err != nil {
		return nil, fmt.Errorf("failed to unmarshal rules: %w", err)
	}

	rules := &Rules{
		AllowList:  make(map[string]struct{}, len(rf.Allowlist)),
		Maybelist: make(map[string]struct{}, len(rf.Maybelist)),
	}

	for _, w := range rf.Allowlist {
		rules.AllowList[w] = struct{}{}
	}

	for _, y := range rf.Maybelist {
		rules.Maybelist[y] = struct{}{}
	}

	return rules, nil
}

// IsAllowed returns true if the given licence is allowed by the rules.
func (r *Rules) IsAllowed(licenceID string) bool {
	_, isAllowListed := r.AllowList[licenceID]
	_, isMaybeListed := r.Maybelist[licenceID]
	return isAllowListed || isMaybeListed
}
