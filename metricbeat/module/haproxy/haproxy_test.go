package haproxy

import (
	"testing"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"

	"github.com/stretchr/testify/assert"
)

func TestHostParser(t *testing.T) {
	tests := []struct {
		host, expected string
	}{
		{"localhost", "tcp://localhost"},
		{"localhost:123", "tcp://localhost:123"},
		{"tcp://localhost:123", "tcp://localhost:123"},
		{"unix:///var/lib/haproxy/stats", "unix:///var/lib/haproxy/stats"},
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
