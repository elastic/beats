package skima

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var IsDuration = func(t *testing.T, _ bool, actual interface{}, dotPath string) {
	converted, ok := actual.(time.Duration)
	assert.True(t, ok, dotPath)
	assert.True(t, converted >= 0, dotPath)
}

var IsNil = func(t *testing.T, _ bool, actual interface{}, dotPath string) {
	assert.Nil(t, actual, dotPath)
}

var IsString = func(t *testing.T, _ bool, actual interface{}, dotPath string) {
	_, ok := actual.(string)
	assert.True(t, ok, dotPath)
}

var DoesExist = func(t *testing.T, exists bool, actual interface{}, dotPath string) {
	assert.True(t, exists, dotPath)
}

var DoesNotExist = func(t *testing.T, exists bool, actual interface{}, dotPath string) {
	assert.False(t, exists, dotPath)
}
