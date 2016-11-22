package cpu

import (
	"github.com/elastic/beats/libbeat/common"

	dc "github.com/fsouza/go-dockerclient"
	"github.com/stretchr/testify/mock"
)

type MockCPUCalculator struct {
	mock.Mock
}

func (_m *MockCPUCalculator) PerCpuUsage(stats *dc.Stats) common.MapStr {
	ret := _m.Called()
	ret0, _ := ret[0].(common.MapStr)
	return ret0
}

func (_m *MockCPUCalculator) TotalUsage(stats *dc.Stats) float64 {
	ret := _m.Called()
	ret0, _ := ret[0].(float64)
	return ret0
}

func (_m *MockCPUCalculator) UsageInKernelmode(stats *dc.Stats) float64 {
	ret := _m.Called()
	ret0, _ := ret[0].(float64)
	return ret0
}

func (_m *MockCPUCalculator) UsageInUsermode(stats *dc.Stats) float64 {
	ret := _m.Called()
	ret0, _ := ret[0].(float64)
	return ret0
}
