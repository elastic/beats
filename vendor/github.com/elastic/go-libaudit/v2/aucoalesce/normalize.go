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

package aucoalesce

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

//go:generate sh -c "go run mknormalize_data.go normalizationData normalizations.yaml > znormalize_data.go"

var (
	syscallNorms    map[string]*Normalization
	recordTypeNorms map[string][]*Normalization
)

func init() {
	data, err := asset("normalizationData")
	if err != nil {
		panic("normalizationData not found in assets")
	}

	syscallNorms, recordTypeNorms, err = LoadNormalizationConfig(data)
	if err != nil {
		panic(errors.Wrap(err, "failed to parse built in normalization mappings"))
	}
}

// Strings is a custom type to enable YAML values that can be either a string
// or a list of strings.
type Strings struct {
	Values []string
}

func (s *Strings) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var singleValue string
	if err := unmarshal(&singleValue); err == nil {
		s.Values = []string{singleValue}
		return nil
	}

	return unmarshal(&s.Values)
}

type NormalizationConfig struct {
	Default        Normalization `yaml:"default"`
	Normalizations []Normalization
}

type Normalization struct {
	Subject     SubjectMapping `yaml:"subject"`
	Action      string         `yaml:"action"`
	Object      ObjectMapping  `yaml:"object"`
	How         Strings        `yaml:"how"`
	RecordTypes Strings        `yaml:"record_types"`
	Syscalls    Strings        `yaml:"syscalls"`
	SourceIP    Strings        `yaml:"source_ip"`
	HasFields   Strings        `yaml:"has_fields"`
	ECS         ECSMapping     `yaml:"ecs"`
}

type ECSFieldMapping struct {
	From readReference  `yaml:"from" json:"from"`
	To   writeReference `yaml:"to" json:"to"`
}

type ECSMapping struct {
	Category Strings           `yaml:"category"`
	Type     Strings           `yaml:"type"`
	Mappings []ECSFieldMapping `yaml:"mappings"`
}

type SubjectMapping struct {
	PrimaryFieldName   Strings `yaml:"primary"`
	SecondaryFieldName Strings `yaml:"secondary"`
}

type ObjectMapping struct {
	PrimaryFieldName   Strings `yaml:"primary"`
	SecondaryFieldName Strings `yaml:"secondary"`
	What               string  `yaml:"what"`
	PathIndex          int     `yaml:"path_index"`
}

type readReference func(*Event) string
type writeReference func(*Event, string)

var (
	fromFieldReferences = map[string]readReference{
		"subject.primary": func(event *Event) string {
			return event.Summary.Actor.Primary
		},
		"subject.secondary": func(event *Event) string {
			return event.Summary.Actor.Secondary
		},
		"object.primary": func(event *Event) string {
			return event.Summary.Object.Primary
		},
		"object.secondary": func(event *Event) string {
			return event.Summary.Object.Secondary
		},
	}

	fromDictReferences = map[string]func(key string) readReference{
		"data": func(key string) readReference {
			return func(event *Event) string {
				return event.Data[key]
			}
		},
		"uid": func(key string) readReference {
			return func(event *Event) string {
				return event.User.IDs[key]
			}
		},
	}

	toFieldReferences = map[string]writeReference{
		"user": func(event *Event, s string) {
			event.ECS.User.set(s)
		},
		"user.effective": func(event *Event, s string) {
			event.ECS.User.Effective.set(s)
		},
		"user.target": func(event *Event, s string) {
			event.ECS.User.Target.set(s)
		},
		"user.changes": func(event *Event, s string) {
			event.ECS.User.Changes.set(s)
		},
		"group": func(event *Event, s string) {
			event.ECS.Group.set(s)
		},
	}
)

func resolveFieldReference(fieldRef string) (ref readReference) {
	if ref = fromFieldReferences[fieldRef]; ref != nil {
		return
	}
	if dot := strings.IndexByte(fieldRef, '.'); dot != -1 {
		dict := fieldRef[:dot]
		key := fieldRef[dot+1:]
		if accessor := fromDictReferences[dict]; accessor != nil {
			return accessor(key)
		}
	}
	return nil
}

func (ref *readReference) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var fieldRef string
	if err := unmarshal(&fieldRef); err != nil {
		return err
	}
	if *ref = resolveFieldReference(fieldRef); *ref == nil {
		return fmt.Errorf("field '%s' is not a valid from-reference for ECS mapping", fieldRef)
	}
	return nil
}

func (ref *writeReference) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var fieldRef string
	if err := unmarshal(&fieldRef); err != nil {
		return err
	}
	if *ref = toFieldReferences[fieldRef]; *ref == nil {
		return fmt.Errorf("field '%s' is not a valid to-reference for ECS mapping", fieldRef)
	}
	return nil
}

func LoadNormalizationConfig(b []byte) (syscalls map[string]*Normalization, recordTypes map[string][]*Normalization, err error) {
	c := &NormalizationConfig{}
	if err := yaml.Unmarshal(b, c); err != nil {
		return nil, nil, err
	}

	syscalls = map[string]*Normalization{}
	recordTypes = map[string][]*Normalization{}

	for i := range c.Normalizations {
		norm := c.Normalizations[i]
		for _, syscall := range norm.Syscalls.Values {
			if _, found := syscalls[syscall]; found {
				return nil, nil, fmt.Errorf("duplication mappings for syscall %v", syscall)
			}
			syscalls[syscall] = &norm
		}
		for _, recordType := range norm.RecordTypes.Values {
			norms, found := recordTypes[recordType]
			if found {
				for _, n := range norms {
					if len(n.HasFields.Values) == 0 {
						return nil, nil, fmt.Errorf("duplication mappings for record_type %v without has_fields qualifier", recordType)
					}
				}
			}
			recordTypes[recordType] = append(norms, &norm)
		}
	}

	return syscalls, recordTypes, nil
}
