// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build windows

package website

import (
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/iis"
	"strings"

"github.com/elastic/beats/v7/libbeat/common"
"github.com/elastic/beats/v7/metricbeat/helper/windows/pdh"
"github.com/elastic/go-sysinfo"

"github.com/pkg/errors"

"github.com/elastic/beats/v7/libbeat/logp"
"github.com/elastic/beats/v7/metricbeat/mb"
)

const ecsProcessId = "process.pid"

// Reader will contain the config options
type WebsiteReader struct {
	query            pdh.Query    // PDH Query
	executed         bool         // Indicates if the query has been executed.
	log              *logp.Logger //
	config           iis.Config       // Metricset configuration
}


var websiteCounters = map[string]string{
		"network.total_bytes_received":      "\\Web Service(*)\\Total Bytes Received",
		"network.total_bytes_sent":          "\\Web Service(*)\\Total Bytes Sent",
		"network.bytes_sent_per_sec":        "\\Web Service(*)\\Bytes Sent/sec",
		"network.bytes_received_per_sec":    "\\Web Service(*)\\Bytes Received/sec",
		"network.current_connections":       "\\Web Service(*)\\Current Connections",
		"network.maximum_connections":       "\\Web Service(*)\\Maximum Connections",
		"network.total_connection_attempts": "\\Web Service(*)\\Total Connection Attempts (all instances)",
		"network.total_get_requests":        "\\Web Service(*)\\Total Get Requests",
		"network.get_requests_per_sec":      "\\Web Service(*)\\Get Requests/sec",
		"network.total_post_requests":       "\\Web Service(*)\\Total Post Requests",
		"network.post_requests_per_sec":     "\\Web Service(*)\\Post Requests/sec",
		"network.total_delete_requests":     "\\Web Service(*)\\Total Delete Requests",
		"network.delete_requests_per_sec":   "\\Web Service(*)\\Delete Requests/sec",
		"network.service_uptime":            "\\Web Service(*)\\Service Uptime",
		"network.total_put_requests":        "\\Web Service(*)\\Total PUT Requests",
		"network.put_requests_per_sec":      "\\Web Service(*)\\PUT Requests/sec",
	}


// newReader creates a new instance of Reader.
func NewReader(config iis.Config) (*WebsiteReader, error) {
	var query pdh.Query
	if err := query.Open(); err != nil {
		return nil, err
	}
	r := &WebsiteReader{
		query:           query,
		log:             logp.NewLogger("iis"),
		config:          config,
	}

	err := r.InitCounters()
	if err != nil {
		return nil, errors.Wrap(err, "error loading counters for existing app pools")
	}
	return r, nil
}

// initAppPools will check for any new instances and add them to the counter list
func (r *WebsiteReader) InitCounters() error {

	for key, value := range appPoolCounters {
		childQueries, err := r.query.GetCounterPaths(value)
		if err != nil {
			if err == pdh.PDH_CSTATUS_NO_COUNTER || err == pdh.PDH_CSTATUS_NO_COUNTERNAME || err == pdh.PDH_CSTATUS_NO_INSTANCE || err == pdh.PDH_CSTATUS_NO_OBJECT {
				r.log.Infow("Ignoring non existent counter", "error", err,
					logp.Namespace("application pool"), "query", value)
			} else {
				r.log.Error(err, `failed to expand counter path (query= "%v")`, value)
			}
			continue
		}
		newQueries = append(newQueries, childQueries...)
		// check if the pdhexpandcounterpath/pdhexpandwildcardpath functions have expanded the counter successfully.
		if len(childQueries) == 0 || (len(childQueries) == 1 && strings.Contains(childQueries[0], "*")) {
			// covering cases when PdhExpandWildCardPathW returns no counter paths or is unable to expand and the ignore_non_existent_counters flag is set
			r.log.Debugw("No counter paths returned but PdhExpandWildCardPathW returned no errors", "initial query", value,
				logp.Namespace("perfmon"), "expanded query", childQueries)
			continue
		}
		for _, v := range childQueries {
			if err := r.query.AddCounter(v, "", "float", len(childQueries) > 1); err != nil {
				return errors.Wrapf(err, `failed to add counter (query="%v")`, v)
			}
			r.workerProcesses[v] = key
		}
	}
	err = r.query.RemoveUnusedCounters(newQueries)
	if err != nil {
		return errors.Wrap(err, "failed removing unused counter values")
	}
	return nil
}

// read executes a query and returns those values in an event.
func (r *WebsiteReader) Read() ([]mb.Event, error) {
	if len(r.applicationPools) == 0 {
		r.executed = true
		return nil, nil
	}

	// Some counters, such as rate counters, require two counter values in order to compute a displayable value. In this case we must call PdhCollectQueryData twice before calling PdhGetFormattedCounterValue.
	// For more information, see Collecting Performance Data (https://docs.microsoft.com/en-us/windows/desktop/PerfCtrs/collecting-performance-data).
	if err := r.query.CollectData(); err != nil {
		return nil, errors.Wrap(err, "failed querying counter values")
	}

	// Get the values.
	values, err := r.query.GetFormattedCounterValues()
	if err != nil {
		r.query.Close()
		return nil, errors.Wrap(err, "failed formatting counter values")
	}
	var events []mb.Event
	eventGroup := r.mapEvents(values)
	r.executed = true
	results := make([]mb.Event, 0, len(events))
	for _, val := range eventGroup {
		results = append(results, val)
	}
	return results, nil
}

func (r *WebsiteReader) mapEvents(values map[string][]pdh.CounterValue) map[string]mb.Event {
	workers := getProcessIds(values)
	events := make(map[string]mb.Event)
	for _, appPool := range r.applicationPools {
		events[appPool.name] = mb.Event{
			MetricSetFields: common.MapStr{
				"name": appPool.name,
			},
			RootFields: common.MapStr{},
		}
		for counterPath, value := range values {
			for _, val := range value {
				// Some counters, such as rate counters, require two counter values in order to compute a displayable value. In this case we must call PdhCollectQueryData twice before calling PdhGetFormattedCounterValue.
				// For more information, see Collecting Performance Data (https://docs.microsoft.com/en-us/windows/desktop/PerfCtrs/collecting-performance-data).
				if val.Err.Error != nil {
					if !r.executed {
						continue
					}
					// The counter has a negative value or the counter was successfully found, but the data returned is not valid.
					// This error can occur if the counter value is less than the previous value. (Because counter values always increment, the counter value rolls over to zero when it reaches its maximum value.)
					// This is not an error that stops the application from running successfully and a positive counter value should be retrieved in the later calls.
					if val.Err.Error == pdh.PDH_CALC_NEGATIVE_VALUE || val.Err.Error == pdh.PDH_INVALID_DATA {
						r.log.Debugw("Counter value retrieval returned",
							"error", val.Err.Error, "cstatus", pdh.PdhErrno(val.Err.CStatus), logp.Namespace("application_pool"), "query", counterPath)
						continue
					}
				}
				if hasWorkerProcess(val.Instance, workers, appPool.workerProcessIds) {
					if r.workerProcesses[counterPath] == ecsProcessId {
						events[appPool.name].RootFields.Put(r.workerProcesses[counterPath], val.Measurement)
					} else if len(r.workerProcesses[counterPath]) != 0 {
						events[appPool.name].MetricSetFields.Put(r.workerProcesses[counterPath], val.Measurement)
					}
				}
			}

		}
	}
	return events
}

// close will close the PDH query for now.
func (r *WebsiteReader) Close() error {
	return r.query.Close()
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
		applicationPools = append(applicationPools, ApplicationPool{name: key, workerProcessIds: value})
	}
	if len(names) == 0 {
		return applicationPools, nil
	}
	var filtered []ApplicationPool
	for _, n := range names {
		for _, w3 := range applicationPools {
			if n == w3.name {
				filtered = append(filtered, w3)
			}
		}
	}
	return filtered, nil
}

// getw3wpProceses func retrieves the running w3wp process ids.
// A worker process is a windows process (w3wp.exe) which runs Web applications,
// and is responsible for handling requests sent to a Web Server for a specific application pool.
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
						break
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
			workers = append(workers, WorkerProcess{instanceName: values[0].Instance, processId: int(values[0].Measurement.(float64))})
		}
	}
	return workers
}

// hasWorkerProcess func checks if worker process list contains the process id
func hasWorkerProcess(instance string, workers []WorkerProcess, pids []int) bool {
	for _, worker := range workers {
		if worker.instanceName == instance {
			for _, pid := range pids {
				if pid == worker.processId {
					return true
				}
			}
		}
	}
	return false
}

