package cpu

import (
	"reflect"
	"testing"

	dc "github.com/fsouza/go-dockerclient"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
)

func TestCPUService_PerCpuUsage(t *testing.T) {
	//GIVEN
	preCpuStats := getCPUStats([]uint64{1, 9, 9, 5}, []uint64{0, 0, 0})
	cpuStats := getCPUStats([]uint64{100000001, 900000009, 900000009, 500000005}, []uint64{0, 0, 0})
	CPUService := NewCpuService()
	stats := dc.Stats{}
	stats.CPUStats = cpuStats
	stats.PreCPUStats = preCpuStats
	// WHEN
	result := CPUService.PerCpuUsage(&stats)
	//THEN
	assert.Equal(t, common.MapStr{
		"0": float64(0.10),
		"1": float64(0.90),
		"2": float64(0.90),
		"3": float64(0.50),
	}, result)
}

func TestCPUService_TotalUsage(t *testing.T) {

	//GIVEN
	preCpuStats := getCPUStats(nil, []uint64{0, 50, 0})
	cpuStats := getCPUStats(nil, []uint64{0, 500000050, 0})
	CPUService := NewCpuService()

	stats := dc.Stats{}
	stats.CPUStats = cpuStats
	stats.PreCPUStats = preCpuStats
	//WHEN
	result := CPUService.TotalUsage(&stats)
	// THEN
	assert.Equal(t, 0.50, result)
}

func TestCPUService_UsageInKernelmode(t *testing.T) {
	//GIVEN
	preCpuStats := getCPUStats(nil, []uint64{0, 0, 0})
	cpuStats := getCPUStats(nil, []uint64{0, 0, 500000000})
	CPUService := NewCpuService()

	stats := dc.Stats{}
	stats.CPUStats = cpuStats
	stats.PreCPUStats = preCpuStats
	//WHEN
	result := CPUService.UsageInKernelmode(&stats)
	//THEN
	assert.Equal(t, float64(0.50), result)
}

func TestCPUService_UsageInUsermode(t *testing.T) {
	//GIVEN
	preCpuStats := getCPUStats(nil, []uint64{0, 0, 0})
	cpuStats := getCPUStats(nil, []uint64{500000000, 0, 0})
	CPUService := NewCpuService()

	stats := dc.Stats{}
	stats.CPUStats = cpuStats
	stats.PreCPUStats = preCpuStats
	//WHEN
	result := CPUService.UsageInUsermode(&stats)
	//  THEN
	assert.Equal(t, float64(0.50), result)
}

/* TODO: uncomment
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

	stats := dc.Stats{}
	stats.Read = time.Now()
	stats.CPUStats = CPUStats
	stats.PreCPUStats = preCPUStats

	cpuStatsStruct := docker.DockerStat{}
	cpuStatsStruct.Container = container
	cpuStatsStruct.Stats = stats

	mockedCPUCalculator := getMockedCPUCalcul(1.0)
	// expected events
	expectedEvent := common.MapStr{
		"@timestamp": common.Time(stats.Read),
		"container": common.MapStr{
			"id":     containerID,
			"name":   "name1",
			"labels": docker.BuildLabelArray(labels),
		},
		"socket": docker.GetSocket(),
		"cpu": common.MapStr{
			"per_cpu_usage":        mockedCPUCalculator.PerCpuUsage(&stats),
			"total_usage":          mockedCPUCalculator.TotalUsage(&stats),
			"usage_in_kernel_mode": mockedCPUCalculator.UsageInKernelmode(&stats),
			"usage_in_user_mode":   mockedCPUCalculator.UsageInUsermode(&stats),
		},
	}

	CPUService := NewCpuService()
	cpuData := CPUService.getCpuStats(&cpuStatsStruct)
	event := eventMapping(&cpuData)
	//THEN
	assert.True(t, equalEvent(expectedEvent, event))
}*/

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
