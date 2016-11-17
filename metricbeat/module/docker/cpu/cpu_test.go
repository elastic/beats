package cpu

import (
	"reflect"
	"testing"
	"time"

	dc "github.com/fsouza/go-dockerclient"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/module/docker"
)

var cpuService CPUService
var statsList = make([]dc.Stats, 3)

func TestCPUService_PerCpuUsage(t *testing.T) {
	oldPerCpuValuesTest := [][]uint64{{1, 9, 9, 5}, {1, 2, 3, 4}, {0, 0, 0, 0}}
	newPerCpuValuesTest := [][]uint64{{100000001, 900000009, 900000009, 500000005}, {101, 202, 303, 404}, {0, 0, 0, 0}}
	for index := range statsList {
		statsList[index].PreCPUStats.CPUUsage.PercpuUsage = oldPerCpuValuesTest[index]
		statsList[index].CPUStats.CPUUsage.PercpuUsage = newPerCpuValuesTest[index]
	}
	testCase := []struct {
		given    dc.Stats
		expected common.MapStr
	}{
		{statsList[0], common.MapStr{
			"0": float64(0.10),
			"1": float64(0.90),
			"2": float64(0.90),
			"3": float64(0.50),
		}},
		{statsList[1], common.MapStr{
			"0": float64(0.0000001),
			"1": float64(0.0000002),
			"2": float64(0.0000003),
			"3": float64(0.0000004),
		}},
		{statsList[2], common.MapStr{
			"0": float64(0),
			"1": float64(0),
			"2": float64(0),
			"3": float64(0),
		}},
	}
	CPUService := NewCpuService()
	for _, tt := range testCase {
		out := CPUService.perCpuUsage(&tt.given)
		if !equalEvent(tt.expected, out) {
			t.Errorf("PerCpuUsage(%v) => %v, want %v", tt.given.CPUStats.CPUUsage.PercpuUsage, out, tt.expected)
		}
	}
}

func TestCPUService_TotalUsage(t *testing.T) {
	oldTotalValuesTest := []uint64{569832511, 50, 10}
	totalValuesTest := []uint64{45996245, 500000050, 10}
	for index := range statsList {
		statsList[index].PreCPUStats.CPUUsage.TotalUsage = oldTotalValuesTest[index]
		statsList[index].CPUStats.CPUUsage.TotalUsage = totalValuesTest[index]
	}
	testCase := []struct {
		given    dc.Stats
		expected float64
	}{
		{statsList[0], 0},
		{statsList[1], 0.50},
		{statsList[2], 0},
	}
	for _, tt := range testCase {
		out := cpuService.totalUsage(&tt.given)
		if out != tt.expected {
			t.Errorf("usageInKernelmode(%v) => %v, want %v", tt.given.CPUStats.CPUUsage.PercpuUsage, out, tt.expected)
		}
	}
}

func TestCPUService_UsageInKernelmode(t *testing.T) {
	usageOldValuesTest := []uint64{0, 10, 356985235698}
	usageValuesTest := []uint64{500000000, 500000010, 500000050}
	for index := range statsList {
		statsList[index].PreCPUStats.CPUUsage.UsageInKernelmode = usageOldValuesTest[index]
		statsList[index].CPUStats.CPUUsage.UsageInKernelmode = usageValuesTest[index]
	}
	testCase := []struct {
		given    dc.Stats
		expected float64
	}{
		{statsList[0], 0.50},
		{statsList[1], 0.50},
		{statsList[2], 0},
	}
	for _, tt := range testCase {
		out := cpuService.usageInKernelmode(&tt.given)
		if out != tt.expected {
			t.Errorf("usageInKernelmode(%v) => %v, want %v", tt.given, out, tt.expected)
		}
	}
}

func TestCPUService_UsageInUsermode(t *testing.T) {
	usageOldValuesTest := []uint64{0, 1958965, 500}
	usageValuesTest := []uint64{500000000, 50, 1000000500}
	for index := range statsList {
		statsList[index].PreCPUStats.CPUUsage.UsageInUsermode = usageOldValuesTest[index]
		statsList[index].CPUStats.CPUUsage.UsageInUsermode = usageValuesTest[index]
	}
	testCase := []struct {
		given    dc.Stats
		expected float64
	}{
		{statsList[0], 0.50},
		{statsList[1], 0},
		{statsList[2], 1},
	}
	for _, tt := range testCase {
		out := cpuService.usageInUsermode(&tt.given)
		if out != tt.expected {
			t.Errorf("usageInKernelmode(%v) => %v, want %v", tt.given, out, tt.expected)
		}
	}
}

//TestCPUService_GetCpuStats simulates the generation of a cpu event, it checks  :
// -The validity of the parameters sent to the different methods used to get the data calculated and the retuned values
//-The generated events are correctly formated

func TestCPUService_GetCpuStats(t *testing.T) {
	// GIVEN
	containerID := "containerID"
	labels := map[string]string{
		"label1": "val1",
		"label2": "val2",
	}
	container := dc.APIContainers{
		ID:         containerID,
		Image:      "image",
		Command:    "command",
		Created:    123789,
		Status:     "Up",
		Ports:      []dc.APIPort{{PrivatePort: 1234, PublicPort: 4567, Type: "portType", IP: "123.456.879.1"}},
		SizeRw:     123,
		SizeRootFs: 456,
		Names:      []string{"/name1", "name1/fake"},
		Labels:     labels,
		Networks:   dc.NetworkList{},
	}

	preCPUStats := getCPUStats([]uint64{1, 9, 9, 5}, []uint64{0, 50, 0})
	CPUStats := getCPUStats([]uint64{100000001, 900000009, 900000009, 500000005}, []uint64{500000000, 500000050, 500000000})

	//CPU stats
	stats := dc.Stats{}
	stats.Read = time.Now()
	stats.CPUStats = CPUStats
	stats.PreCPUStats = preCPUStats

	cpuStatsStruct := docker.DockerStat{}
	cpuStatsStruct.Container = container
	cpuStatsStruct.Stats = stats

	mockedCPUCalculator := getMockedCPUCalcul(1.0)
	// expected events : The generated event should be equal to the expected event
	expectedEvent := common.MapStr{
		"_module": common.MapStr{
			"container": common.MapStr{
				"id":     containerID,
				"name":   "name1",
				"socket": docker.GetSocket(),
				"labels": docker.BuildLabelArray(labels),
			},
		},
		"usage": common.MapStr{
			"per_cpu":     mockedCPUCalculator.PerCpuUsage(&stats),
			"total":       mockedCPUCalculator.TotalUsage(&stats),
			"kernel_mode": mockedCPUCalculator.UsageInKernelmode(&stats),
			"user_mode":   mockedCPUCalculator.UsageInUsermode(&stats),
		},
	}

	cpuData := cpuService.getCpuStats(&cpuStatsStruct)
	event := eventMapping(&cpuData)
	//THEN
	assert.True(t, equalEvent(expectedEvent, event))
}

func getMockedCPUCalcul(number float64) MockCPUCalculator {
	mockedCPU := MockCPUCalculator{}
	percpuUsage := common.MapStr{
		"0": float64(0.10),
		"1": float64(0.90),
		"2": float64(0.90),
		"3": float64(0.50),
	}
	mockedCPU.On("PerCpuUsage").Return(percpuUsage)
	mockedCPU.On("TotalUsage").Return(float64(0.50))
	mockedCPU.On("UsageInKernelmode").Return(float64(0.50))
	mockedCPU.On("UsageInUsermode").Return(float64(0.50))
	return mockedCPU
}
func equalEvent(expectedEvent common.MapStr, event common.MapStr) bool {

	return reflect.DeepEqual(expectedEvent, event)

}
func getCPUStats(perCPU []uint64, numbers []uint64) dc.CPUStats {
	return dc.CPUStats{
		CPUUsage: struct {
			PercpuUsage       []uint64 `json:"percpu_usage,omitempty" yaml:"percpu_usage,omitempty"`
			UsageInUsermode   uint64   `json:"usage_in_usermode,omitempty" yaml:"usage_in_usermode,omitempty"`
			TotalUsage        uint64   `json:"total_usage,omitempty" yaml:"total_usage,omitempty"`
			UsageInKernelmode uint64   `json:"usage_in_kernelmode,omitempty" yaml:"usage_in_kernelmode,omitempty"`
		}{
			PercpuUsage:       perCPU,
			UsageInUsermode:   numbers[0],
			TotalUsage:        numbers[1],
			UsageInKernelmode: numbers[2],
		},
		SystemCPUUsage: 0,
		ThrottlingData: struct {
			Periods          uint64 `json:"periods,omitempty"`
			ThrottledPeriods uint64 `json:"throttled_periods,omitempty"`
			ThrottledTime    uint64 `json:"throttled_time,omitempty"`
		}{
			Periods:          0,
			ThrottledPeriods: 0,
			ThrottledTime:    0,
		},
	}
}
