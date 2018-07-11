package docker

import (
	"net/http"
	"os"

	"github.com/elastic/beats/libbeat/logp"

	"github.com/docker/docker/api"
	"github.com/docker/docker/api/types/versions"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
)

// NewClient builds and returns a new Docker client
// It uses version 1.30 by default, and negotiates it with the server so it is downgraded if 1.30 is too high
func NewClient(host string, httpClient *http.Client, httpHeaders map[string]string) (*client.Client, error) {
	version := os.Getenv("DOCKER_API_VERSION")
	if version == "" {
		version = api.DefaultVersion
	}

	c, err := client.NewClient(host, version, httpClient, nil)
	if err != nil {
		return c, err
	}

	if os.Getenv("DOCKER_API_VERSION") == "" {
		logp.Debug("docker", "Negotiating client version")
		ping, err := c.Ping(context.Background())
		if err != nil {
			logp.Debug("docker", "Failed to perform ping: %s", err)
		}

		// try a really old version, before versioning headers existed
		if ping.APIVersion == "" {
			ping.APIVersion = "1.22"
		}

		// if server version is lower than the client version, downgrade
		if versions.LessThan(ping.APIVersion, version) {
			c.UpdateClientVersion(ping.APIVersion)
		}
	}

	logp.Debug("docker", "Client version set to %s", c.ClientVersion())

	return c, nil
}
