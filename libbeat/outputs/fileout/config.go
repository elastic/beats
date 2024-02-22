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
	"github.com/elastic/beats/v7/libbeat/outputs/codec"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/file"
)

type fileOutConfig struct {
	Path            *PathFormatString `config:"path"`
	Filename        string            `config:"filename"`
	RotateEveryKb   uint              `config:"rotate_every_kb" validate:"min=1"`
	NumberOfFiles   uint              `config:"number_of_files"`
	Codec           codec.Config      `config:"codec"`
	Permissions     uint32            `config:"permissions"`
	RotateOnStartup bool              `config:"rotate_on_startup"`
	Queue           config.Namespace  `config:"queue"`
}

func defaultConfig() fileOutConfig {
	return fileOutConfig{
		NumberOfFiles:   7,
		RotateEveryKb:   10 * 1024,
		Permissions:     0600,
		RotateOnStartup: true,
	}
}

func readConfig(cfg *config.C) (*fileOutConfig, error) {
	foConfig := defaultConfig()
	if err := cfg.Unpack(&foConfig); err != nil {
		return nil, err
	}

	// disable bulk support in publisher pipeline
	_ = cfg.SetInt("bulk_max_size", -1, -1)

	return &foConfig, nil
}

func (c *fileOutConfig) Validate() error {
	if c.NumberOfFiles < 2 || c.NumberOfFiles > file.MaxBackupsLimit {
		return fmt.Errorf("the number_of_files to keep should be between 2 and %v",
			file.MaxBackupsLimit)
	}

	return nil
}
