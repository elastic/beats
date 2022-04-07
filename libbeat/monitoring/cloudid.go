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

	errw "github.com/pkg/errors"

	"github.com/elastic/beats/v8/libbeat/cloudid"
	"github.com/elastic/beats/v8/libbeat/common"
)

type cloudConfig struct {
	Cloud struct {
		ID   string `config:"id"`
		Auth string `config:"auth"`
	} `config:"cloud"`
}

func cfgError(cause error) error {
	return errw.Wrap(cause, "error using monitoring.cloud.* settings")
}

var errCloudCfgIncomplete = errors.New("monitoring.cloud.auth specified but monitoring.cloud.id is empty. Please specify both")

// OverrideWithCloudSettings overrides monitoring.elasticsearch.* with
// monitoring.cloud.* if the latter are set.
func OverrideWithCloudSettings(monitoringCfg *common.Config) error {
	var config cloudConfig
	if err := monitoringCfg.Unpack(&config); err != nil {
		return cfgError(err)
	}

	if config.Cloud.ID == "" && config.Cloud.Auth == "" {
		// Nothing to do
		return nil
	}

	if config.Cloud.ID == "" && config.Cloud.Auth != "" {
		return cfgError(errCloudCfgIncomplete)
	}

	// We remove monitoring.cloud.* so that "cloud" is not treated as a type of
	// monitoring reporter later.
	if _, err := monitoringCfg.Remove("cloud", -1); err != nil {
		return cfgError(err)
	}

	cid, err := cloudid.NewCloudID(config.Cloud.ID, config.Cloud.Auth)
	if err != nil {
		return cfgError(err)
	}

	if err := overwriteWithCloudID(monitoringCfg, cid); err != nil {
		return cfgError(err)
	}

	if config.Cloud.Auth != "" {
		if err := overwriteWithCloudAuth(monitoringCfg, cid); err != nil {
			return cfgError(err)
		}
	}

	return nil
}

func overwriteWithCloudID(monitoringCfg *common.Config, cid *cloudid.CloudID) error {
	esURLConfig, err := common.NewConfigFrom([]string{cid.ElasticsearchURL()})
	if err != nil {
		return err
	}

	if err := monitoringCfg.SetChild("elasticsearch.hosts", -1, esURLConfig); err != nil {
		return err
	}

	return nil
}

func overwriteWithCloudAuth(monitoringCfg *common.Config, cid *cloudid.CloudID) error {
	if err := monitoringCfg.SetString("elasticsearch.username", -1, cid.Username()); err != nil {
		return err
	}

	if err := monitoringCfg.SetString("elasticsearch.password", -1, cid.Password()); err != nil {
		return err
	}

	return nil
}
