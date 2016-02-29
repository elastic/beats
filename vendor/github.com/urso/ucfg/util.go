package ucfg

import (
	"reflect"
	"strings"
)

type tagOptions struct {
	squash bool
}

func parseTags(tag string) (string, tagOptions) {
	s := strings.Split(tag, ",")
	opts := tagOptions{}
	for _, opt := range s[1:] {
		switch opt {
		case "squash", "inline":
			opts.squash = true
		}
	}
	return s[0], opts
}

func fieldName(tagName, structName string) string {
	if tagName != "" {
		return tagName
	}
	return strings.ToLower(structName)
}

func chaseValueInterfaces(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Interface && !v.IsNil() {
		v = v.Elem()
	}
	return v
}

func chaseValuePointers(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}
	return v
}

func chaseValue(v reflect.Value) reflect.Value {
	for (v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface) && !v.IsNil() {
		v = v.Elem()
	}
	return v
}

func chaseTypePointers(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}
