package calculator

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewBlkioCalculator(t *testing.T) {
	// GIVEN
	// a factory
	factory := CalculatorFactoryImpl{}
	new := BlkioData{}
	old := BlkioData{}

	// WHEN
	calculator := factory.NewBlkioCalculator(old, new)

	// THEN
	// calculator is not null and data stored are correct
	assert.Equal(t, new, calculator.(BlkioCalculatorImpl).New)
	assert.Equal(t, old, calculator.(BlkioCalculatorImpl).Old)
}

func TestNewCPUCalculator(t *testing.T) {
	// GIVEN
	// a factory
	factory := CalculatorFactoryImpl{}
	new := CPUData{}
	old := CPUData{}

	// WHEN
	calculator := factory.NewCPUCalculator(old, new)

	// THEN
	// calculator is not null and data stored are correct
	assert.Equal(t, new, calculator.(CPUCalculatorImpl).New)
	assert.Equal(t, old, calculator.(CPUCalculatorImpl).Old)
}

func TestNewNetworkCalculator(t *testing.T) {
	// GIVEN
	// a factory
	factory := CalculatorFactoryImpl{}
	new := NetworkData{}
	old := NetworkData{}

	// WHEN
	calculator := factory.NewNetworkCalculator(old, new)

	// THEN
	// calculator is not null and data stored are correct
	assert.Equal(t, new, calculator.(NetworkCalculatorImpl).new)
	assert.Equal(t, old, calculator.(NetworkCalculatorImpl).old)
}
