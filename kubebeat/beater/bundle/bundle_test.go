package bundle

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateServer(t *testing.T) {
	assert := assert.New(t)

	server, err := CreateServer()
	assert.NoError(err)

	var tests = []struct {
		path               string
		expectedStatusCode string
	}{
		{
			"/bundles/bundle.tar.gz", "200 OK",
		},
		{
			"/bundles/notExistBundle.tar.gz", "404 Not Found",
		},
		{
			"/bundles/notExistBundle", "404 Not Found",
		},
	}

	for _, test := range tests {
		target := "http://" + server.Addr + test.path
		client := &http.Client{}
		res, err := client.Get(target)

		assert.NoError(err)
		assert.Equal(test.expectedStatusCode, res.Status)
	}
}
