package v2

import (
	"github.com/elastic/beats/v7/x-pack/filebeat/input/httpjson/v2/internal/transforms"
)

type responseConfig struct {
	Transforms transforms.Config `config:"transforms"`
}

func (c *responseConfig) Validate() error {
	if _, err := transforms.New(c.Transforms, responseNamespace); err != nil {
		return err
	}

	return nil
}
