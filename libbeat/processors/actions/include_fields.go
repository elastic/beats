package actions

import (
	"fmt"
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/processors"
)

type IncludeFields struct {
	Fields []string
	// condition
	Cond *processors.Condition
}

type IncludeFieldsConfig struct {
	Fields                     []string `config:"fields"`
	processors.ConditionConfig `config:",inline"`
}

func init() {
	if err := processors.RegisterPlugin("include_fields", newIncludeFields); err != nil {
		panic(err)
	}
}

func newIncludeFields(c common.Config) (processors.Processor, error) {

	f := IncludeFields{}

	if err := f.CheckConfig(c); err != nil {
		return nil, err
	}

	config := IncludeFieldsConfig{}

	err := c.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("fail to unpack the include_fields configuration: %s", err)
	}

	/* add read only fields if they are not yet */
	for _, readOnly := range processors.MandatoryExportedFields {
		found := false
		for _, field := range config.Fields {
			if readOnly == field {
				found = true
			}
		}
		if !found {
			config.Fields = append(config.Fields, readOnly)
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

func (f *IncludeFields) CheckConfig(c common.Config) error {

	complete := false

	for _, field := range c.GetFields() {
		if !processors.AvailableCondition(field) {
			if field != "fields" {
				return fmt.Errorf("unexpected %s option in the include_fields configuration", field)
			}
		}
		if field == "fields" {
			complete = true
		}
	}

	if !complete {
		return fmt.Errorf("missing fields option in the include_fields configuration")
	}
	return nil
}

func (f *IncludeFields) Run(event common.MapStr) (common.MapStr, error) {

	if f.Cond != nil && !f.Cond.Check(event) {
		return event, nil
	}

	filtered := common.MapStr{}

	for _, field := range f.Fields {
		hasKey, err := event.HasKey(field)
		if err != nil {
			return filtered, fmt.Errorf("Fail to check the key %s: %s", field, err)
		}

		if hasKey {
			errorOnCopy := event.CopyFieldsTo(filtered, field)
			if errorOnCopy != nil {
				return filtered, fmt.Errorf("Fail to copy key %s: %s", field, err)
			}
		}
	}

	return filtered, nil
}

func (f IncludeFields) String() string {

	if f.Cond != nil {
		return "include_fields=" + strings.Join(f.Fields, ", ") + ", condition=" + f.Cond.String()
	}
	return "include_fields=" + strings.Join(f.Fields, ", ")
}
