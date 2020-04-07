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

// +build windows

package perfmon

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/pkg/errors"
)

const replaceUpperCaseRegex = `(?:[^A-Z_\W])([A-Z])[^A-Z]`

// Config for the windows perfmon metricset.
type Config struct {
	IgnoreNECounters   bool      `config:"perfmon.ignore_non_existent_counters"`
	GroupMeasurements  bool      `config:"perfmon.group_measurements_by_instance"`
	Counters           []Counter `config:"perfmon.counters"`
	Queries            []Query   `config:"perfmon.queries"`
	GroupAllCountersTo string    `config:"perfmon.group_all_counter"`
	MetricFormat       bool
}

// Counter for perfmon counters.
type Counter struct {
	InstanceLabel    string `config:"instance_label"`
	InstanceName     string `config:"instance_name"`
	MeasurementLabel string `config:"measurement_label" validate:"required"`
	Query            string `config:"query"             validate:"required"`
	Format           string `config:"format"`
	Object           string
	ChildQueries     []string
}

// QueryConfig for perfmon queries. This will be used as the new configuration format
type Query struct {
	Name     string         `config:"object" validate:"required"`
	Field    string         `config:"field"`
	Instance []string       `config:"instance"`
	Counters []QueryCounter `config:"counters" validate:"required"`
}

// QueryConfigCounter for perfmon queries. This will be used as the new configuration format
type QueryCounter struct {
	Name   string `config:"name" validate:"required"`
	Field  string `config:"field"`
	Format string `config:"format"`
}

func (conf *Config) ValidateConfig() error {
	if len(conf.Counters) == 0 && len(conf.Queries) == 0 {
		return errors.New("no perfmon counters or queries have been configured")
	}

	if len(conf.Queries) > 0 {
		conf.MetricFormat = true
		conf.Counters = []Counter{}
		for _, query := range conf.Queries {
			for _, counter := range query.Counters {
				if len(query.Instance) == 0 {
					var counterConf = Counter{
						InstanceLabel:    "instance",
						InstanceName:     "",
						MeasurementLabel: "metrics." + mapCounterPathLabel(counter.Field, counter.Name),
						Query:            mapQuery(query.Name, "", counter.Name),
						Format:           counter.Format,
						Object:           query.Name,
					}
					conf.Counters = append(conf.Counters, counterConf)
				} else {
					for _, instance := range query.Instance {
						var counterConf = Counter{
							InstanceLabel:    "instance",
							InstanceName:     instance,
							MeasurementLabel: "metrics." + mapCounterPathLabel(counter.Field, counter.Name),
							Query:            mapQuery(query.Name, instance, counter.Name),
							Format:           counter.Format,
							Object:           query.Name,
						}
						conf.Counters = append(conf.Counters, counterConf)
					}
				}

			}

		}

	}

	// add default format in the config
	for _, value := range conf.Counters {
		form := strings.ToLower(value.Format)
		switch form {
		case "", "float":
			value.Format = "float"
		case "long", "large":
		default:
			return errors.Errorf("initialization failed: format '%s' "+
				"for counter '%s' is invalid (must be float, large or long)",
				value.Format, value.InstanceLabel)
		}
	}

	return nil
}

func (re *Reader) getCounter(query string) (bool, Counter) {
	for _, counter := range re.config.Counters {
		for _, childQuery := range counter.ChildQueries {
			if childQuery == query {
				return true, counter
			}
		}
	}
	return false, Counter{}
}

func mapQuery(obj string, instance string, path string) string {
	var query string
	if strings.HasPrefix(obj, "\\") {
		query = obj
	} else {
		query = fmt.Sprintf("\\%s", obj)
	}
	if instance != "" {
		query += fmt.Sprintf("(%s)", instance)
	}
	if strings.HasPrefix(path, "\\") {
		query += path
	} else {
		query += fmt.Sprintf("\\%s", path)
	}
	return query
}

func mapCounterPathLabel(label string, path string) string {
	var resultMetricName string
	if label != "" {
		resultMetricName = label
	} else {
		resultMetricName = path
	}
	// replace spaces with underscores
	resultMetricName = strings.Replace(resultMetricName, " ", "_", -1)
	// replace backslashes with "per"
	resultMetricName = strings.Replace(resultMetricName, "/sec", "_per_sec", -1)
	resultMetricName = strings.Replace(resultMetricName, "/_sec", "_per_sec", -1)
	resultMetricName = strings.Replace(resultMetricName, "\\", "_", -1)
	// replace actual percentage symbol with the smbol "pct"
	resultMetricName = strings.Replace(resultMetricName, "_%_", "_pct_", -1)
	// create an object in case of ":"
	resultMetricName = strings.Replace(resultMetricName, ":", "_", -1)
	// create an object in case of ":"
	resultMetricName = strings.Replace(resultMetricName, "_-_", "_", -1)
	// replace uppercases with underscores
	resultMetricName = replaceUpperCase(resultMetricName)

	//  avoid cases as this "logicaldisk_avg._disk_sec_per_transfer"
	obj := strings.Split(resultMetricName, ".")
	for index := range obj {
		// in some cases a trailing "_" is found
		obj[index] = strings.TrimPrefix(obj[index], "_")
		obj[index] = strings.TrimSuffix(obj[index], "_")
	}
	resultMetricName = strings.ToLower(strings.Join(obj, "_"))

	return resultMetricName
}

// replaceUpperCase func will replace upper case with '_'
func replaceUpperCase(src string) string {
	replaceUpperCaseRegexp := regexp.MustCompile(replaceUpperCaseRegex)
	return replaceUpperCaseRegexp.ReplaceAllStringFunc(src, func(str string) string {
		var newStr string
		for _, r := range str {
			// split into fields based on class of unicode character
			if unicode.IsUpper(r) {
				newStr += "_" + strings.ToLower(string(r))
			} else {
				newStr += string(r)
			}
		}
		return newStr
	})
}
