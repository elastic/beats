package pipeline

import (
	"errors"
	"fmt"

	"github.com/elastic/beats/v7/libbeat/common"
)

type Settings struct {
	Inputs  []InputSettings           `config:"club.inputs"`
	Outputs map[string]*common.Config `config:"outputs"`
}

type InputSettings struct {
	ID              string                 `config:"id"`
	Name            string                 `config:"name"`
	Type            string                 `config:"type"`
	Meta            map[string]interface{} `config:"name"`
	Namespace       string                 `config:"data_stream.namespace"`
	UseOutput       string                 `config:"use_output"`
	DefaultSettings *common.Config         `config:"default"`
	Streams         []*common.Config       `config:"streams"`
}

func (s *Settings) Validate() error {
	fmt.Printf("new configuration: %#v\n", s)

	if _, exists := s.Outputs["default"]; !exists {
		return errors.New("no default output configured")
	}

	for _, inp := range s.Inputs {
		if inp.UseOutput == "" {
			continue
		}
		if _, exist := s.Outputs[inp.UseOutput]; !exist {
			return fmt.Errorf("output '%v' not defined", inp.UseOutput)
		}
	}

	return nil
}
