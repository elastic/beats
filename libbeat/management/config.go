package management

import (
	"os"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/file"
	"github.com/elastic/beats/libbeat/paths"
	"github.com/elastic/beats/libbeat/setup/kibana"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// Config for central management
type Config struct {
	// true when enrolled
	Enabled bool

	// TODO use beat.Keystore() for access_token
	AccessToken string

	Kibana *kibana.Config

	Configs []struct {
		Name   string
		Config *common.Config
	}
}

// Load settings from its source file
func (c *Config) Load() error {
	path := paths.Resolve(paths.Data, "management.yaml")
	config, err := common.LoadFile(path)
	if err != nil {
		return err
	}

	if err = config.Unpack(c); err != nil {
		return err
	}

	return nil
}

// Save settings to management.yaml file
func (c *Config) Save() error {
	path := paths.Resolve(paths.Data, "management.yaml")

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	// write temporary file first
	// TODO this should be owned by the beats user, ensure that
	tempFile := path + ".new"
	f, err := os.OpenFile(tempFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return errors.Wrap(err, "failed to store central management settings")
	}

	_, err = f.Write(data)
	f.Close()
	if err != nil {
		return err
	}

	// move temporary file into final location
	err = file.SafeFileRotate(path, tempFile)
	return err
}
