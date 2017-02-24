package template

import "github.com/elastic/beats/libbeat/common"

var (
	defaultType = "keyword"
)

type Fields []Field

func (f Fields) process(path string, esVersion Version) common.MapStr {
	output := common.MapStr{}

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
			mapping = common.MapStr{
				"properties": field.Fields.process(newPath, esVersion),
			}
		default:
			mapping = field.other()
		}

		output.Put(generateKey(field.Name), mapping)
	}

	return output
}
