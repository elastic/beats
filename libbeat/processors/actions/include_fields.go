package actions

import (
	"fmt"
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/processors"
)

type includeFields struct {
	Fields []string
}

func init() {
	processors.RegisterPlugin("include_fields",
		configChecked(newIncludeFields,
			requireFields("fields"),
			allowedFields("fields", "when")))
}

func newIncludeFields(c common.Config) (processors.Processor, error) {
	config := struct {
		Fields []string `config:"fields"`
	}{}
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

	f := includeFields{Fields: config.Fields}
	return &f, nil
}

func (f includeFields) Run(event common.MapStr) (common.MapStr, error) {
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

func (f includeFields) String() string {
	return "include_fields=" + strings.Join(f.Fields, ", ")
}
