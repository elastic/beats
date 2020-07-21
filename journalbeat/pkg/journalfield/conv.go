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

package journalfield

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

// FieldConversion provides the mappings and conversion rules for raw fields of journald entries.
type FieldConversion map[string]Conversion

// Conversion configures the conversion rules for a field.
type Conversion struct {
	Names     []string
	IsInteger bool
	Dropped   bool
}

// Converter applis configured conversion rules to journald entries, producing
// a new common.MapStr.
type Converter struct {
	log         *logp.Logger
	conversions FieldConversion
}

// NewConverter creates a new Converter from the given conversion rules. If
// conversions is nil, internal default conversion rules will be applied.
func NewConverter(log *logp.Logger, conversions FieldConversion) *Converter {
	if conversions == nil {
		conversions = journaldEventFields
	}

	return &Converter{log: log, conversions: conversions}
}

// Convert creates a common.MapStr from the raw fields by applying the
// configured conversion rules.
// Field type conversion errors are logged to at debug level and the original
// value is added to the map.
func (c *Converter) Convert(entryFields map[string]string) common.MapStr {
	fields := common.MapStr{}
	var custom common.MapStr

	for entryKey, v := range entryFields {
		if fieldConversionInfo, ok := c.conversions[entryKey]; !ok {
			if custom == nil {
				custom = common.MapStr{}
			}
			normalized := strings.ToLower(strings.TrimLeft(entryKey, "_"))
			custom.Put(normalized, v)
		} else if !fieldConversionInfo.Dropped {
			value, err := convertValue(fieldConversionInfo, v)
			if err != nil {
				value = v
				c.log.Debugf("Journald mapping error: %v", err)
			}
			for _, name := range fieldConversionInfo.Names {
				fields.Put(name, value)
			}
		}
	}

	if len(custom) != 0 {
		fields.Put("journald.custom", custom)
	}

	return fields
}

func convertValue(fc Conversion, value string) (interface{}, error) {
	if fc.IsInteger {
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			// On some versions of systemd the 'syslog.pid' can contain the username
			// appended to the end of the pid. In most cases this does not occur
			// but in the cases that it does, this tries to strip ',\w*' from the
			// value and then perform the conversion.
			s := strings.Split(value, ",")
			v, err = strconv.ParseInt(s[0], 10, 64)
			if err != nil {
				return value, fmt.Errorf("failed to convert field %s \"%v\" to int: %v", fc.Names[0], value, err)
			}
		}
		return v, nil
	}
	return value, nil
}

// helpers for creating a field conversion table.

var ignoredField = Conversion{Dropped: true}

func text(names ...string) Conversion {
	return Conversion{Names: names, IsInteger: false, Dropped: false}
}

func integer(names ...string) Conversion {
	return Conversion{Names: names, IsInteger: true, Dropped: false}
}
