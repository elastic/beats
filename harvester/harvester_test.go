package harvester

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

// Most harvester tests need real files to tes that can be modified. These tests are implemented with
// system tests

func TestExampleTest(t *testing.T) {

	h := Harvester{
		Path:   "/var/log/",
		Offset: 0,
	}

	assert.Equal(t, "/var/log/", h.Path)

}
