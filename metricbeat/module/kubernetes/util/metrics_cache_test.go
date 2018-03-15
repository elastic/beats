package util

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTimeout(t *testing.T) {
	// Mocknotonic time:
	fakeTime := time.Now().Unix()
	now = func() time.Time {
		fakeTime++
		return time.Unix(fakeTime, 0)
	}

	// Blocking sleep:
	sleepCh := make(chan struct{})
	sleep = func(time.Duration) {
		<-sleepCh
	}

	test := newValueMap(1 * time.Second)

	test.Set("foo", 3.14)

	// Let cleanup do its job
	sleepCh <- struct{}{}
	sleepCh <- struct{}{}
	sleepCh <- struct{}{}

	// Check it expired
	assert.Equal(t, 0.0, test.Get("foo"))
}

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

func TestContainerUID(t *testing.T) {
	assert.Equal(t, "a-b-c", ContainerUID("a", "b", "c"))
}
