package ilm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnabledFor(t *testing.T) {
	testdata := []struct {
		enabled bool
		client  ESClient
	}{
		{enabled: false},
		{enabled: false, client: &ES65{}},
		{enabled: false, client: &ES66Disabled{}},
		{enabled: true, client: &ES66{}},
	}
	for _, td := range testdata {
		assert.Equal(t, td.enabled, EnabledFor(td.client))
	}

}
