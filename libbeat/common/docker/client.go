package docker

import (
	"net/http"

	"github.com/elastic/beats/libbeat/logp"

	client "docker.io/go-docker"
	"docker.io/go-docker/api"
	"golang.org/x/net/context"
)

// Select Docker API version
const dockerAPIVersion = api.DefaultVersion

// NewClient builds and returns a new Docker client
// It uses version 1.26 by default, and negotiates it with the server so it is downgraded if 1.26 is too high
func NewClient(host string, httpClient *http.Client, httpHeaders map[string]string) (*client.Client, error) {
	c, err := client.NewClient(host, dockerAPIVersion, httpClient, nil)
	if err != nil {
		return c, err
	}

	logp.Debug("docker", "Negotiating client version")
	c.NegotiateAPIVersion(context.Background())
	logp.Debug("docker", "Client version set to %s", c.ClientVersion())

	return c, err
}
