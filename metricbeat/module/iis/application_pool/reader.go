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

package application_pool

import (
	"strings"

	"github.com/elastic/go-sysinfo"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/helper/windows/pdh"
	"github.com/elastic/beats/metricbeat/mb"
)

// Reader  strucr will contain the pdh query and config options
type Reader struct {
	Query            pdh.Query         // PDH Query
	ApplicationPools []ApplicationPool // Mapping of counter path to key used for the label (e.g. processor.name)
	log              *logp.Logger      // logger
	hasRun           bool              // will check if the reader has run a first time
	WorkerProcesses  map[string]string
}

// ApplicationPool struct contains the list of applications and their worker processes
type ApplicationPool struct {
	Name             string
	WorkerProcessIds []int
	counters         map[string]string
}

// WorkerProcess struct contains the worker process details
type WorkerProcess struct {
	ProcessId    int
	InstanceName string
}

var appPoolCounters = map[string]string{
	"worker_process_id":                    "\\Process(w3wp*)\\ID Process",
	"process.cpu_usage_perc":               "\\Process(w3wp*)\\% Processor Time",
	"process.handle_count":                 "\\Process(w3wp*)\\Handle Count",
	"process.thread_count":                 "\\Process(w3wp*)\\Thread Count",
	"process.working_set":                  "\\Process(w3wp*)\\Working Set",
	"process.private_byte":                 "\\Process(w3wp*)\\Private Bytes",
	"process.virtual_bytes":                "\\Process(w3wp*)\\Virtual Bytes",
	"process.page_faults_per_sec":          "\\Process(w3wp*)\\Page Faults/sec",
	"process.io_read_operations_per_sec":   "\\Process(w3wp*)\\IO Read Operations/sec",
	"process.io_write_operations_per_sec":  "\\Process(w3wp*)\\IO Write Operations/sec",
	"net_clr.total_exceptions_thrown":      "\\.NET CLR Exceptions(w3wp*)\\# of Exceps Thrown",
	"net_clr.exceptions_thrown_per_sec":    "\\.NET CLR Exceptions(w3wp*)\\# of Exceps Thrown / sec",
	"net_clr.filters_per_sec":              "\\.NET CLR Exceptions(w3wp*)\\# of Filters / sec",
	"net_clr.finallys_per_sec":             "\\.NET CLR Exceptions(w3wp*)\\# of Finallys / sec",
	"net_clr.throw_to_catch_depth_per_sec": "\\.NET CLR Exceptions(w3wp*)\\Throw To Catch Depth / sec",
}

// newReader creates a new instance of Reader.
func newReader() (*Reader, error) {
	var query pdh.Query
	if err := query.Open(); err != nil {
		return nil, err
	}
	reader := &Reader{
		Query: query,
		log:   logp.NewLogger("website"),
	}

	return reader, nil
}

// initCounters func retrieves the running application worker processes and adds the counters to the pdh query
func (re *Reader) initCounters(filtered []string) error {
	apps, err := getApplicationPools(filtered)
	if err != nil {
		return errors.Wrap(err, "failed retrieving running worker processes")
	}
	if len(apps) == 0 {
		re.log.Info("no running application pools found")
		return nil
	}
	re.ApplicationPools = apps
	re.WorkerProcesses = make(map[string]string)
	var newQueries []string
	for key, value := range appPoolCounters {
		counters, err := re.Query.ExpandWildCardPath(value)
		if err != nil {
			return errors.Wrapf(err, `failed to expand counter path (query="%v")`, value)
		}
		for _, count := range counters {
			if err = re.Query.AddCounter(count, "", "float", true); err != nil {
				return errors.Wrapf(err, `failed to add counter (query="%v")`, count)
			}
			newQueries = append(newQueries, count)
			re.WorkerProcesses[count] = key
		}
	}
	err = re.Query.RemoveUnusedCounters(newQueries)
	if err != nil {
		return errors.Wrap(err, "failed removing unused counter values")
	}
	return nil
}

// fetch executes collects the query data and maps the counter values to events.
func (re *Reader) fetch(names []string) ([]mb.Event, error) {
	// refresh performance counter list
	// Some counters, such as rate counters, require two counter values in order to compute a displayable value. In this case we must call PdhCollectQueryData twice before calling PdhGetFormattedCounterValue.
	// For more information, see Collecting Performance Data (https://docs.microsoft.com/en-us/windows/desktop/PerfCtrs/collecting-performance-data).
	// A flag is set if the second call has been executed else refresh will fail (reader.executed)
	if re.hasRun || len(re.Query.Counters) == 0 {
		err := re.initCounters(names)
		if err != nil {
			return nil, errors.Wrap(err, "failed retrieving counters")
		}
	}
	// if the ignore_non_existent_counters flag is set and no valid counter paths are found the Read func will still execute, a check is done before
	if len(re.Query.Counters) == 0 {
		return nil, nil
	}
	// Some counters, such as rate counters, require two counter values in order to compute a displayable value. In this case we must call PdhCollectQueryData twice before calling PdhGetFormattedCounterValue.
	// For more information, see Collecting Performance Data (https://docs.microsoft.com/en-us/windows/desktop/PerfCtrs/collecting-performance-data).
	if err := re.Query.CollectData(); err != nil {
		return nil, errors.Wrap(err, "failed querying counter values")
	}

	// Get the values.
	values, err := re.Query.GetFormattedCounterValues()
	if err != nil {
		return nil, errors.Wrap(err, "failed formatting counter values")
	}

	workers := getProcessIds(values)
	events := make(map[string]mb.Event)
	for _, appPool := range re.ApplicationPools {
		events[appPool.Name] = mb.Event{
			MetricSetFields: common.MapStr{
				"name": appPool.Name,
			},
		}
		for counterPath, value := range values {
			for _, val := range value {
				// Some counters, such as rate counters, require two counter values in order to compute a displayable value. In this case we must call PdhCollectQueryData twice before calling PdhGetFormattedCounterValue.
				// For more information, see Collecting Performance Data (https://docs.microsoft.com/en-us/windows/desktop/PerfCtrs/collecting-performance-data).
				if val.Err != nil && !re.hasRun {
					re.log.Debugw("Ignoring the first measurement because the data isn't ready",
						"error", val.Err, logp.Namespace("website"), "query", counterPath)
					continue
				}
				if val.Instance == appPool.Name {
					events[appPool.Name].MetricSetFields.Put(appPool.counters[counterPath], val.Measurement)
				} else if hasWorkerProcess(val.Instance, workers, appPool.WorkerProcessIds) {
					events[appPool.Name].MetricSetFields.Put(re.WorkerProcesses[counterPath], val.Measurement)
				}
			}

		}
	}

	re.hasRun = true
	results := make([]mb.Event, 0, len(events))
	for _, val := range events {
		results = append(results, val)
	}
	return results, nil
}

// Close will close the PDH query for now.
func (re *Reader) close() error {
	return re.Query.Close()
}

// getApplicationPools method retrieves the w3wp.exe processes and the application pool name, also filters on the application pool names configured by users
func getApplicationPools(names []string) ([]ApplicationPool, error) {
	processes, err := getw3wpProceses()
	if err != nil {
		return nil, err
	}
	var appPools = make(map[string][]int)
	for key, value := range processes {
		appPools[value] = append(appPools[value], key)
	}
	var applicationPools []ApplicationPool
	for key, value := range appPools {
		applicationPools = append(applicationPools, ApplicationPool{Name: key, WorkerProcessIds: value})
	}
	if len(names) == 0 {
		return applicationPools, nil
	}
	var filtered []ApplicationPool
	for _, n := range names {
		for _, w3 := range applicationPools {
			if n == w3.Name {
				filtered = append(filtered, w3)
			}
		}
	}
	return filtered, nil
}

// getw3wpProceses func retrieves the running w3wp process ids
func getw3wpProceses() (map[int]string, error) {
	processes, err := sysinfo.Processes()
	if err != nil {
		return nil, err
	}
	wps := make(map[int]string)
	for _, p := range processes {
		info, err := p.Info()
		if err != nil {
			continue
		}
		if info.Name == "w3wp.exe" {
			if len(info.Args) > 0 {
				for i, ar := range info.Args {
					if ar == "-ap" && len(info.Args) > i+1 {
						wps[info.PID] = info.Args[i+1]
						continue
					}
				}
			}
		}
	}
	return wps, nil
}

// getProcessIds func maps the process ids from the counter values to worker process obj
func getProcessIds(counterValues map[string][]pdh.CounterValue) []WorkerProcess {
	var workers []WorkerProcess
	for key, values := range counterValues {
		if strings.Contains(key, "\\ID Process") {
			workers = append(workers, WorkerProcess{InstanceName: values[0].Instance, ProcessId: int(values[0].Measurement.(float64))})
		}
	}
	return workers
}

// hasWorkerProcess func checks if workerprocess list contains the process id
func hasWorkerProcess(instance string, workers []WorkerProcess, pids []int) bool {
	for _, worker := range workers {
		if worker.InstanceName == instance {
			for _, pid := range pids {
				if pid == worker.ProcessId {
					return true
				}
			}
		}
	}
	return false
}
