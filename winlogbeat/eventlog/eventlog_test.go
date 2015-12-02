package eventlog

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSidTypeValues verifies that the values of the constants are not
// accidentally changed by a developer. The values should never be changed
// because they correspond to enum values defined on Windows.
// https://msdn.microsoft.com/en-us/library/windows/desktop/aa379601(v=vs.85).aspx
func TestSidTypeValues(t *testing.T) {
	values := []SIDType{
		SidTypeUser,
		SidTypeGroup,
		SidTypeDomain,
		SidTypeAlias,
		SidTypeWellKnownGroup,
		SidTypeDeletedAccount,
		SidTypeInvalid,
		SidTypeUnknown,
		SidTypeComputer,
		SidTypeLabel,
	}
	for i, sidType := range values {
		assert.Equal(t, SIDType(i+1), sidType, "SID type: "+sidType.String())
	}
}

// TestEventLogValues verifies that the EventType constants match up with the
// Microsoft declared values.
// https://msdn.microsoft.com/en-us/library/windows/desktop/aa363679(v=vs.85).aspx
func TestEventLogValues(t *testing.T) {
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
