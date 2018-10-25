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

// +build !linux !cgo

package cmd

import (
	"fmt"

	"github.com/elastic/beats/libbeat/beat"
	cmd "github.com/elastic/beats/libbeat/cmd"
	"github.com/elastic/beats/libbeat/common"
)

// Name of this beat
var Name = "journalbeat"

// RootCmd to handle beats cli
var RootCmd = cmd.GenRootCmd(Name, "", newBeat)

func newBeat(b *beat.Beat, cfg *common.Config) (beat.Beater, error) {
	return nil, fmt.Errorf("journalbeat is not supported on your platform")
}
