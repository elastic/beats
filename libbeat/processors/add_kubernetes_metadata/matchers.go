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

package add_kubernetes_metadata

import (
	"fmt"

	"github.com/elastic/beats/v8/libbeat/beat"
	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/common/fmtstr"
	"github.com/elastic/beats/v8/libbeat/logp"
	"github.com/elastic/beats/v8/libbeat/outputs/codec"
	"github.com/elastic/beats/v8/libbeat/outputs/codec/format"
)

const (
	FieldMatcherName       = "fields"
	FieldFormatMatcherName = "field_format"
)

// Matcher takes a new event and returns the index
type Matcher interface {
	// MetadataIndex returns the index string to use in annotation lookups for the given
	// event. A previous indexer should have generated that index for this to work
	// This function can return "" if the event doesn't match
	MetadataIndex(event common.MapStr) string
}

type Matchers struct {
	matchers []Matcher
}

type MatcherConstructor func(config common.Config) (Matcher, error)

func NewMatchers(configs PluginConfig) *Matchers {
	matchers := []Matcher{}
	for _, pluginConfigs := range configs {
		for name, pluginConfig := range pluginConfigs {
			matchFunc := Indexing.GetMatcher(name)
			if matchFunc == nil {
				logp.Warn("Unable to find matcher plugin %s", name)
				continue
			}

			matcher, err := matchFunc(pluginConfig)
			if err != nil {
				logp.Warn("Unable to initialize matcher plugin %s due to error %v", name, err)
				continue
			}

			matchers = append(matchers, matcher)

		}
	}
	return &Matchers{
		matchers: matchers,
	}
}

// MetadataIndex returns the index string for the first matcher from the Registry returning one
func (m *Matchers) MetadataIndex(event common.MapStr) string {
	for _, matcher := range m.matchers {
		index := matcher.MetadataIndex(event)
		if index != "" {
			return index
		}
	}

	// No index returned
	return ""
}

func (m *Matchers) Empty() bool {
	if len(m.matchers) == 0 {
		return true
	}

	return false
}

type FieldMatcher struct {
	MatchFields []string
}

func NewFieldMatcher(cfg common.Config) (Matcher, error) {
	config := struct {
		LookupFields []string `config:"lookup_fields"`
	}{}

	err := cfg.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("fail to unpack the `lookup_fields` configuration: %s", err)
	}

	if len(config.LookupFields) == 0 {
		return nil, fmt.Errorf("lookup_fields can not be empty")
	}

	return &FieldMatcher{MatchFields: config.LookupFields}, nil
}

func (f *FieldMatcher) MetadataIndex(event common.MapStr) string {
	for _, field := range f.MatchFields {
		keyIface, err := event.GetValue(field)
		if err == nil {
			key, ok := keyIface.(string)
			if ok {
				return key
			}
		}
	}

	return ""
}

type FieldFormatMatcher struct {
	Codec codec.Codec
}

func NewFieldFormatMatcher(cfg common.Config) (Matcher, error) {
	config := struct {
		Format string `config:"format"`
	}{}

	err := cfg.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("fail to unpack the `format` configuration of `field_format` matcher: %s", err)
	}

	if config.Format == "" {
		return nil, fmt.Errorf("`format` of `field_format` matcher can't be empty")
	}

	return &FieldFormatMatcher{
		Codec: format.New(fmtstr.MustCompileEvent(config.Format)),
	}, nil

}

func (f *FieldFormatMatcher) MetadataIndex(event common.MapStr) string {
	bytes, err := f.Codec.Encode("", &beat.Event{
		Fields: event,
	})

	if err != nil {
		logp.Debug("kubernetes", "Unable to apply field format pattern on event")
	}

	if len(bytes) == 0 {
		return ""
	}

	return string(bytes)
}
