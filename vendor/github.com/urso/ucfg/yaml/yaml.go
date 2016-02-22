package yaml

import (
	"io/ioutil"

	"github.com/urso/ucfg"
	"gopkg.in/yaml.v2"
)

func NewConfig(in []byte) (*ucfg.Config, error) {
	var m map[string]interface{}
	if err := yaml.Unmarshal(in, &m); err != nil {
		return nil, err
	}
	return ucfg.NewFrom(m)
}

func NewConfigWithFile(name string) (*ucfg.Config, error) {
	input, err := ioutil.ReadFile(name)
	if err != nil {
		return nil, err
	}
	return NewConfig(input)
}
