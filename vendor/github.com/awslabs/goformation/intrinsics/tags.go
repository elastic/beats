package intrinsics

import (
	"reflect"

	yaml "github.com/sanathkr/go-yaml"
)

var allTags = []string{"Ref", "GetAtt", "Base64", "FindInMap", "GetAZs",
	"ImportValue", "Join", "Select", "Split", "Sub",
}

type tagUnmarshalerType struct {
}

func (t *tagUnmarshalerType) UnmarshalYAMLTag(tag string, fieldValue reflect.Value) reflect.Value {

	prefix := "Fn::"
	if tag == "Ref" || tag == "Condition" {
		prefix = ""
	}

	tag = prefix + tag

	output := reflect.ValueOf(make(map[string]interface{}))
	key := reflect.ValueOf(tag)

	output.SetMapIndex(key, fieldValue)

	return output
}

var tagUnmarshaller = &tagUnmarshalerType{}

func registerTagMarshallers() {
	for _, tag := range allTags {
		yaml.RegisterTagUnmarshaler("!"+tag, tagUnmarshaller)
	}
}

func unregisterTagMarshallers() {
	for _, tag := range allTags {
		yaml.RegisterTagUnmarshaler("!"+tag, tagUnmarshaller)
	}
}
