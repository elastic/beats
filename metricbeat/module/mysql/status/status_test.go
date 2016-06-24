package status

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"

	"github.com/stretchr/testify/assert"
)

// TestConfigValidation validates that the configuration and the DSN are
// validated when the MetricSet is created.
func TestConfigValidation(t *testing.T) {
	tests := []struct {
		in  interface{}
		err string
	}{
		{
			// Missing 'hosts'
			in: map[string]interface{}{
				"module":     "mysql",
				"metricsets": []string{"status"},
			},
			err: "missing required field accessing 'hosts'",
		},
		{
			// Invalid DSN
			in: map[string]interface{}{
				"module":     "mysql",
				"metricsets": []string{"status"},
				"hosts":      []string{"127.0.0.1"},
			},
			err: "config error for host '127.0.0.1': invalid DSN: missing the slash separating the database name",
		},
		{
			// Local unix socket
			in: map[string]interface{}{
				"module":     "mysql",
				"metricsets": []string{"status"},
				"hosts":      []string{"user@unix(/path/to/socket)/"},
			},
		},
		{
			// TCP on a remote host, e.g. Amazon RDS:
			in: map[string]interface{}{
				"module":     "mysql",
				"metricsets": []string{"status"},
				"hosts":      []string{"id:password@tcp(your-amazonaws-uri.com:3306)/}"},
			},
		},
		{
			// TCP on a remote host with user/pass specified separately
			in: map[string]interface{}{
				"module":     "mysql",
				"metricsets": []string{"status"},
				"hosts":      []string{"tcp(your-amazonaws-uri.com:3306)/}"},
				"username":   "id",
				"password":   "mypass",
			},
		},
	}

	for i, test := range tests {
		c, err := common.NewConfigFrom(test.in)
		if err != nil {
			t.Fatal(err)
		}

		_, err = mb.NewModules([]*common.Config{c}, mb.Registry)
		if err != nil && test.err == "" {
			t.Errorf("unexpected error in testcase %d: %v", i, err)
			continue
		}
		if test.err != "" && assert.Error(t, err, "expected '%v' in testcase %d", test.err, i) {
			assert.Contains(t, err.Error(), test.err, "testcase %d", i)
			continue
		}
	}
}
