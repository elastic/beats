package cpu

import (
	go_dockerclient "github.com/fsouza/go-dockerclient"
	"github.com/stretchr/testify/mock"

	common "github.com/elastic/beats/libbeat/common"
)

type MockCPUCalculator struct {
	mock.Mock
}

func (_m *MockCPUCalculator) PerCpuUsage(stats *go_dockerclient.Stats) common.MapStr {
	ret := _m.Called()
	ret0, _ := ret[0].(common.MapStr)
	return ret0
}

func (_m *MockCPUCalculator) TotalUsage(stats *go_dockerclient.Stats) float64 {
	ret := _m.Called()
	ret0, _ := ret[0].(float64)
	return ret0
}

func (_m *MockCPUCalculator) UsageInKernelmode(stats *go_dockerclient.Stats) float64 {
	ret := _m.Called()
	ret0, _ := ret[0].(float64)
	return ret0
}

func (_m *MockCPUCalculator) UsageInUsermode(stats *go_dockerclient.Stats) float64 {
	ret := _m.Called()
	ret0, _ := ret[0].(float64)
	return ret0
}
