package pandora

import "errors"

type pandoraConfig struct {
	Endpoint   string `config:"endpoint"`
	AK         string `config:"ak"`
	SK         string `config:"sk"`
	Region     string `config:"region"`
	Batch      int    `config:"batch" validate:"min=1"`
	MaxRetries int    `config:"max_retries"`
}

var (
	defaultConfig = pandoraConfig{
		//Timeout:          90 * time.Second,
		MaxRetries: 3,
		Batch:      10,
	}
)

func (c *pandoraConfig) Validate() error {
	if c.Endpoint != "" {
		if _, err := parseProxyURL(c.Endpoint); err != nil {
			return err
		}
	}
	if c.Region == "" {
		return errors.New("should set region for repo info")
	}
	return nil
}
