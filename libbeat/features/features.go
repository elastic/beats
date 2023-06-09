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

package features

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
	conf "github.com/elastic/elastic-agent-libs/config"
)

var (
	flags = fflags{}
)

type boolValueOnChangeCallback func(new, old bool)

type fflags struct {
	// controls access to the callback hashmap
	callbackMut sync.RWMutex

	// TODO: Refactor to generalize for other feature flags
	fqdnEnabled   atomic.Bool
	fqdnCallbacks map[string]boolValueOnChangeCallback
}

// NewConfigFromProto converts the given *proto.Features object to
// a *config.C object.
func NewConfigFromProto(f *proto.Features) (*conf.C, error) {
	if f == nil {
		return nil, nil
	}

	var beatCfg struct {
		Features *proto.Features `config:"features"`
	}

	beatCfg.Features = f

	c, err := conf.NewConfigFrom(&beatCfg)
	if err != nil {
		return nil, fmt.Errorf("unable to parse feature flags message into beat configuration: %w", err)
	}

	_, err = c.Remove("features.source", -1)
	if err != nil {
		return nil, fmt.Errorf("unable to convert feature flags message to beat configuration: %w", err)
	}

	return c, nil
}

// UpdateFromConfig updates the feature flags configuration. If c is nil UpdateFromConfig is no-op.
func UpdateFromConfig(c *conf.C) error {
	if c == nil {
		return nil
	}

	type cfg struct {
		Features struct {
			FQDN *conf.C `json:"fqdn" yaml:"fqdn" config:"fqdn"`
		} `json:"features" yaml:"features" config:"features"`
	}

	parsedFlags := cfg{}
	if err := c.Unpack(&parsedFlags); err != nil {
		return fmt.Errorf("could not unpack features config: %w", err)
	}

	flags.SetFQDNEnabled(parsedFlags.Features.FQDN.Enabled())

	return nil
}

func (f *fflags) SetFQDNEnabled(newValue bool) {
	f.callbackMut.Lock()
	defer f.callbackMut.Unlock()
	oldValue := f.fqdnEnabled.Swap(newValue)

	for _, cb := range f.fqdnCallbacks {
		cb(newValue, oldValue)
	}

}

// FQDN reports if FQDN should be used instead of hostname for host.name.
// If it hasn't been set by UpdateFromConfig or UpdateFromProto, it returns false.
func FQDN() bool {
	return flags.fqdnEnabled.Load()
}

// AddFQDNOnChangeCallback takes a callback function that will be called with the new and old values
// of `flags.fqdnEnabled` whenever it changes. It also takes a string ID - this is useful
// in calling `RemoveFQDNOnChangeCallback` to de-register the callback.
// if the ID already exists, this returns an error.
func AddFQDNOnChangeCallback(cb boolValueOnChangeCallback, id string) error {
	flags.callbackMut.Lock()
	defer flags.callbackMut.Unlock()

	// Initialize callbacks map if necessary.
	if flags.fqdnCallbacks == nil {
		flags.fqdnCallbacks = map[string]boolValueOnChangeCallback{}
	}

	if _, ok := flags.fqdnCallbacks[id]; ok {
		return fmt.Errorf("callback with ID %s already registered", id)
	}

	flags.fqdnCallbacks[id] = cb
	return nil
}

// RemoveFQDNOnChangeCallback removes the callback function associated with the given ID (originally
// returned by `AddFQDNOnChangeCallback` so that function will be no longer be called when
// `flags.fqdnEnabled` changes.
func RemoveFQDNOnChangeCallback(id string) {
	flags.callbackMut.Lock()
	defer flags.callbackMut.Unlock()

	delete(flags.fqdnCallbacks, id)
}
