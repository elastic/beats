package mocks

import "github.com/ingensi/dockerbeat/calculator"
import "github.com/stretchr/testify/mock"

type CalculatorFactory struct {
	mock.Mock
}

func (_m *CalculatorFactory) NewBlkioCalculator(old calculator.BlkioData, new calculator.BlkioData) calculator.BlkioCalculator {
	ret := _m.Called(old, new)

	var r0 calculator.BlkioCalculator
	if rf, ok := ret.Get(0).(func(calculator.BlkioData, calculator.BlkioData) calculator.BlkioCalculator); ok {
		r0 = rf(old, new)
	} else {
		r0 = ret.Get(0).(calculator.BlkioCalculator)
	}

	return r0
}
func (_m *CalculatorFactory) NewCPUCalculator(old calculator.CPUData, new calculator.CPUData) calculator.CPUCalculator {
	ret := _m.Called(old, new)

	var r0 calculator.CPUCalculator
	if rf, ok := ret.Get(0).(func(calculator.CPUData, calculator.CPUData) calculator.CPUCalculator); ok {
		r0 = rf(old, new)
	} else {
		r0 = ret.Get(0).(calculator.CPUCalculator)
	}

	return r0
}
func (_m *CalculatorFactory) NewNetworkCalculator(old calculator.NetworkData, new calculator.NetworkData) calculator.NetworkCalculator {
	ret := _m.Called(old, new)

	var r0 calculator.NetworkCalculator
	if rf, ok := ret.Get(0).(func(calculator.NetworkData, calculator.NetworkData) calculator.NetworkCalculator); ok {
		r0 = rf(old, new)
	} else {
		r0 = ret.Get(0).(calculator.NetworkCalculator)
	}

	return r0
}
