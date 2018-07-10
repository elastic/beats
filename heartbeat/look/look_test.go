package look

import (
	"testing"
	"time"

	"fmt"

	"github.com/stretchr/testify/assert"

	reason2 "github.com/elastic/beats/heartbeat/reason"
	"github.com/elastic/beats/libbeat/common"
)

// helper
func testRTT(t *testing.T, expected time.Duration, provided time.Duration) {
	actual, err := RTT(provided).GetValue("us")
	assert.Nil(t, err)
	assert.Equal(t, expected, actual)
}

func TestPositiveRTTIsKept(t *testing.T) {
	testRTT(t, 5, time.Duration(5*time.Microsecond))
}

func TestNegativeRTTIsZero(t *testing.T) {
	testRTT(t, time.Duration(0), time.Duration(-1))
}

func TestReason(t *testing.T) {
	reason := reason2.ValidateFailed(fmt.Errorf("an error"))
	res := Reason(reason)
	assert.Equal(t,
		common.MapStr{
			"type":    reason.Type(),
			"message": reason.Error(),
		}, res)
}

func TestReasonGenericError(t *testing.T) {
	msg := "An error"
	res := Reason(fmt.Errorf(msg))
	assert.Equal(t, common.MapStr{
		"type":    "io",
		"message": msg,
	}, res)
}

func TestTimestamp(t *testing.T) {
	now := time.Now()
	assert.Equal(t, common.Time(now), Timestamp(now))
}

func TestStatusNil(t *testing.T) {
	assert.Equal(t, "up", Status(nil))
}

func TestStatusErr(t *testing.T) {
	assert.Equal(t, "down", Status(fmt.Errorf("something")))
}
