package actions

import (
	"fmt"
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/processors"
	"github.com/pkg/errors"
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
	errs := []string{}

	for _, field := range f.Fields {
		err := event.CopyFieldsTo(filtered, field)
		// Ignore errors caused by a field not existing in the event.
		if err != nil && errors.Cause(err) != common.ErrKeyNotFound {
			errs = append(errs, err.Error())
		}
	}

	return filtered, fmt.Errorf(strings.Join(errs, ", "))
}

func (f includeFields) String() string {
	return "include_fields=" + strings.Join(f.Fields, ", ")
}
