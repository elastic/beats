package withassert

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSimpleAssertFails(t *testing.T) {
	assert.True(t, false)
}

func TestSimpleAssertWithMessage(t *testing.T) {
	assert.True(t, false, "My message")
}

func TestSimpleAssertWithMessagef(t *testing.T) {
	assert.True(t, false, "My message with arguments: %v", 42)
}

func TestSimpleRequireFails(t *testing.T) {
	require.True(t, false)
}

func TestSimpleRequireWithMessage(t *testing.T) {
	require.True(t, false, "My message")
}

func TestSimpleRequireWithMessagef(t *testing.T) {
	require.True(t, false, "My message with arguments: %v", 42)
}

func TestFailEqualMaps(t *testing.T) {
	want := map[string]interface{}{
		"a": 1,
		"b": true,
		"c": "test",
		"e": map[string]interface{}{
			"x": "y",
		},
	}

	got := map[string]interface{}{
		"a": 42,
		"b": false,
		"c": "test",
	}

	assert.Equal(t, want, got)
}
