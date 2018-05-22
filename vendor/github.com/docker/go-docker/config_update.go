package docker // import "docker.io/go-docker"

import (
	"net/url"
	"strconv"

	"docker.io/go-docker/api/types/swarm"
	"golang.org/x/net/context"
)

// ConfigUpdate attempts to update a Config
func (cli *Client) ConfigUpdate(ctx context.Context, id string, version swarm.Version, config swarm.ConfigSpec) error {
	if err := cli.NewVersionError("1.30", "config update"); err != nil {
		return err
	}
	query := url.Values{}
	query.Set("version", strconv.FormatUint(version.Index, 10))
	resp, err := cli.post(ctx, "/configs/"+id+"/update", query, config, nil)
	ensureReaderClosed(resp)
	return err
}
