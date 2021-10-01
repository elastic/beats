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

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/paths"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/fearful-symmetry/gorapl/rapl"
	"github.com/pkg/errors"
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

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The linux rapl metricset is beta.")

	config := config{}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	CPUList, err := getMSRCPUs()
	if err != nil {
		return nil, errors.Wrap(err, "error getting list of CPUs to query")
	}

	// check to see if msr-safe is installed
	if config.UseMSRSafe {
		queryPath := filepath.Join(paths.Paths.Hostfs, "/dev/cpu/", fmt.Sprint(CPUList[0]), "msr_safe")
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
		formatPath := filepath.Join(paths.Paths.Hostfs, "/dev/cpu/%d")
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

	// Get initial time values

	return &MetricSet{
		BaseMetricSet: base,
		handlers:      handlers,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	report.Event(mb.Event{
		MetricSetFields: common.MapStr{},
	})

	return nil
}

func (m *MetricSet) updatePower() map[int]map[rapl.RAPLDomain]float64 {
	newEnergy := map[int]map[rapl.RAPLDomain]float64{}
	for cpu, handler := range m.handlers {
		domainList := map[rapl.RAPLDomain]float64{}
		for _, domain := range handler.GetDomains() {
			joules, err := handler.ReadEnergyStatus(domain)
			// This is a bit hard to check for, as many of the registers are model-specific
			// Unless we want a map of every CPU we want to maintain, we sort of have to play it fast and loose.
			if err == rapl.ErrMSRDoesNotExist {
				continue
			}
			if err != nil {
				logp.L().Infof("Error reading MSR from domain %s: %s skipping.", domain, err)
				continue
			}
			domainList[domain] = joules
		}
		newEnergy[cpu] = domainList
	}

	return newEnergy
}

// getMSRCPUs forms a list of MSR paths to query
// For multi-processor systems, this will be more than 1.
func getMSRCPUs() ([]int, error) {
	CPUs, err := topoPkgCPUMap()
	if err != nil {
		return nil, errors.Wrap(err, "error fetching CPU topology")
	}
	coreList := []int{}
	for _, cores := range CPUs {
		coreList = append(coreList, cores[0])
	}

	return coreList, nil
}

//I'm not really sure how portable this algo is
//it is, however, the simplest way to do this. The intel power gaget iterates through each CPU using affinity masks, and runs `cpuid` in a loop to
//figure things out
//This uses /sys/devices/system/cpu/cpu*/topology/physical_package_id, which is what lscpu does. I *think* geopm does something similar to this.
func topoPkgCPUMap() (map[int][]int, error) {

	sysdir := "/sys/devices/system/cpu/"
	cpuMap := make(map[int][]int)

	files, err := ioutil.ReadDir(sysdir)
	if err != nil {
		return nil, err
	}

	re := regexp.MustCompile("cpu[0-9]+")

	for _, file := range files {
		if file.IsDir() && re.MatchString(file.Name()) {

			fullPkg := filepath.Join(sysdir, file.Name(), "/topology/physical_package_id")
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
