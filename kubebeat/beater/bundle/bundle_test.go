package bundle

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCreateServer(t *testing.T) {
	assert := assert.New(t)

	_, err := StartServer()
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

	time.Sleep(time.Second * 2)
	for _, test := range tests {
		target := ServerAddress + test.path
		client := &http.Client{}
		res, err := client.Get(target)

		assert.NoError(err)
		assert.Equal(test.expectedStatusCode, res.Status)
	}
}
