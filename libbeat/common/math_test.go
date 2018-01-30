package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRound(t *testing.T) {
	assert.EqualValues(t, 0.5, Round(0.5, DefaultDecimalPlacesCount))
	assert.EqualValues(t, 0.5, Round(0.50004, DefaultDecimalPlacesCount))
	assert.EqualValues(t, 0.5001, Round(0.50005, DefaultDecimalPlacesCount))

	assert.EqualValues(t, 1234.5, Round(1234.5, DefaultDecimalPlacesCount))
	assert.EqualValues(t, 1234.5, Round(1234.50004, DefaultDecimalPlacesCount))
	assert.EqualValues(t, 1234.5001, Round(1234.50005, DefaultDecimalPlacesCount))
}
