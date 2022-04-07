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
	"regexp"
	"strconv"
	"strings"

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/logp"
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

	return withECSEnrichment(fields)
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

func withECSEnrichment(fields common.MapStr) common.MapStr {
	// from https://www.freedesktop.org/software/systemd/man/systemd.journal-fields.html
	// we see journald.object fields are populated by systemd on behalf of a different program
	// so we want them to favor their use in root fields as they are the from the effective program
	// performing the action.
	setGidUidFields("journald", fields)
	setGidUidFields("journald.object", fields)
	setProcessFields("journald", fields)
	setProcessFields("journald.object", fields)
	return fields
}

func setGidUidFields(prefix string, fields common.MapStr) {
	var auditLoginUid string
	if found, _ := fields.HasKey(prefix + ".audit.login_uid"); found {
		auditLoginUid = fmt.Sprint(getIntegerFromFields(prefix+".audit.login_uid", fields))
		fields.Put("user.id", auditLoginUid)
	}

	if found, _ := fields.HasKey(prefix + ".uid"); !found {
		return
	}

	uid := fmt.Sprint(getIntegerFromFields(prefix+".uid", fields))
	gid := fmt.Sprint(getIntegerFromFields(prefix+".gid", fields))
	if auditLoginUid != "" && auditLoginUid != uid {
		putStringIfNotEmtpy("user.effective.id", uid, fields)
		putStringIfNotEmtpy("user.effective.group.id", gid, fields)
	} else {
		putStringIfNotEmtpy("user.id", uid, fields)
		putStringIfNotEmtpy("user.group.id", gid, fields)
	}
}

var cmdlineRegexp = regexp.MustCompile(`"(\\"|[^"])*?"|[^\s]+`)

func setProcessFields(prefix string, fields common.MapStr) {
	if found, _ := fields.HasKey(prefix + ".pid"); found {
		pid := getIntegerFromFields(prefix+".pid", fields)
		fields.Put("process.pid", pid)
	}

	name := getStringFromFields(prefix+".name", fields)
	if name != "" {
		fields.Put("process.name", name)
	}

	executable := getStringFromFields(prefix+".executable", fields)
	if executable != "" {
		fields.Put("process.executable", executable)
	}

	cmdline := getStringFromFields(prefix+".process.command_line", fields)
	if cmdline == "" {
		return
	}

	fields.Put("process.command_line", cmdline)

	args := cmdlineRegexp.FindAllString(cmdline, -1)
	if len(args) > 0 {
		fields.Put("process.args", args)
		fields.Put("process.args_count", len(args))
	}
}

func getStringFromFields(key string, fields common.MapStr) string {
	value, _ := fields.GetValue(key)
	str, _ := value.(string)
	return str
}

func getIntegerFromFields(key string, fields common.MapStr) int64 {
	value, _ := fields.GetValue(key)
	i, _ := value.(int64)
	return i
}

func putStringIfNotEmtpy(k, v string, fields common.MapStr) {
	if v == "" {
		return
	}
	fields.Put(k, v)
}

// helpers for creating a field conversion table.

var ignoredField = Conversion{Dropped: true}

func text(names ...string) Conversion {
	return Conversion{Names: names, IsInteger: false, Dropped: false}
}

func integer(names ...string) Conversion {
	return Conversion{Names: names, IsInteger: true, Dropped: false}
}
