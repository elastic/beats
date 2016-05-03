// +build !integration

package mb

import (
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"

	"github.com/stretchr/testify/assert"
)

func TestModuleConfig(t *testing.T) {
	tests := []struct {
		in  map[string]interface{}
		out ModuleConfig
		err string
	}{
		{
			in:  map[string]interface{}{},
			out: defaultModuleConfig,
		},
	}

	for _, test := range tests {
		c, err := common.NewConfigFrom(test.in)
		if err != nil {
			t.Fatal(err)
		}

		mc := defaultModuleConfig
		err = c.Unpack(&mc)
		if test.err != "" {
			assert.Error(t, err)
			assert.Contains(t, err.Error(), test.err)
			continue
		}

		assert.Equal(t, test.out, mc)
	}
}

// TestModuleConfigDefaults validates that the default values are not changed.
// Any changes to this test case are probably indicators of non-backwards
// compatible changes affect all modules (including community modules).
func TestModuleConfigDefaults(t *testing.T) {
	c, err := common.NewConfigFrom(map[string]interface{}{
		"module":     "mymodule",
		"metricsets": []string{"mymetricset"},
	})
	if err != nil {
		t.Fatal(err)
	}

	mc := defaultModuleConfig
	err = c.Unpack(&mc)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, true, mc.Enabled)
	assert.Equal(t, time.Second, mc.Period)
	assert.Equal(t, time.Second, mc.Timeout)
	assert.Empty(t, mc.Hosts)
}
