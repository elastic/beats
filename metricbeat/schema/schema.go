package schema

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// Schema describes how a map[string]interface{} object can be parsed and converted into
// an event. The conversions can be described using an (optionally nested) common.MapStr
// that contains Conv objects.
type Schema map[string]Mapper

// Mapper interface represents a valid type to be used in a schema.
type Mapper interface {
	// Map applies the Mapper conversion on the data and adds the result
	// to the event on the key.
	Map(key string, event common.MapStr, data map[string]interface{})
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
func (conv Conv) Map(key string, event common.MapStr, data map[string]interface{}) {
	value, err := conv.Func(conv.Key, data)
	if err != nil {
		if !conv.Optional {
			logp.Err("Error on field '%s': %v", key, err)
		}
	} else {
		event[key] = value
	}
}

// implements Mapper interface for structure
type Object map[string]Mapper

func (o Object) Map(key string, event common.MapStr, data map[string]interface{}) {
	subEvent := common.MapStr{}
	applySchemaToEvent(subEvent, data, o)
	event[key] = subEvent
}

// ApplyTo adds the fields extracted from data, converted using the schema, to the
// event map.
func (s Schema) ApplyTo(event common.MapStr, data map[string]interface{}) common.MapStr {
	applySchemaToEvent(event, data, s)
	return event
}

// Apply converts the fields extracted from data, using the schema, into a new map.
func (s Schema) Apply(data map[string]interface{}) common.MapStr {
	return s.ApplyTo(common.MapStr{}, data)
}

func applySchemaToEvent(event common.MapStr, data map[string]interface{}, conversions map[string]Mapper) {
	for key, mapper := range conversions {
		mapper.Map(key, event, data)
	}
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
