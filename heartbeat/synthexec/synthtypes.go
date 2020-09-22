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

package synthexec

import (
	"fmt"
	"net/url"
	"time"

	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

type SynthEvent struct {
	Type           string                 `json:"type"`
	PackageVersion string                 `json:"package_version"`
	Index          int                    `json:"index""`
	Step           *Step                  `json:"step"`
	Journey        *Journey               `json:"journey"`
	Timestamp      time.Time              `json:"@timestamp"`
	Payload        map[string]interface{} `json:"payload"`
	Blob           *string                `json:"blob"`
	Error          *SynthError            `json:"error"`
	URL            string                 `json:"url"`
}

type SynthError struct {
	Name    string `json:"name"`
	Message string `json:"message"`
	Stack   string `json:"stack"`
}

func (se *SynthError) String() string {
	return fmt.Sprintf("%s: %s\n%s", se.Name, se.Message, se.Stack)
}

func (se SynthEvent) ToMap() common.MapStr {
	// We don't add @timestamp to the map string since that's specially handled in beat.Event
	e := common.MapStr{
		"type":            se.Type,
		"package_version": se.PackageVersion,
		"index":           se.Index,
		"payload":         se.Payload,
		"blob":            se.Blob,
	}
	if se.Step != nil {
		e.Put("step", se.Step.ToMap())
	}
	if se.Journey != nil {
		e.Put("journey", se.Journey.ToMap())
	}
	m := common.MapStr{"synthetics": e}
	if se.Error != nil {
		m["error"] = common.MapStr{
			"type":    "synthetics",
			"message": se.Error.String(),
		}
	}
	if se.URL != "" {
		u, e := url.Parse(se.URL)
		if e != nil {
			logp.Warn("Could not parse synthetics URL '%s': %s", se.URL, e.Error())
		} else {
			m["url"] = wrappers.URLFields(u)
		}
	}

	return m
}

type Step struct {
	Name  string `json:"name"`
	Index int    `json:"index"`
}

func (s *Step) ToMap() common.MapStr {
	return common.MapStr{
		"name":  s.Name,
		"index": s.Index,
	}
}

type Journey struct {
	Name string `json:"name"`
	Id   string `json:"id"`
}

func (j Journey) ToMap() common.MapStr {
	return common.MapStr{
		"name": j.Name,
		"id":   j.Id,
	}
}
