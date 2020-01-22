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

package fileout

import (
	"fmt"

	"github.com/elastic/beats/libbeat/common/file"
	"github.com/elastic/beats/libbeat/outputs/codec"
)

type config struct {
	Path          string       `config:"path"`
	Filename      string       `config:"filename"`
	RotateEveryKb uint         `config:"rotate_every_kb" validate:"min=1"`
	NumberOfFiles uint         `config:"number_of_files"`
	Codec         codec.Config `config:"codec"`
	Permissions   uint32       `config:"permissions"`
}

var (
	defaultConfig = config{
		NumberOfFiles: 7,
		RotateEveryKb: 10 * 1024,
		Permissions:   0600,
	}
)

func (c *config) Validate() error {
	if c.NumberOfFiles < 2 || c.NumberOfFiles > file.MaxBackupsLimit {
		return fmt.Errorf("The number_of_files to keep should be between 2 and %v",
			file.MaxBackupsLimit)
	}

	return nil
}
