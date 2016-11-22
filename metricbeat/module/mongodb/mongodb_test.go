package mongodb

import (
	"testing"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"

	"github.com/stretchr/testify/assert"
)

func TestParseMongoURL(t *testing.T) {
	tests := []struct {
		Name             string
		URL              string
		Username         string
		Password         string
		ExpectedAddr     string
		ExpectedUsername string
		ExpectedPassword string
	}{
		{
			Name:     "basic test",
			URL:      "localhost:40001",
			Username: "user",
			Password: "secret",

			ExpectedAddr:     "localhost:40001",
			ExpectedUsername: "user",
			ExpectedPassword: "secret",
		},
		{
			Name:     "with schema",
			URL:      "mongodb://localhost:40001",
			Username: "user",
			Password: "secret",

			ExpectedAddr:     "localhost:40001",
			ExpectedUsername: "user",
			ExpectedPassword: "secret",
		},
		{
			Name:     "user password in url",
			URL:      "mongodb://user:secret@localhost:40001",
			Username: "",
			Password: "",

			ExpectedAddr:     "localhost:40001",
			ExpectedUsername: "user",
			ExpectedPassword: "secret",
		},
		{
			Name:     "username and password do not overwride",
			URL:      "mongodb://user:secret@localhost:40001",
			Username: "anotheruser",
			Password: "anotherpass",

			ExpectedAddr:     "localhost:40001",
			ExpectedUsername: "user",
			ExpectedPassword: "secret",
		},
		{
			Name:     "with options",
			URL:      "mongodb://localhost:40001?connect=direct&authSource=me",
			Username: "anotheruser",
			Password: "anotherpass",

			ExpectedAddr:     "localhost:40001",
			ExpectedUsername: "anotheruser",
			ExpectedPassword: "anotherpass",
		},
		{
			Name:     "multiple hosts",
			URL:      "mongodb://localhost:40001,localhost:40002",
			Username: "",
			Password: "",

			ExpectedAddr:     "localhost:40001,localhost:40002",
			ExpectedUsername: "",
			ExpectedPassword: "",
		},
	}

	for _, test := range tests {
		mod := mbtest.NewTestModule(t, map[string]interface{}{
			"username": test.Username,
			"password": test.Password,
		})
		hostData, err := ParseURL(mod, test.URL)
		if err != nil {
			t.Error(err)
			continue
		}

		assert.Equal(t, test.ExpectedAddr, hostData.Host, test.Name)
		assert.Equal(t, test.ExpectedUsername, hostData.User, test.Name)
		assert.Equal(t, test.ExpectedPassword, hostData.Password, test.Name)
	}
}
