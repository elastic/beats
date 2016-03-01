// build +windows,!integration

package eventlogging

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEventLogReadFlags(t *testing.T) {
	assert.Equal(t, EventLogReadFlag(0x0001), EVENTLOG_SEQUENTIAL_READ)
	assert.Equal(t, EventLogReadFlag(0x0002), EVENTLOG_SEEK_READ)
	assert.Equal(t, EventLogReadFlag(0x0004), EVENTLOG_FORWARDS_READ)
	assert.Equal(t, EventLogReadFlag(0x0008), EVENTLOG_BACKWARDS_READ)
}

func TestLoadLibraryExFlags(t *testing.T) {
	assert.Equal(t, uint32(0x00000001), DONT_RESOLVE_DLL_REFERENCES)
	assert.Equal(t, uint32(0x00000010), LOAD_IGNORE_CODE_AUTHZ_LEVEL)
	assert.Equal(t, uint32(0x00000002), LOAD_LIBRARY_AS_DATAFILE)
	assert.Equal(t, uint32(0x00000040), LOAD_LIBRARY_AS_DATAFILE_EXCLUSIVE)
	assert.Equal(t, uint32(0x00000020), LOAD_LIBRARY_AS_IMAGE_RESOURCE)
	assert.Equal(t, uint32(0x00000200), LOAD_LIBRARY_SEARCH_APPLICATION_DIR)
	assert.Equal(t, uint32(0x00001000), LOAD_LIBRARY_SEARCH_DEFAULT_DIRS)
	assert.Equal(t, uint32(0x00000100), LOAD_LIBRARY_SEARCH_DLL_LOAD_DIR)
	assert.Equal(t, uint32(0x00000800), LOAD_LIBRARY_SEARCH_SYSTEM32)
	assert.Equal(t, uint32(0x00000400), LOAD_LIBRARY_SEARCH_USER_DIRS)
	assert.Equal(t, uint32(0x00000008), LOAD_WITH_ALTERED_SEARCH_PATH)
}

// TestEventTypeValues verifies that the EventType constants match up with the
// Microsoft declared values.
// https://msdn.microsoft.com/en-us/library/windows/desktop/aa363679(v=vs.85).aspx
func TestEventTypeValues(t *testing.T) {
	testCases := []struct {
		observed EventType
		expected uint8
	}{
		{EVENTLOG_SUCCESS, 0},
		{EVENTLOG_ERROR_TYPE, 0x1},
		{EVENTLOG_WARNING_TYPE, 0x2},
		{EVENTLOG_INFORMATION_TYPE, 0x4},
		{EVENTLOG_AUDIT_SUCCESS, 0x8},
		{EVENTLOG_AUDIT_FAILURE, 0x10},
	}
	for _, test := range testCases {
		assert.Equal(t, EventType(test.expected), test.observed,
			"Event type: "+test.observed.String())
	}
}
