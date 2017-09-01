package template

import (
	"errors"
	"strings"

	"github.com/elastic/beats/libbeat/common"
)

var (
	defaultType = "keyword"
)

type Fields []Field

func (f Fields) process(path string, esVersion common.Version, output common.MapStr) error {
	for _, field := range f {

		var mapping common.MapStr
		field.path = path
		field.esVersion = esVersion

		// If not type is defined, it assumes keyword
		if field.Type == "" {
			field.Type = defaultType
		}

		switch field.Type {
		case "ip":
			mapping = field.ip()
		case "scaled_float":
			mapping = field.scaledFloat()
		case "half_float":
			mapping = field.halfFloat()
		case "integer":
			mapping = field.integer()
		case "text":
			mapping = field.text()
		case "keyword":
			mapping = field.keyword()
		case "object":
			mapping = field.object()
		case "array":
			mapping = field.array()
		case "group":
			var newPath string
			if path == "" {
				newPath = field.Name
			} else {
				newPath = path + "." + field.Name
			}
			mapping = common.MapStr{}

			// Combine properties with previous field definitions (if any)
			properties := common.MapStr{}
			key := generateKey(field.Name) + ".properties"
			currentProperties, err := output.GetValue(key)
			if err == nil {
				var ok bool
				properties, ok = currentProperties.(common.MapStr)
				if !ok {
					// This should never happen
					return errors.New(key + " is expected to be a MapStr")
				}
			}

			if err := field.Fields.process(newPath, esVersion, properties); err != nil {
				return err
			}
			mapping["properties"] = properties

		default:
			mapping = field.other()
		}

		if len(mapping) > 0 {
			output.Put(generateKey(field.Name), mapping)
		}
	}

	return nil
}

// HasKey checks if inside fields the given key exists
// The key can be in the form of a.b.c and it will check if the nested field exist
// In case the key is `a` and there is a value `a.b` false is return as it only
// returns true if it's a leave node
func (f Fields) HasKey(key string) bool {
	keys := strings.Split(key, ".")
	return f.hasKey(keys)
}

func (f Fields) hasKey(keys []string) bool {
	// Nothing to compare anymore
	if len(keys) == 0 {
		return false
	}

	key := keys[0]
	keys = keys[1:]

	for _, field := range f {
		if field.Name == key {

			if len(field.Fields) > 0 {
				return field.Fields.hasKey(keys)
			}
			// Last entry in the tree but still more keys
			if len(keys) > 0 {
				return false
			}

			return true
		}
	}
	return false
}
