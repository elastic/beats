package actions

import (
	"fmt"
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/processors"
)

type dropFields struct {
	Fields []string
}

func init() {
	processors.RegisterPlugin("drop_fields",
		configChecked(newDropFields,
			requireFields("fields"),
			allowedFields("fields", "when")))
}

func newDropFields(c common.Config) (processors.Processor, error) {
	config := struct {
		Fields []string `config:"fields"`
	}{}
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

	f := dropFields{Fields: config.Fields}
	return f, nil
}

func (f dropFields) Run(event common.MapStr) (common.MapStr, error) {
	errors := []string{}

	for _, field := range f.Fields {
		err := event.Delete(field)
		if err != nil {
			errors = append(errors, err.Error())
		}

	}
	return event, fmt.Errorf(strings.Join(errors, ", "))
}

func (f dropFields) String() string {
	return "drop_fields=" + strings.Join(f.Fields, ", ")
}
