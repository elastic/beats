package cpu

import (
	"reflect"
	"testing"

	"github.com/elastic/beats/libbeat/common"

	dc "github.com/fsouza/go-dockerclient"
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
			"0": common.MapStr{"pct": float64(0.10)},
			"1": common.MapStr{"pct": float64(0.90)},
			"2": common.MapStr{"pct": float64(0.90)},
			"3": common.MapStr{"pct": float64(0.50)},
		}},
		{statsList[1], common.MapStr{
			"0": common.MapStr{"pct": float64(0.0000001)},
			"1": common.MapStr{"pct": float64(0.0000002)},
			"2": common.MapStr{"pct": float64(0.0000003)},
			"3": common.MapStr{"pct": float64(0.0000004)},
		}},
		{statsList[2], common.MapStr{
			"0": common.MapStr{"pct": float64(0)},
			"1": common.MapStr{"pct": float64(0)},
			"2": common.MapStr{"pct": float64(0)},
			"3": common.MapStr{"pct": float64(0)},
		}},
	}
	for _, tt := range testCase {
		out := perCpuUsage(&tt.given)
		// Remove ticks for test
		for _, s := range out {
			s.(common.MapStr).Delete("ticks")
		}
		if !equalEvent(tt.expected, out) {
			t.Errorf("PerCpuUsage(%v) => %v, want %v", tt.given.CPUStats.CPUUsage.PercpuUsage, out, tt.expected)
		}
	}
}

func TestCPUService_TotalUsage(t *testing.T) {
	oldTotalValuesTest := []uint64{0, 50, 10}
	totalValuesTest := []uint64{0, 500000050, 10}
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
		out := totalUsage(&tt.given)
		if tt.expected != out {
			t.Errorf("usageInKernelmode(%v) => %v, want %v", tt.given.CPUStats.CPUUsage.PercpuUsage, out, tt.expected)
		}
	}
}

func TestCPUService_UsageInKernelmode(t *testing.T) {
	usageOldValuesTest := []uint64{0, 10, 500000050}
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
		out := usageInKernelmode(&tt.given)
		if out != tt.expected {
			t.Errorf("usageInKernelmode(%v) => %v, want %v", tt.given, out, tt.expected)
		}
	}
}

func TestCPUService_UsageInUsermode(t *testing.T) {
	usageOldValuesTest := []uint64{0, 1958965, 500}
	usageValuesTest := []uint64{500000000, 1958965, 1000000500}
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
		out := usageInUsermode(&tt.given)
		if out != tt.expected {
			t.Errorf("usageInKernelmode(%v) => %v, want %v", tt.given, out, tt.expected)
		}
	}
}

func equalEvent(expectedEvent common.MapStr, event common.MapStr) bool {
	return reflect.DeepEqual(expectedEvent, event)
}
