package uwsgi

import (
	"testing"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"

	"github.com/stretchr/testify/assert"
)

func TestHostParser(t *testing.T) {
	tests := []struct {
		host, expected string
	}{
		{"https://localhost", "https://localhost"},
		{"http://localhost:9191", "http://localhost:9191"},
		{"localhost", "tcp://localhost"},
		{"localhost:9191", "tcp://localhost:9191"},
		{"tcp://localhost:9191", "tcp://localhost:9191"},
		{"unix:///var/lib/uwsgi.sock", "unix:///var/lib/uwsgi.sock"},
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
