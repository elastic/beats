package json

import (
	"encoding/json"
	"io/ioutil"

	"github.com/elastic/go-ucfg"
)

func NewConfig(in []byte, opts ...ucfg.Option) (*ucfg.Config, error) {
	var m interface{}
	if err := json.Unmarshal(in, &m); err != nil {
		return nil, err
	}
	return ucfg.NewFrom(m, opts...)
}

func NewConfigWithFile(name string, opts ...ucfg.Option) (*ucfg.Config, error) {
	input, err := ioutil.ReadFile(name)
	if err != nil {
		return nil, err
	}

	opts = append([]ucfg.Option{ucfg.MetaData(ucfg.Meta{name})}, opts...)
	return NewConfig(input, opts...)
}
