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

package website

import (
	"encoding/xml"
	"github.com/StackExchange/wmi"
	"github.com/elastic/beats/metricbeat/module/iis"
	"github.com/elastic/go-sysinfo"
	"github.com/elastic/go-sysinfo/types"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/helper/windows/pdh"
	"github.com/elastic/beats/metricbeat/mb"
)

// Reader will contain the config options
type Reader struct {
	Query           pdh.Query    // PDH Query
	Websites        []Website    // Mapping of counter path to key used for the label (e.g. processor.name)
	log             *logp.Logger // logger
	hasRun          bool         // will check if the reader has run a first time
	WorkerProcesses map[string]string
}

type Website struct {
	Name             string
	WorkerProcessIds []int
	ApplicationName  string
	counters         map[string]string
}

type WorkerProcess struct {
	ProcessId    int
	InstanceName string
	counters     map[string]string
}

type Application struct {
	ApplicationPool string
	SiteName        string
}
type Worker struct {
	AppPoolName string
	ProcessId   int
}

// NewReader creates a new instance of Reader.
func NewReader() (*Reader, error) {
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

func (re *Reader) InitCounters() error {
	websites, err := GetWebsites1()
	if err != nil {
		return err
	}
	re.Websites = websites
	re.WorkerProcesses = make(map[string]string)
	var newQueries []string
	for i, instance := range re.Websites {
		re.Websites[i].counters = make(map[string]string)
		for key, value := range iis.WebsiteCounters {
			value = strings.Replace(value, "*", instance.Name, 1)
			if err := re.Query.AddCounter(value, "", "float", true); err != nil {
				return errors.Wrapf(err, `failed to add counter (query="%v")`, value)
			}
			newQueries = append(newQueries, value)
			re.Websites[i].counters[value] = key
		}
	}
	for key, value := range iis.AppPoolCounters {
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

// Read executes a query and returns those values in an event.
func (re *Reader) Fetch() ([]mb.Event, error) {
	// if the ignore_non_existent_counters flag is set and no valid counter paths are found the Read func will still execute, a check is done before
	if len(re.Query.Counters) == 0 {
		return nil, errors.New("no counters to read")
	}

	// refresh performance counter list
	// Some counters, such as rate counters, require two counter values in order to compute a displayable value. In this case we must call PdhCollectQueryData twice before calling PdhGetFormattedCounterValue.
	// For more information, see Collecting Performance Data (https://docs.microsoft.com/en-us/windows/desktop/PerfCtrs/collecting-performance-data).
	// A flag is set if the second call has been executed else refresh will fail (reader.executed)
	if re.hasRun {
		err := re.InitCounters()
		if err != nil {
			return nil, errors.Wrap(err, "failed retrieving counters")
		}
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
	for _, host := range re.Websites {
		events[host.Name] = mb.Event{
			MetricSetFields: common.MapStr{
				"name":             host.Name,
				"application_pool": host.ApplicationName,
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
				if val.Instance == host.Name {
					events[host.Name].MetricSetFields.Put(host.counters[counterPath], val.Measurement)
				} else if hasWorkerProcess(val.Instance, workers, host.WorkerProcessIds) {
					events[host.Name].MetricSetFields.Put(re.WorkerProcesses[counterPath], val.Measurement)
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
func (re *Reader) Close() error {
	return re.Query.Close()
}

func getProcessIds(counterValues map[string][]pdh.CounterValue) []WorkerProcess {
	var workers []WorkerProcess
	for key, values := range counterValues {
		if strings.Contains(key, "\\ID Process") {
			workers = append(workers, WorkerProcess{InstanceName: values[0].Instance, ProcessId: int(values[0].Measurement.(float64))})
		}
	}
	return workers
}

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

func GetWebsites() ([]Website, error) {
	var applications []Application
	err := wmi.QueryNamespace("Select ApplicationPool, SiteName from Application", &applications, "root\\webadministration")
	if err != nil {
		// Don't return from this error since the name space might exist.
	}
	var workerProcesses []Worker
	err = wmi.QueryNamespace("Select AppPoolName, ProcessId from WorkerProcess", &workerProcesses, "root\\webadministration")
	if err != nil {
		// Don't return from this error since the name space might exist.
	}
	var sites []Website
	for _, application := range applications {
		site := Website{Name: application.SiteName, ApplicationName: application.ApplicationPool}
		for _, wp := range workerProcesses {
			if wp.AppPoolName == application.ApplicationPool {
				site.WorkerProcessIds = append(site.WorkerProcessIds, wp.ProcessId)
			}
		}
		sites = append(sites, site)
	}
	return sites, nil
}

func GetWebsites1() ([]Website, error) {
	path := os.Getenv("windir")
	body, err := ioutil.ReadFile(path + "/system32/inetsrv/config/applicationHost.config")
	if err != nil {
		return nil, err
	}
	var config Configuration
	err = xml.Unmarshal(body, &config)
	if err != nil {
		return nil, err
	}
	var websites []Website
	for _, site := range config.SystemApplicationHost.Sites.Site {
		if site.Name != "Default Web Site" {
			websites = append(websites, Website{Name: site.Name, ApplicationName: site.Application.ApplicationPool})
		}
	}
	processes, err := sysinfo.Processes()
	if err != nil {
		return nil, err
	}
	var filt []types.Process
	for _, pro := range processes {
		info, err := pro.Info()
		if err != nil {
			return nil, err
		}
		if info.Name == "w3wp.exe" {
			for i, si := range websites {
				if getapp(info.Args) == si.ApplicationName {
					websites[i].WorkerProcessIds = append(websites[i].WorkerProcessIds, info.PID)
				}
			}
			filt = append(filt, pro)
		}
	}
	return websites, nil
}

func getapp(as []string) string {
	for i, sd := range as {
		if sd == "-ap" {
			return as[i+1]
		}
	}
	return ""

}

type Configuration struct {
	SystemApplicationHost struct {
		Sites struct {
			Site []struct {
				Text        string `xml:",chardata"`
				Name        string `xml:"name,attr"`
				ID          string `xml:"id,attr"`
				Application struct {
					Text            string `xml:",chardata"`
					Path            string `xml:"path,attr"`
					ApplicationPool string `xml:"applicationPool,attr"`
				} `xml:"application"`
			} `xml:"site"`
		} `xml:"sites"`
	} `xml:"system.applicationHost"`
}
