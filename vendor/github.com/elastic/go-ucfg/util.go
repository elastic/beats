package ucfg

import (
	"reflect"
	"strings"
)

type tagOptions struct {
	squash bool
}

var noTagOpts = tagOptions{}

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

// tryTConfig tries to convert input value into addressable Config by converting
// to *Config first. If value is convertible to Config, but not addressable a new
// value is allocated in order to guarantee returned value of type Config is
// addressable. Returns false if type value is not convertible to TConfig.
func tryTConfig(value reflect.Value) (reflect.Value, bool) {
	v := chaseValue(value)
	t := v.Type()

	if t == tConfig {
		v := pointerize(tConfigPtr, tConfig, v)
		return v.Elem(), true
	}

	if !t.ConvertibleTo(tConfig) {
		return reflect.Value{}, false
	}

	v = pointerize(reflect.PtrTo(v.Type()), v.Type(), v)
	if !v.Type().ConvertibleTo(tConfigPtr) {
		return reflect.Value{}, false
	}

	v = v.Convert(tConfigPtr)
	return v.Elem(), true
}

func pointerize(t, base reflect.Type, v reflect.Value) reflect.Value {
	if t == base {
		return v
	}

	if t.Kind() == reflect.Interface {
		return v
	}

	for t != v.Type() {
		if !v.CanAddr() {
			tmp := reflect.New(v.Type())
			tmp.Elem().Set(v)
			v = tmp
		} else {
			v = v.Addr()
		}
	}
	return v
}

func isInt(k reflect.Kind) bool {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return true
	default:
		return false
	}
}

func isUint(k reflect.Kind) bool {
	switch k {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	default:
		return false
	}
}

func implementsUnpacker(v reflect.Value) (reflect.Value, bool) {
	for {
		if v.Type().Implements(tUnpacker) {
			return v, true
		}

		if !v.CanAddr() {
			break
		}
		v = v.Addr()
	}
	return reflect.Value{}, false
}

func typeIsUnpacker(t reflect.Type) (reflect.Value, bool) {
	if t.Implements(tUnpacker) {
		return reflect.New(t).Elem(), true
	}

	if reflect.PtrTo(t).Implements(tUnpacker) {
		return reflect.New(t), true
	}

	return reflect.Value{}, false
}

func unpackWith(ctx context, meta *Meta, v reflect.Value, with interface{}) Error {
	err := v.Interface().(Unpacker).Unpack(with)
	if err != nil {
		return raisePathErr(err, meta, "", ctx.path("."))
	}
	return nil
}
