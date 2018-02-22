package util

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestValueMap(t *testing.T) {
	test := newValueMap(defaultTimeout)

	// no value
	assert.Equal(t, 0.0, test.Get("foo"))

	// Set and test
	test.Set("foo", 3.14)
	assert.Equal(t, 3.14, test.Get("foo"))
}

func TestGetWithDefault(t *testing.T) {
	test := newValueMap(defaultTimeout)

	// Empty + default
	assert.Equal(t, 0.0, test.Get("foo"))
	assert.Equal(t, 3.14, test.GetWithDefault("foo", 3.14))

	// Defined value
	test.Set("foo", 38.2)
	assert.Equal(t, 38.2, test.GetWithDefault("foo", 3.14))
}

func TestTimeout(t *testing.T) {
	test := newValueMap(10 * time.Millisecond)

	test.Set("foo", 3.14)
	assert.Equal(t, 3.14, test.Get("foo"))

	// expired:
	time.Sleep(60 * time.Millisecond)
	assert.Equal(t, 0.0, test.Get("foo"))
}

func TestContainerUID(t *testing.T) {
	assert.Equal(t, "a-b-c", ContainerUID("a", "b", "c"))
}
