package valschema

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var IsDuration = func(t *testing.T, _ bool, actual interface{}) {
	converted, ok := actual.(time.Duration)
	assert.True(t, ok)
	assert.True(t, converted >= 0)
}

var IsNil = func(t *testing.T, _ bool, actual interface{}) {
	assert.Nil(t, actual)
}

var IsString = func(t *testing.T, _ bool, actual interface{}) {
	_, ok := actual.(string)
	assert.True(t, ok)
}

var DoesExist = func(t *testing.T, exists bool, actual interface{}) {
	assert.True(t, exists)
}

var DoesNotExist = func(t *testing.T, exists bool, actual interface{}) {
	assert.False(t, exists)
}
