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

//go:build linux
// +build linux

package rapl

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/fearful-symmetry/gorapl/rapl"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v8/libbeat/logp"
	"github.com/elastic/beats/v8/libbeat/metric/system/resolve"
	"github.com/elastic/beats/v8/metricbeat/mb"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("linux", "rapl", New)
}

type config struct {
	UseMSRSafe bool `config:"rapl.use_msr_safe"`
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	handlers   map[int]rapl.RAPLHandler
	lastValues map[int]map[rapl.RAPLDomain]energyTrack
}

type energyTrack struct {
	joules float64
	time   time.Time
}

type energyUsage struct {
	joules float64
	watts  float64
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The linux rapl metricset is beta.")

	config := config{}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	sys := base.Module().(resolve.Resolver)
	CPUList, err := getMSRCPUs(sys)
	if err != nil {
		return nil, errors.Wrap(err, "error getting list of CPUs to query")
	}

	// check to see if msr-safe is installed
	if config.UseMSRSafe {
		queryPath := sys.ResolveHostFS(filepath.Join("/dev/cpu/", fmt.Sprint(CPUList[0]), "msr_safe"))
		_, err := os.Stat(queryPath)
		if errors.Is(err, os.ErrNotExist) {
			return nil, errors.New("no msr_safe device found. Is the kernel module loaded?")
		}
		if err != nil {
			return nil, errors.Wrapf(err, "could not check msr_safe device at %s", queryPath)
		}
	} else {
		user, err := user.Current()
		if err != nil {
			return nil, errors.Wrap(err, "error fetching user list")
		}
		if user.Uid != "0" {
			return nil, errors.New("linux/rapl must run as root if not using msr-safe")
		}
	}

	handlers := map[int]rapl.RAPLHandler{}
	for _, cpu := range CPUList {
		formatPath := sys.ResolveHostFS("/dev/cpu/%d")
		if config.UseMSRSafe {
			formatPath = filepath.Join(formatPath, "/msr_safe")
		} else {
			formatPath = filepath.Join(formatPath, "/msr")
		}
		handler, err := rapl.CreateNewHandler(cpu, formatPath)
		if err != nil {
			return nil, errors.Wrapf(err, "error creating handler at path %s for CPU %d", formatPath, cpu)
		}
		handlers[cpu] = handler

	}

	ms := &MetricSet{
		BaseMetricSet: base,
		handlers:      handlers,
	}

	ms.updatePower()

	return ms, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {

	watts := m.updatePower()

	for cpu, metric := range watts {
		evt := common.MapStr{
			"core": cpu,
		}
		for domain, power := range metric {
			evt[strings.ToLower(domain.Name)] = common.MapStr{
				"watts":  common.Round(power.watts, common.DefaultDecimalPlacesCount),
				"joules": common.Round(power.joules, common.DefaultDecimalPlacesCount),
			}
		}
		report.Event(mb.Event{
			MetricSetFields: evt,
		})
	}

	return nil
}

func (m *MetricSet) updatePower() map[int]map[rapl.RAPLDomain]energyUsage {
	newEnergy := make(map[int]map[rapl.RAPLDomain]energyTrack)
	powerUsage := make(map[int]map[rapl.RAPLDomain]energyUsage)

	for cpu, handler := range m.handlers {
		powerUsage[cpu] = make(map[rapl.RAPLDomain]energyUsage)
		domainList := map[rapl.RAPLDomain]energyTrack{}

		for _, domain := range handler.GetDomains() {
			joules, err := handler.ReadEnergyStatus(domain)
			// This is a bit hard to check for, as many of the registers are model-specific
			// Unless we want to maintain a map of every CPU, we sort of have to play it fast and loose.
			if err == rapl.ErrMSRDoesNotExist {
				continue
			}
			if err != nil {
				logp.L().Infof("Error reading MSR from domain %s: %s skipping.", domain, err)
				continue
			}
			domainList[domain] = energyTrack{joules: joules, time: time.Now()}
			// divide the delta of joules by the time interval to get watts
			if m.lastValues != nil {
				// This register can roll over. If/when it does, skip reporting
				if m.lastValues[cpu][domain].joules > joules {
					continue
				}
				delta := m.lastValues[cpu][domain].joules - joules
				timeDelta := m.lastValues[cpu][domain].time.Sub(domainList[domain].time)
				powerUsage[cpu][domain] = energyUsage{watts: delta / timeDelta.Seconds(), joules: joules}
			}
		}
		newEnergy[cpu] = domainList
	}

	m.lastValues = newEnergy
	if m.lastValues == nil {
		return nil
	}

	return powerUsage
}

// getMSRCPUs forms a list of CPU cores to query
// For multi-processor systems, this will be more than 1.
func getMSRCPUs(hostfs resolve.Resolver) ([]int, error) {
	CPUs, err := topoPkgCPUMap(hostfs)
	if err != nil {
		return nil, errors.Wrap(err, "error fetching CPU topology")
	}
	coreList := []int{}
	for _, cores := range CPUs {
		coreList = append(coreList, cores[0])
	}

	// if we don't have any cores, assume something has gone wrong
	if len(coreList) == 0 {
		return coreList, errors.New("no cores found")
	}

	return coreList, nil
}

//I'm not really sure how portable this algo is
//it is, however, the simplest way to do this. The intel power gadget iterates through each CPU using affinity masks, and runs `cpuid` in a loop to
//figure things out
//This uses /sys/devices/system/cpu/cpu*/topology/physical_package_id, which is what lscpu does. I *think* geopm does something similar to this.
func topoPkgCPUMap(hostfs resolve.Resolver) (map[int][]int, error) {

	sysdir := "/sys/devices/system/cpu/"
	cpuMap := make(map[int][]int)

	files, err := ioutil.ReadDir(hostfs.ResolveHostFS(sysdir))
	if err != nil {
		return nil, err
	}

	re := regexp.MustCompile("cpu[0-9]+")

	for _, file := range files {
		if file.IsDir() && re.MatchString(file.Name()) {

			fullPkg := hostfs.ResolveHostFS(filepath.Join(sysdir, file.Name(), "/topology/physical_package_id"))
			dat, err := ioutil.ReadFile(fullPkg)
			if err != nil {
				return nil, errors.Wrapf(err, "error reading file %s", fullPkg)
			}
			phys, err := strconv.ParseInt(strings.TrimSpace(string(dat)), 10, 64)
			if err != nil {
				return nil, errors.Wrapf(err, "error parsing value from %s", fullPkg)
			}
			var cpuCore int
			_, err = fmt.Sscanf(file.Name(), "cpu%d", &cpuCore)
			if err != nil {
				return nil, errors.Wrapf(err, "error fetching CPU core value from string %s", file.Name())
			}
			pkgList, ok := cpuMap[int(phys)]
			if !ok {
				cpuMap[int(phys)] = []int{cpuCore}
			} else {
				pkgList = append(pkgList, cpuCore)
				cpuMap[int(phys)] = pkgList
			}

		}
	}

	return cpuMap, nil
}
