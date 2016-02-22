package json

import (
	"encoding/json"
	"io/ioutil"

	"github.com/urso/ucfg"
)

func NewConfig(in []byte) (*ucfg.Config, error) {
	var m map[string]interface{}
	if err := json.Unmarshal(in, &m); err != nil {
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
