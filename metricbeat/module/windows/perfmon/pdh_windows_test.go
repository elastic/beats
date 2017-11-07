package perfmon

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestPdhErrno checks that PdhError provides the correct message for known
// PDH errors and also falls back to Windows error messages for non-PDH errors.
func TestPdhErrno_Error(t *testing.T) {
	assert.Contains(t, PdhErrno(PDH_CSTATUS_BAD_COUNTERNAME).Error(), "Unable to parse the counter path.")
	assert.Contains(t, PdhErrno(15).Error(), "The system cannot find the drive specified.")
}
