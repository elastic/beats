// Licensed to Elasticsearch B.V. under one or more contributor license agreements.
// See the NOTICE file distributed with this work for additional information regarding copyright ownership.
// Elasticsearch B.V. licenses this file to you under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software distributed
// under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
// either express or implied.  See the License for the specific language governing permissions and limitations under the License.

package ntp

import (
	"fmt"
	"time"
)

type config struct {
	Host    string        `config:"host"`
	Timeout time.Duration `config:"timeout"`
	Version int           `config:"version"`
}

func defaultConfig() config {
	return config{
		Timeout: 5 * time.Second,
		Version: 4,
	}
}

func validateConfig(cfg *config) error {
	if cfg.Host == "" {
		return fmt.Errorf("NTP host must be set in config")
	}
	if cfg.Timeout <= 0 {
		return fmt.Errorf("invalid NTP timeout: %s", cfg.Timeout.String())
	}
	if cfg.Version != 3 && cfg.Version != 4 {
		return fmt.Errorf("NTP version must be 3 or 4")
	}
	return nil
}
