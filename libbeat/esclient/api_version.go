package client

import "github.com/elastic/beats/libbeat/common"

func (c *Client) GetVersion() *common.Version {
	return c.version
}
