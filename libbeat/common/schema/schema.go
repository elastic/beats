package schema

import (
	"github.com/elastic/beats/libbeat/common"
)

// Schema describes how a map[string]interface{} object can be parsed and converted into
// an event. The conversions can be described using an (optionally nested) common.MapStr
// that contains Conv objects.
type Schema map[string]Mapper

// Mapper interface represents a valid type to be used in a schema.
type Mapper interface {
	// Map applies the Mapper conversion on the data and adds the result
	// to the event on the key.
	Map(key string, event common.MapStr, data map[string]interface{}) *Errors

	HasKey(key string) bool
}

// A Conv object represents a conversion mechanism from the data map to the event map.
type Conv struct {
	Func     Converter // Convertor function
	Key      string    // The key in the data map
	Optional bool      // Whether to log errors if the key is not found
}

// Convertor function type
type Converter func(key string, data map[string]interface{}) (interface{}, error)

// Map applies the conversion on the data and adds the result
// to the event on the key.
func (conv Conv) Map(key string, event common.MapStr, data map[string]interface{}) *Errors {
	value, err := conv.Func(conv.Key, data)
	if err != nil {
		err := NewError(key, err.Error())
		if conv.Optional {
			err.SetType(OptionalType)
		}

		errs := NewErrors()
		errs.AddError(err)
		return errs

	} else {
		event[key] = value
	}
	return nil
}

func (conv Conv) HasKey(key string) bool {
	return conv.Key == key
}

// implements Mapper interface for structure
type Object map[string]Mapper

func (o Object) Map(key string, event common.MapStr, data map[string]interface{}) *Errors {
	subEvent := common.MapStr{}
	errs := applySchemaToEvent(subEvent, data, o)
	event[key] = subEvent
	return errs
}

func (o Object) HasKey(key string) bool {
	return hasKey(key, o)
}

// ApplyTo adds the fields extracted from data, converted using the schema, to the
// event map.
func (s Schema) ApplyTo(event common.MapStr, data map[string]interface{}) (common.MapStr, *Errors) {
	errors := applySchemaToEvent(event, data, s)
	errors.Log()
	return event, errors
}

// Apply converts the fields extracted from data, using the schema, into a new map and reports back the errors.
func (s Schema) Apply(data map[string]interface{}) (common.MapStr, *Errors) {
	return s.ApplyTo(common.MapStr{}, data)
}

// HasKey checks if the key is part of the schema
func (s Schema) HasKey(key string) bool {
	return hasKey(key, s)
}

func hasKey(key string, mappers map[string]Mapper) bool {
	for _, mapper := range mappers {
		if mapper.HasKey(key) {
			return true
		}
	}
	return false
}

func applySchemaToEvent(event common.MapStr, data map[string]interface{}, conversions map[string]Mapper) *Errors {
	errs := NewErrors()
	for key, mapper := range conversions {
		errors := mapper.Map(key, event, data)
		errs.AddErrors(errors)
	}
	return errs
}

// SchemaOption is for adding optional parameters to the conversion
// functions
type SchemaOption func(c Conv) Conv

// The optional flag suppresses the error message in case the key
// doesn't exist or results in an error.
func Optional(c Conv) Conv {
	c.Optional = true
	return c
}

// setOptions adds the optional flags to the Conv object
func SetOptions(c Conv, opts []SchemaOption) Conv {
	for _, opt := range opts {
		c = opt(c)
	}
	return c
}
