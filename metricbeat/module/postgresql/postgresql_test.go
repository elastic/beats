package postgresql

import (
	"testing"
	"time"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"

	"github.com/stretchr/testify/assert"
)

func TestParseUrl(t *testing.T) {
	tests := []struct {
		Name     string
		URL      string
		Username string
		Password string
		Timeout  time.Duration
		Expected string
	}{
		{
			Name:     "simple test",
			URL:      "postgres://host1:5432",
			Expected: "host=host1 port=5432",
		},
		{
			Name:     "no port",
			URL:      "postgres://host1",
			Expected: "host=host1",
		},
		{
			Name:     "user/pass in URL",
			URL:      "postgres://user:pass@host1:5432",
			Expected: "host=host1 password=pass port=5432 user=user",
		},
		{
			Name:     "user/pass in params",
			URL:      "postgres://host1:5432",
			Username: "user",
			Password: "secret",
			Expected: "host=host1 password=secret port=5432 user=user",
		},
		{
			Name:     "user/pass in URL take precedence",
			URL:      "postgres://user1:pass@host1:5432",
			Username: "user",
			Password: "secret",
			Expected: "host=host1 password=pass port=5432 user=user1",
		},
		{
			Name:     "timeout no override",
			URL:      "postgres://host1:5432?connect_timeout=2",
			Expected: "connect_timeout=2 host=host1 port=5432",
		},
		{
			Name:     "timeout from param",
			URL:      "postgres://host1:5432",
			Timeout:  3 * time.Second,
			Expected: "connect_timeout=3 host=host1 port=5432",
		},
		{
			Name:     "user/pass in URL take precedence, and timeout override",
			URL:      "postgres://user1:pass@host1:5432?connect_timeout=2",
			Username: "user",
			Password: "secret",
			Timeout:  3 * time.Second,
			Expected: "connect_timeout=3 host=host1 password=pass port=5432 user=user1",
		},
		{
			Name:     "unix socket",
			URL:      "postgresql:///dbname?host=/var/lib/postgresql",
			Expected: "dbname=dbname host=/var/lib/postgresql",
		},
	}

	for _, test := range tests {
		mod := mbtest.NewTestModule(t, map[string]interface{}{
			"username": test.Username,
			"password": test.Password,
		})
		mod.ModConfig.Timeout = test.Timeout
		hostData, err := ParseURL(mod, test.URL)
		if err != nil {
			t.Error(err)
			continue
		}

		assert.Equal(t, test.Expected, hostData.URI, test.Name)
	}
}
