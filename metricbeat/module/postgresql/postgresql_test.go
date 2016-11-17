package postgresql

import (
	"testing"
	"time"

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
			Name:     "user/pass override",
			URL:      "postgres://user1:pass@host1:5432",
			Username: "user",
			Password: "secret",
			Expected: "host=host1 password=secret port=5432 user=user",
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
			Name:     "user/pass override, and timeout override",
			URL:      "postgres://user1:pass@host1:5432?connect_timeout=2",
			Username: "user",
			Password: "secret",
			Timeout:  3 * time.Second,
			Expected: "connect_timeout=3 host=host1 password=secret port=5432 user=user",
		},
	}

	for _, test := range tests {
		url, err := ParseURL(test.URL, test.Username, test.Password, test.Timeout)
		assert.NoError(t, err, test.Name)
		assert.Equal(t, test.Expected, url, test.Name)
	}
}
