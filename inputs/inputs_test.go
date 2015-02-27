package inputs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInputNames(t *testing.T) {
	assert.Equal(t, "udpjson", UdpjsonInput.String())
	assert.Equal(t, "sniffer", SnifferInput.String())
	assert.Equal(t, "impossible", Input(2).String())
}

func TestIsInList(t *testing.T) {
	assert.True(t, UdpjsonInput.IsInList([]string{"sniffer", "udpjson"}))
}
