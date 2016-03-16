// +build !integration

package sys

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
