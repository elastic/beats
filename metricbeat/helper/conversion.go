package helper

import (
	"fmt"
	"strconv"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// Schema describes how a map[string]string object can be parsed and converted into
// an event. The conversions can be described using an (optionally nested) common.MapStr
// that contains Conv objects.
type Schema struct {
	conversions common.MapStr
}

// A Conv object represents a conversion mechanism from the data map to the event map.
type Conv struct {
	Func     Convertor // Convertor function
	Key      string    // The key in the data map
	Optional bool      // Whether to log errors if the key is not found
}

// Convertor function type
type Convertor func(key string, data map[string]string) (interface{}, error)

// NewSchema creates a new converting schema.
func NewSchema(conversions common.MapStr) Schema {
	return Schema{conversions}
}

// ApplyTo adds the fields extracted from data, converted using the schema, to the
// event map.
func (s Schema) ApplyTo(event common.MapStr, data map[string]string) common.MapStr {
	applySchemaToEvent(event, data, s.conversions)
	return event
}

// Apply converts the fields extracted from data, using the schema, into a new map.
func (s Schema) Apply(data map[string]string) common.MapStr {
	return s.ApplyTo(common.MapStr{}, data)
}

func applySchemaToEvent(event common.MapStr, data map[string]string, conversions common.MapStr) {
	for key, conversion := range conversions {
		switch conversion.(type) {
		case Conv:
			conv := conversion.(Conv)
			value, err := conv.Func(conv.Key, data)
			if err != nil {
				if !conv.Optional {
					logp.Err("Error on field '%s': %v", key, err)
				}
			} else {
				event[key] = value
			}
		case common.MapStr:
			subEvent := common.MapStr{}
			applySchemaToEvent(subEvent, data, conversion.(common.MapStr))
			event[key] = subEvent
		default:
			logp.Err("Unexpected type for '%s' in schema: %T", key, conversion)
		}
	}
}

// ToBool converts value to bool. In case of error, returns false
func ToBool(key string, data map[string]string) (interface{}, error) {

	exists := checkExist(key, data)
	if !exists {
		return false, fmt.Errorf("Key `%s` not found", key)
	}

	value, err := strconv.ParseBool(data[key])
	if err != nil {
		return false, fmt.Errorf("Error converting param to bool: %s", key)
	}

	return value, nil
}

// Bool creates a Conv object for parsing booleans
func Bool(key string, opts ...SchemaOption) Conv {
	return setOptions(Conv{Key: key, Func: ToBool}, opts)
}

// ToFloat converts value to float64. In case of error, returns 0.0
func ToFloat(key string, data map[string]string) (interface{}, error) {

	exists := checkExist(key, data)
	if !exists {
		return false, fmt.Errorf("Key `%s` not found", key)
	}

	value, err := strconv.ParseFloat(data[key], 64)
	if err != nil {
		return 0.0, fmt.Errorf("Error converting param to float: %s", key)
	}

	return value, nil
}

// Float creates a Conv object for parsing floats
func Float(key string, opts ...SchemaOption) Conv {
	return setOptions(Conv{Key: key, Func: ToFloat}, opts)
}

// ToInt converts value to int. In case of error, returns 0
func ToInt(key string, data map[string]string) (interface{}, error) {

	exists := checkExist(key, data)
	if !exists {
		return false, fmt.Errorf("Key `%s` not found", key)
	}

	value, err := strconv.ParseInt(data[key], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("Error converting param to int: %s", key)
	}

	return value, nil
}

// Int creates a Conv object for parsing integers
func Int(key string, opts ...SchemaOption) Conv {
	return setOptions(Conv{Key: key, Func: ToInt}, opts)
}

// ToStr converts value to str. In case of error, returns ""
func ToStr(key string, data map[string]string) (interface{}, error) {

	exists := checkExist(key, data)
	if !exists {
		return false, fmt.Errorf("Key `%s` not found", key)
	}

	return data[key], nil
}

// Str creates a Conv object for parsing strings
func Str(key string, opts ...SchemaOption) Conv {
	return setOptions(Conv{Key: key, Func: ToStr}, opts)
}

// checkExists checks if a key exists in the given data set
func checkExist(key string, data map[string]string) bool {
	_, ok := data[key]
	return ok
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
func setOptions(c Conv, opts []SchemaOption) Conv {
	for _, opt := range opts {
		c = opt(c)
	}
	return c
}
