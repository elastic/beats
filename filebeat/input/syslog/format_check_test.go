package syslog

import (
	"gotest.tools/assert"
	"testing"
)

func TestIsRFC5424(t *testing.T) {
	assert.Equal(t, IsRFC5424Format([]byte(RfcDoc65Example1)), true)
	assert.Equal(t, IsRFC5424Format([]byte(RfcDoc65Example2)), true)
	assert.Equal(t, IsRFC5424Format([]byte(RfcDoc65Example3)), true)
	assert.Equal(t, IsRFC5424Format([]byte(RfcDoc65Example4)), true)
	assert.Equal(t, IsRFC5424Format([]byte("<190>2018-06-19T02:13:38.635322-0700 super mon message")), false)
	assert.Equal(t, IsRFC5424Format([]byte("<190>589265: Feb 8 18:55:31.306: %SEC-11-IPACCESSLOGP: list 177 denied udp 10.0.0.1(53640) -> 10.100.0.1(15600), 1 packet")), false)
}
