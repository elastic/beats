// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package application_pool

import (
	"errors"
	"fmt"
	"strings"
	"syscall"

	"github.com/elastic/beats/v7/metricbeat/helper/windows/pdh"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/go-sysinfo"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/logp"
)

const ecsProcessId = "process.pid"

// Reader will contain the config options
type Reader struct {
	applicationPools []ApplicationPool
	workerProcesses  map[string]string
	query            pdh.Query    // PDH Query
	executed         bool         // Indicates if the query has been executed.
	log              *logp.Logger //
	config           Config       // Metricset configuration
}

// ApplicationPool struct contains the list of applications and their worker processes
type ApplicationPool struct {
	name             string
	workerProcessIds []int
}

// WorkerProcess struct contains the worker process details
type WorkerProcess struct {
	processId    int
	instanceName string
}

var appPoolCounters = map[string]string{
	"process.pid":                         "\\Process(w3wp*)\\ID Process",
	"process.cpu_usage_perc":              "\\Process(w3wp*)\\% Processor Time",
	"process.handle_count":                "\\Process(w3wp*)\\Handle Count",
	"process.thread_count":                "\\Process(w3wp*)\\Thread Count",
	"process.working_set":                 "\\Process(w3wp*)\\Working Set",
	"process.private_bytes":               "\\Process(w3wp*)\\Private Bytes",
	"process.virtual_bytes":               "\\Process(w3wp*)\\Virtual Bytes",
	"process.page_faults_per_sec":         "\\Process(w3wp*)\\Page Faults/sec",
	"process.io_read_operations_per_sec":  "\\Process(w3wp*)\\IO Read Operations/sec",
	"process.io_write_operations_per_sec": "\\Process(w3wp*)\\IO Write Operations/sec",

	// .NET CLR Memory
	"net_clr.memory.bytes_in_all_heaps":      "\\.NET CLR Memory(w3wp*)\\# Bytes in all Heaps",
	"net_clr.memory.gen_0_collections":       "\\.NET CLR Memory(w3wp*)\\# Gen 0 Collections",
	"net_clr.memory.gen_1_collections":       "\\.NET CLR Memory(w3wp*)\\# Gen 1 Collections",
	"net_clr.memory.gen_2_collections":       "\\.NET CLR Memory(w3wp*)\\# Gen 2 Collections",
	"net_clr.memory.total_committed_bytes":   "\\.NET CLR Memory(w3wp*)\\# Total committed Bytes",
	"net_clr.memory.allocated_bytes_per_sec": "\\.NET CLR Memory(w3wp*)\\Allocated Bytes/sec",
	"net_clr.memory.gen_0_heap_size":         "\\.NET CLR Memory(w3wp*)\\Gen 0 heap size",
	"net_clr.memory.gen_1_heap_size":         "\\.NET CLR Memory(w3wp*)\\Gen 1 heap size",
	"net_clr.memory.gen_2_heap_size":         "\\.NET CLR Memory(w3wp*)\\Gen 2 heap size",
	"net_clr.memory.large_object_heap_size":  "\\.NET CLR Memory(w3wp*)\\Large Object Heap size",
	"net_clr.memory.time_in_gc_perc":         "\\.NET CLR Memory(w3wp*)\\% Time in GC",

	// .NET CLR Exceptions
	"net_clr.total_exceptions_thrown":      "\\.NET CLR Exceptions(w3wp*)\\# of Exceps Thrown",
	"net_clr.exceptions_thrown_per_sec":    "\\.NET CLR Exceptions(w3wp*)\\# of Exceps Thrown / sec",
	"net_clr.filters_per_sec":              "\\.NET CLR Exceptions(w3wp*)\\# of Filters / sec",
	"net_clr.finallys_per_sec":             "\\.NET CLR Exceptions(w3wp*)\\# of Finallys / sec",
	"net_clr.throw_to_catch_depth_per_sec": "\\.NET CLR Exceptions(w3wp*)\\Throw To Catch Depth / sec",

	// .NET CLR LocksAndThreads
	"net_clr.locks_and_threads.contention_rate_per_sec": "\\.NET CLR LocksAndThreads(w3wp*)\\Contention Rate / sec",
	"net_clr.locks_and_threads.current_queue_length":    "\\.NET CLR LocksAndThreads(w3wp*)\\Current Queue Length",
}

// newReader creates a new instance of Reader.
func newReader(config Config) (*Reader, error) {
	var query pdh.Query
	if err := query.Open(); err != nil {
		return nil, err
	}
	r := &Reader{
		query:           query,
		log:             logp.NewLogger("application_pool"),
		config:          config,
		workerProcesses: make(map[string]string),
	}

	err := r.initAppPools()
	if err != nil {
		return nil, fmt.Errorf("error loading counters for existing app pools: %w", err)
	}
	return r, nil
}

// initAppPools will check for any new instances and add them to the counter list
func (r *Reader) initAppPools() error {
	apps, err := getApplicationPools(r.config.Names)
	if err != nil {
		return fmt.Errorf("failed retrieving running worker processes: %w", err)
	}
	r.applicationPools = apps
	if len(apps) == 0 {
		r.log.Info("no running application pools found")
		return nil
	}
	// Helper function to identify known PDH errors, such as missing counters or instances.
	// These errors are expected in certain cases (e.g. "No Managed Code" environments).
	isPDHError := func(err error) bool {
		return errors.Is(err, pdh.PdhErrno(syscall.ERROR_NOT_FOUND)) ||
			errors.Is(err, pdh.PDH_CSTATUS_NO_COUNTER) ||
			errors.Is(err, pdh.PDH_CSTATUS_NO_COUNTERNAME) ||
			errors.Is(err, pdh.PDH_CSTATUS_NO_INSTANCE) ||
			errors.Is(err, pdh.PDH_CSTATUS_NO_OBJECT)
	}
	var newQueries []string
	r.workerProcesses = make(map[string]string)
	for key, value := range appPoolCounters {
		childQueries, err := r.query.GetCounterPaths(value)
		if err != nil {
			// Handle known PDH errors as informational (e.g. missing counters).
			if isPDHError(err) {
				r.log.Infow("Ignoring non existent counter", "error", err,
					logp.Namespace("application pool"), "query", value,
				)
			} else {
				r.log.Errorf(`failed to expand counter path (query= "%v"): %w`, value, err)
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
				return fmt.Errorf(`failed to add counter (query="%v"): %w`, v, err)
			}
			r.workerProcesses[v] = key
		}
	}
	err = r.query.RemoveUnusedCounters(newQueries)
	if err != nil {
		return fmt.Errorf("failed removing unused counter values: %w", err)
	}
	return nil
}

// read executes a query and returns those values in an event.
func (r *Reader) read() ([]mb.Event, error) {
	if len(r.applicationPools) == 0 {
		r.executed = true
		return nil, nil
	}

	// Some counters, such as rate counters, require two counter values in order to compute a displayable value. In this case we must call PdhCollectQueryData twice before calling PdhGetFormattedCounterValue.
	// For more information, see Collecting Performance Data (https://docs.microsoft.com/en-us/windows/desktop/PerfCtrs/collecting-performance-data).
	if err := r.query.CollectData(); err != nil {
		return nil, fmt.Errorf("failed querying counter values: %w", err)
	}

	// Get the values.
	values, err := r.query.GetFormattedCounterValues()
	if err != nil {
		r.close()
		return nil, fmt.Errorf("failed formatting counter values: %w", err)
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

func (r *Reader) mapEvents(values map[string][]pdh.CounterValue) map[string]mb.Event {
	workers := getProcessIds(values)
	events := make(map[string]mb.Event)
	for _, appPool := range r.applicationPools {
		events[appPool.name] = mb.Event{
			MetricSetFields: mapstr.M{
				"name": appPool.name,
			},
			RootFields: mapstr.M{},
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
					if errors.Is(val.Err.Error, pdh.PDH_CALC_NEGATIVE_VALUE) || errors.Is(val.Err.Error, pdh.PDH_INVALID_DATA) {
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
func (r *Reader) close() error {
	return r.query.Close()
}

// getApplicationPools method retrieves the w3wp.exe processes and the application pool name, also filters on the application pool names configured by users
func getApplicationPools(names []string) ([]ApplicationPool, error) {
	processes, err := getw3wpProceses()
	if err != nil {
		return nil, err
	}
	appPools := make(map[string][]int)
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
		if strings.Contains(key, "\\ID Process") && values[0].Measurement != nil {
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
