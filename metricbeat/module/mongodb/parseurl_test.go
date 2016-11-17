package mongodb

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseURL(t *testing.T) {
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
			Name:     "user password overwride",
			URL:      "mongodb://user:secret@localhost:40001",
			Username: "anotheruser",
			Password: "anotherpass",

			ExpectedAddr:     "localhost:40001",
			ExpectedUsername: "anotheruser",
			ExpectedPassword: "anotherpass",
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
	}

	for _, test := range tests {
		info, err := ParseURL(test.URL, test.Username, test.Password)
		assert.NoError(t, err, test.Name)
		assert.Equal(t, info.Addrs[0], test.ExpectedAddr, test.Name)
		assert.Equal(t, info.Username, test.ExpectedUsername, test.Name)
		assert.Equal(t, info.Password, test.ExpectedPassword, test.Name)
	}
}
