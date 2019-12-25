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

package monitoring

import (
	"errors"
	"fmt"

	"github.com/elastic/beats/libbeat/cloudid"

	"github.com/elastic/beats/libbeat/common"
)

type cloudConfig struct {
	Cloud struct {
		ID   string `config:"id"`
		Auth string `config:"auth"`
	} `config:"cloud"`
}

// ErrCloudCfg represents an error when trying to use monitoring.cloud.* settings.
type ErrCloudCfg struct {
	cause error
}

// Error returns the error as a string.
func (e ErrCloudCfg) Error() string {
	return fmt.Sprintf("error using monitoring.cloud.* settings: %v", e.cause)
}

// Unwrap returns the underlying cause of the error.
func (e ErrCloudCfg) Unwrap() error {
	return e.cause
}

// ErrCloudCfgIncomplete is an error indicating that monitoring.cloud.auth has
// been specified but monitoring.cloud.id has not.
var ErrCloudCfgIncomplete = ErrCloudCfg{errors.New("monitoring.cloud.auth specified but monitoring.cloud.id is empty. Please specify both")}

// OverrideWithCloudSettings overrides monitoring.elasticsearch.* with
// monitoring.cloud.* if the latter are set.
func OverrideWithCloudSettings(monitoringCfg *common.Config) error {
	var config cloudConfig
	if err := monitoringCfg.Unpack(&config); err != nil {
		return ErrCloudCfg{err}
	}

	if config.Cloud.Auth != "" && config.Cloud.ID == "" {
		return ErrCloudCfgIncomplete
	}

	// We remove monitoring.cloud.* so that "cloud" is not treated as a type of
	// monitoring reporter later.
	if _, err := monitoringCfg.Remove("cloud", -1); err != nil {
		return ErrCloudCfg{err}
	}

	if err := overwriteWithCloudID(config, monitoringCfg); err != nil {
		return ErrCloudCfg{err}
	}

	if err := overwriteWithCloudAuth(config, monitoringCfg); err != nil {
		return ErrCloudCfg{err}
	}

	return nil
}

func overwriteWithCloudID(config cloudConfig, monitoringCfg *common.Config) error {
	if config.Cloud.ID == "" {
		// Nothing to do
		return nil
	}

	esURL, _, err := cloudid.DecodeCloudID(config.Cloud.ID)
	if err != nil {
		return err
	}

	esURLConfig, err := common.NewConfigFrom([]string{esURL})
	if err != nil {
		return err
	}

	if err := monitoringCfg.SetChild("elasticsearch.hosts", -1, esURLConfig); err != nil {
		return err
	}

	return err
}

func overwriteWithCloudAuth(config cloudConfig, monitoringCfg *common.Config) error {
	if config.Cloud.Auth == "" {
		// Nothing to do
		return nil
	}

	username, password, err := cloudid.DecodeCloudAuth(config.Cloud.Auth)
	if err != nil {
		return err
	}

	if err := monitoringCfg.SetString("elasticsearch.username", -1, username); err != nil {
		return err
	}

	if err := monitoringCfg.SetString("elasticsearch.password", -1, password); err != nil {
		return err
	}

	return err
}
