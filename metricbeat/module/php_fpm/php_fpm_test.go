package php_fpm

import (
	"testing"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"

	"github.com/stretchr/testify/assert"
)

func TestHostParser(t *testing.T) {
	tests := []struct {
		host, expected string
	}{
		{"localhost", "http://localhost/status?json="},
		{"localhost:123", "http://localhost:123/status?json="},
		{"http://localhost:123", "http://localhost:123/status?json="},
	}

	m := mbtest.NewTestModule(t, map[string]interface{}{})

	for _, test := range tests {
		hi, err := HostParser(m, test.host)
		if err != nil {
			t.Error("failed on", test.host, err)
			continue
		}
		assert.Equal(t, test.expected, hi.URI)
	}
}
