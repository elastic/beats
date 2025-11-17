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

package initialization

import (
	"errors"
	"fmt"

	"github.com/elastic/beats/v7/libbeat/autodiscover"
	"github.com/elastic/beats/v7/libbeat/autodiscover/appenders"
	"github.com/elastic/beats/v7/libbeat/autodiscover/appenders/config"
	"github.com/elastic/beats/v7/libbeat/autodiscover/builder"
	"github.com/elastic/beats/v7/libbeat/autodiscover/providers"
	"github.com/elastic/beats/v7/libbeat/plugin"
)

func Setup(r *autodiscover.Registry) error {
	if err := providers.AddKnownProviders(r, providers.KnownProviders); err != nil {
		return fmt.Errorf("error adding libbeat autodiscover providers: %w", err)
	}
	if err := r.AddAppender("config", config.NewConfigAppender); err != nil {
		return fmt.Errorf("error adding libbeat autodiscover configuration appender: %w", err)
	}
	if err := providers.RegisterPluginProviders(r); err != nil && !errors.Is(err, plugin.ErrLoaderAlreadyRegistered) {
		return fmt.Errorf("error registering autodiscover plugin providers: %w", err)
	}
	if err := appenders.PluginInit(r); err != nil && !errors.Is(err, plugin.ErrLoaderAlreadyRegistered) {
		return fmt.Errorf("error registering autodiscover plugin appenders: %w", err)
	}
	if err := builder.PluginInit(r); err != nil && !errors.Is(err, plugin.ErrLoaderAlreadyRegistered) {
		return fmt.Errorf("error registering autodiscover plugin builder: %w", err)
	}
	return nil
}
