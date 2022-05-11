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

//go:build windows
// +build windows

package perfmon

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/elastic/beats/v7/metricbeat/helper/windows/pdh"

	"github.com/pkg/errors"

	"math/rand"

	"golang.org/x/sys/windows"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/logp"
)

const (
	instanceCountLabel    = ":count"
	defaultInstanceField  = "instance"
	defaultObjectField    = "object"
	replaceUpperCaseRegex = `(?:[^A-Z_\W])([A-Z])[^A-Z]`
	collectFailedMsg      = "failed collecting counter values"
)

// Reader will contain the config options
type Reader struct {
	query    pdh.Query    // PDH Query
	log      *logp.Logger //
	config   Config       // Metricset configuration
	counters []PerfCounter
	event    windows.Handle
}

type PerfCounter struct {
	InstanceField string
	InstanceName  string
	QueryField    string
	QueryName     string
	Format        string
	ObjectName    string
	ObjectField   string
	ChildQueries  []string
}

// NewReader creates a new instance of Reader.
func NewReader(config Config) (*Reader, error) {
	var query pdh.Query
	if err := query.Open(); err != nil {
		return nil, err
	}
	event, err := windows.CreateEvent(nil, 0, 0, nil)
	if err != nil {
		return nil, err
	}
	r := &Reader{
		query:  query,
		log:    logp.NewLogger("perfmon"),
		config: config,
		event:  event,
	}
	r.mapCounters(config)
	_, err = r.getCounterPaths()
	if err != nil {
		return nil, err
	}
	return r, nil
}

// RefreshCounterPaths will recheck for any new instances and add them to the counter list
func (re *Reader) RefreshCounterPaths() error {
	newCounters, err := re.getCounterPaths()
	if err != nil {
		return errors.Wrap(err, "failed retrieving counter paths")
	}
	err = re.query.RemoveUnusedCounters(newCounters)
	if err != nil {
		return errors.Wrap(err, "failed removing unused counter values")
	}
	return nil
}

// Read executes a query and returns those values in an event.
func (re *Reader) Read() ([]mb.Event, error) {
	// Some counters, such as rate counters, require two counter values in order to compute a displayable value. In this case we must call PdhCollectQueryData twice before calling PdhGetFormattedCounterValue.
	// For more information, see Collecting Performance Data (https://docs.microsoft.com/en-us/windows/desktop/PerfCtrs/collecting-performance-data).
	if err := re.query.CollectData(); err != nil {
		// users can encounter the case no counters are found (services/processes stopped), this should not generate an event with the error message,
		//could be the case the specific services are started after and picked up by the next RefreshCounterPaths func
		if err == pdh.PDH_NO_COUNTERS {
			re.log.Warnf("%s %v", collectFailedMsg, err)
		} else {
			return nil, errors.Wrap(err, collectFailedMsg)
		}
	}

	// Get the values.
	values, err := re.getValues()
	if err != nil {
		return nil, errors.Wrap(err, "failed formatting counter values")
	}
	var events []mb.Event
	// GroupAllCountersTo config option where counters for all instances are aggregated and instance count is added in the event under the string value provided by this option.
	if re.config.GroupAllCountersTo != "" {
		event := re.groupToSingleEvent(values)
		events = append(events, event)
	} else {
		events = re.groupToEvents(values)
	}
	return events, nil
}

func (re *Reader) getValues() (map[string][]pdh.CounterValue, error) {
	var val map[string][]pdh.CounterValue
	var sec uint32 = 1
	err := re.query.CollectDataEx(sec, re.event)
	if err != nil {
		return nil, err
	}
	waitFor, err := windows.WaitForSingleObject(re.event, windows.INFINITE)
	if err != nil {
		return nil, err
	}
	switch waitFor {
	case windows.WAIT_OBJECT_0:
		val, err = re.query.GetFormattedCounterValues()
		if err != nil {
			return nil, err
		}
	case windows.WAIT_FAILED:
		return nil, errors.New("WaitForSingleObject has failed")
	default:
		return nil, errors.New("WaitForSingleObject was abandoned or still waiting for completion")
	}
	return val, err
}

func randSeq(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// Close will close the PDH query for now.
func (re *Reader) Close() error {
	defer windows.CloseHandle(re.event)
	return re.query.Close()
}

// getCounterPaths func will process the counter paths based on the configuration options entered
func (re *Reader) getCounterPaths() ([]string, error) {
	var newCounters []string
	for i, counter := range re.counters {
		re.counters[i].ChildQueries = []string{}
		childQueries, err := re.query.GetCounterPaths(counter.QueryName)
		if err != nil {
			if re.config.IgnoreNECounters {
				switch err {
				case pdh.PDH_CSTATUS_NO_COUNTER, pdh.PDH_CSTATUS_NO_COUNTERNAME,
					pdh.PDH_CSTATUS_NO_INSTANCE, pdh.PDH_CSTATUS_NO_OBJECT:
					re.log.Infow("Ignoring non existent counter", "error", err,
						logp.Namespace("perfmon"), "query", counter.QueryName)
					continue
				}
			} else {
				return newCounters, errors.Wrapf(err, `failed to expand counter (query="%v")`, counter.QueryName)
			}
		}
		newCounters = append(newCounters, childQueries...)
		// there are cases when the ExpandWildCardPath will retrieve a successful status but not an expanded query so we need to check for the size of the list
		if err == nil && len(childQueries) >= 1 && !strings.Contains(childQueries[0], "*") {
			for _, v := range childQueries {
				if err := re.query.AddCounter(v, counter.InstanceName, counter.Format, isWildcard(childQueries, counter.InstanceName)); err != nil {
					return newCounters, errors.Wrapf(err, "failed to add counter (query='%v')", counter.QueryName)
				}
				re.counters[i].ChildQueries = append(re.counters[i].ChildQueries, v)
			}
		}
	}
	return newCounters, nil
}

func (re *Reader) getCounter(query string) (bool, PerfCounter) {
	for _, counter := range re.counters {
		for _, childQuery := range counter.ChildQueries {
			if childQuery == query {
				return true, counter
			}
		}
	}
	return false, PerfCounter{}
}

func (re *Reader) mapCounters(config Config) {
	re.counters = []PerfCounter{}
	if len(config.Queries) > 0 {
		for _, query := range config.Queries {
			for _, counter := range query.Counters {
				// counter paths can also not contain any instances
				if len(query.Instance) == 0 {
					re.counters = append(re.counters, PerfCounter{
						InstanceField: defaultInstanceField,
						InstanceName:  "",
						QueryField:    mapCounterPathLabel(query.Namespace, counter.Field, counter.Name),
						QueryName:     mapQuery(query.Name, "", counter.Name),
						Format:        counter.Format,
						ObjectName:    query.Name,
						ObjectField:   mapObjectName(query.Field),
					})
				} else {
					for _, instance := range query.Instance {
						re.counters = append(re.counters, PerfCounter{
							InstanceField: defaultInstanceField,
							InstanceName:  instance,
							QueryField:    mapCounterPathLabel(query.Namespace, counter.Field, counter.Name),
							QueryName:     mapQuery(query.Name, instance, counter.Name),
							Format:        counter.Format,
							ObjectName:    query.Name,
							ObjectField:   mapObjectName(query.Field),
						})
					}
				}
			}
		}
	}
}

func mapObjectName(objectField string) string {
	if objectField != "" {
		return objectField
	}
	return defaultObjectField
}

func mapQuery(obj string, instance string, path string) string {
	var query string
	// trim object
	obj = strings.TrimPrefix(obj, "\\")
	obj = strings.TrimSuffix(obj, "\\")
	query = fmt.Sprintf("\\%s", obj)

	if instance != "" {
		// trim instance
		instance = strings.TrimPrefix(instance, "(")
		instance = strings.TrimSuffix(instance, ")")
		query += fmt.Sprintf("(%s)", instance)
	}

	if strings.HasPrefix(path, "\\") {
		query += path
	} else {
		query += fmt.Sprintf("\\%s", path)
	}
	return query
}

func mapCounterPathLabel(namespace string, label string, path string) string {
	if label == "" {
		label = path
	}
	// replace spaces with underscores
	// replace backslashes with "per"
	// replace actual percentage symbol with the symbol "pct"
	r := strings.NewReplacer(" ", "_", "/sec", "_per_sec", "/_sec", "_per_sec", "\\", "_", "_%_", "_pct_", ":", "_", "_-_", "_")
	label = r.Replace(label)
	// replace uppercases with underscores
	label = replaceUpperCase(label)

	//  avoid cases as this "logicaldisk_avg._disk_sec_per_transfer"
	obj := strings.Split(label, ".")
	for index := range obj {
		// in some cases a trailing "_" is found
		obj[index] = strings.TrimPrefix(obj[index], "_")
		obj[index] = strings.TrimSuffix(obj[index], "_")
	}
	label = strings.ToLower(strings.Join(obj, "_"))
	label = strings.Replace(label, "__", "_", -1)
	return namespace + "." + label
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

// isWildcard function checks if users has configured a wildcard inside the instance configuration option and if the wildcard has been resulted in a valid number of queries
func isWildcard(queries []string, instance string) bool {
	if len(queries) > 1 {
		return true
	}
	if len(queries) == 1 && strings.Contains(instance, "*") {
		return true
	}
	return false
}
