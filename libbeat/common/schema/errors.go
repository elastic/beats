package schema

import (
	"strings"

	"github.com/elastic/beats/libbeat/logp"
)

type Errors []*Error

func (errs Errors) HasRequiredErrors() bool {
	for _, err := range errs {
		if err.IsType(RequiredType) {
			return true
		}
	}
	return false
}

func (errs Errors) Error() string {
	var required []string
	for _, err := range errs {
		if err.IsType(RequiredType) {
			required = append(required, err.key)
		}
	}
	return "Required fields are missing: " + strings.Join(required, ", ")
}

// Log logs all missing required and optional fields to the debug log.
func (errs Errors) Log() {
	if len(errs) == 0 {
		return
	}
	var optional, required []string

	for _, err := range errs {
		if err.IsType(RequiredType) {
			required = append(required, err.key)
		} else {
			optional = append(optional, err.key)
		}
	}

	log := ""
	if len(required) > 0 {
		log = log + "required: " + strings.Join(required, ",") + "; "
	}

	if len(optional) > 0 {
		log = log + "optional: " + strings.Join(optional, ",") + ";"
	}

	logp.Debug("schema", "Fields missing - %s", log)
}
