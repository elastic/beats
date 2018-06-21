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

package autodiscover

import (
	"fmt"
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
	"github.com/elastic/beats/libbeat/logp"
)

// Appender provides an interface by which extra configuration can be added into configs
type Appender interface {
	// Append takes a processed event and add extra configuration
	Append(event bus.Event)
}

// Appenders is a list of Appender objects
type Appenders []Appender

// AppenderBuilder is a func used to generate a Appender object
type AppenderBuilder func(*common.Config) (Appender, error)

// AddBuilder registers a new AppenderBuilder
func (r *registry) AddAppender(name string, appender AppenderBuilder) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if name == "" {
		return fmt.Errorf("appender name is required")
	}

	_, exists := r.appenders[name]
	if exists {
		return fmt.Errorf("appender '%s' is already registered", name)
	}

	if appender == nil {
		return fmt.Errorf("appender '%s' cannot be registered with a nil factory", name)
	}

	r.appenders[name] = appender
	logp.Debug(debugK, "Appender registered: %s", name)
	return nil
}

// GetAppender returns the appender with the giving name, nil if it doesn't exist
func (r *registry) GetAppender(name string) AppenderBuilder {
	r.lock.RLock()
	defer r.lock.RUnlock()

	name = strings.ToLower(name)
	return r.appenders[name]
}

// BuildAppender reads provider configuration and instantiate one
func (r *registry) BuildAppender(c *common.Config) (Appender, error) {
	var config AppenderConfig
	err := c.Unpack(&config)
	if err != nil {
		return nil, err
	}

	appender := r.GetAppender(config.Type)
	if appender == nil {
		return nil, fmt.Errorf("unknown autodiscover appender %s", config.Type)
	}

	return appender(c)
}

// Append uses all initialized appenders to modify generated bus.Events.
func (a Appenders) Append(event bus.Event) {
	for _, appender := range a {
		appender.Append(event)
	}
}

// NewAppenders instances and returns the given list of appenders.
func NewAppenders(aConfigs []*common.Config) (Appenders, error) {
	var appenders Appenders
	for _, acfg := range aConfigs {
		appender, err := Registry.BuildAppender(acfg)
		if err != nil {
			return nil, err
		}
		appenders = append(appenders, appender)
	}

	return appenders, nil
}
