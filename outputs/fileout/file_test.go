package fileout

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetNameByIP(t *testing.T) {
	out := new(FileOutput)
	assert.Empty(t, out.GetNameByIP("192.168.1.1"))
}

func TestPublishIPs(t *testing.T) {
	out := new(FileOutput)
	localAddrs := []string{"192", "168", "1", "1"}
	assert.Nil(t, out.PublishIPs("test", localAddrs))
}
