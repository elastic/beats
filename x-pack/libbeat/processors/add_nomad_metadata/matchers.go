// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package add_nomad_metadata

import (
	"fmt"
	"sync"

	"github.com/elastic/beats/v8/libbeat/beat"
	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/common/fmtstr"
	"github.com/elastic/beats/v8/libbeat/logp"
	"github.com/elastic/beats/v8/libbeat/outputs/codec"
	"github.com/elastic/beats/v8/libbeat/outputs/codec/format"
)

const (
	// FieldMatcherName fields to be used as key for the allocation metadata lookup
	FieldMatcherName = "fields"
	// FieldFormatMatcherName format of the string used to create a key for the allocation metadata lookup
	FieldFormatMatcherName = "field_format"
)

// Matcher takes a new event and returns the index
type Matcher interface {
	// MetadataIndex returns the index string to use in annotation lookups for the given
	// event. A previous indexer should have generated that index for this to work
	// This function can return "" if the event doesn't match
	MetadataIndex(event common.MapStr) string
}

// Matchers holds a collections of Matcher objects and the associated lock
type Matchers struct {
	sync.RWMutex
	matchers []Matcher
}

// MatcherConstructor builds a new Matcher from its settings
type MatcherConstructor func(config common.Config) (Matcher, error)

// NewMatchers builds a Matchers object from its configurations
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
	m.RLock()
	defer m.RUnlock()
	for _, matcher := range m.matchers {
		index := matcher.MetadataIndex(event)
		if index != "" {
			return index
		}
	}

	// No index returned
	return ""
}

// Empty returns true if the matchers list is empty
func (m *Matchers) Empty() bool {
	m.RLock()
	defer m.RUnlock()
	if len(m.matchers) == 0 {
		return true
	}

	return false
}

// FieldMatcher represents a list of fields to match
type FieldMatcher struct {
	MatchFields []string
}

// NewFieldMatcher builds a new list of fields to match from lookup_fields
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

// MetadataIndex returns the first key of an event that satisfies a matcher
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

// FieldFormatMatcher represents a special field formatter that its created given a custom format
type FieldFormatMatcher struct {
	Codec codec.Codec
}

// NewFieldFormatMatcher creates a custom FieldFormtMatcher given a configuration (`format` key)
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

// MetadataIndex returns the content of the dynamic field defined by the FieldFormatMatcher, usually,
// but not limited to, a concatenation of multiple fields
func (f *FieldFormatMatcher) MetadataIndex(event common.MapStr) string {
	bytes, err := f.Codec.Encode("", &beat.Event{
		Fields: event,
	})

	if err != nil {
		logp.Debug("nomad", "Unable to apply field format pattern on event")
	}

	if len(bytes) == 0 {
		return ""
	}

	return string(bytes)
}
