package actions

import (
	"fmt"
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/processors"
)

type DropFields struct {
	Fields []string
	// condition
	Cond *processors.Condition
}

type DropFieldsConfig struct {
	Fields                     []string `config:"fields"`
	processors.ConditionConfig `config:",inline"`
}

func init() {
	if err := processors.RegisterPlugin("drop_fields", newDropFields); err != nil {
		panic(err)
	}
}

func newDropFields(c common.Config) (processors.Processor, error) {

	f := DropFields{}

	if err := f.CheckConfig(c); err != nil {
		return nil, err
	}

	config := DropFieldsConfig{}

	err := c.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("fail to unpack the drop_fields configuration: %s", err)
	}

	/* remove read only fields */
	for _, readOnly := range processors.MandatoryExportedFields {
		for i, field := range config.Fields {
			if readOnly == field {
				config.Fields = append(config.Fields[:i], config.Fields[i+1:]...)
			}
		}
	}
	f.Fields = config.Fields

	cond, err := processors.NewCondition(config.ConditionConfig)
	if err != nil {
		return nil, err
	}
	f.Cond = cond

	return &f, nil
}

func (f *DropFields) CheckConfig(c common.Config) error {

	complete := false

	for _, field := range c.GetFields() {
		if !processors.AvailableCondition(field) {
			if field != "fields" {
				return fmt.Errorf("unexpected %s option in the drop_fields configuration", field)
			}
		}
		if field == "fields" {
			complete = true
		}
	}

	if !complete {
		return fmt.Errorf("missing fields option in the drop_fields configuration")
	}
	return nil
}

func (f *DropFields) Run(event common.MapStr) (common.MapStr, error) {

	if f.Cond != nil && !f.Cond.Check(event) {
		return event, nil
	}

	for _, field := range f.Fields {
		err := event.Delete(field)
		if err != nil {
			return event, fmt.Errorf("Fail to delete key %s: %s", field, err)
		}

	}
	return event, nil
}

func (f DropFields) String() string {

	if f.Cond != nil {
		return "drop_fields=" + strings.Join(f.Fields, ", ") + ", condition=" + f.Cond.String()
	}
	return "drop_fields=" + strings.Join(f.Fields, ", ")

}
