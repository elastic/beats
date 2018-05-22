package docker // import "docker.io/go-docker"

import (
	"golang.org/x/net/context"
)

// PluginSet modifies settings for an existing plugin
func (cli *Client) PluginSet(ctx context.Context, name string, args []string) error {
	resp, err := cli.post(ctx, "/plugins/"+name+"/set", nil, args, nil)
	ensureReaderClosed(resp)
	return err
}
