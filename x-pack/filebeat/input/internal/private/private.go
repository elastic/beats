// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Package private implements field redaction in maps and structs.
package private

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"unsafe"
)

const tooDeep = 100

var privateKey = reflect.ValueOf("private")

// Redact returns a copy of val with any fields or map elements that have been
// marked as private removed. Fields can be marked as private by including a
// sibling string- or []string-valued field or element with the name of the
// private field. The names of fields are interpreted through the tag parameter
// if present. For example if tag is "json", the `json:"<name>"` name would be
// used, falling back to the field name if not present. The tag parameter is
// ignored for map values.
//
// The global parameter indicates a set of dot-separated paths to redact. Paths
// originate at the root of val. If global is used, the resultin redaction is on
// the union of the fields redacted with tags and the fields redacted with the
// global paths.
//
// If a field has a `private:...` tag, its tag value will also be used to
// determine the list of private fields. If the private tag is empty,
// `private:""`, the fields with the tag will be marked as private. Otherwise
// the comma-separated list of names with be used. The list may refer to its
// own field.
func Redact[T any](val T, tag string, global []string) (redacted T, err error) {
	defer func() {
		switch r := recover().(type) {
		case nil:
			return
		case cycle:
			// Make the returned type informative in all cases.
			// If Redact[any](v) is called and we use the zero
			// value, we would return a nil any, which is less
			// informative.
			redacted = reflect.New(reflect.TypeOf(val)).Elem().Interface().(T)
			err = r
		default:
			panic(r)
		}
	}()
	rv := reflect.ValueOf(val)
	switch rv.Kind() {
	case reflect.Map, reflect.Pointer, reflect.Struct:
		return redact(rv, tag, slices.Clone(global), 0, make(map[any]int)).Interface().(T), nil
	default:
		return val, nil
	}
}

func redact(v reflect.Value, tag string, global []string, depth int, seen map[any]int) reflect.Value {
	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			return v
		}
		if depth > tooDeep {
			ident := v.Interface()
			if last, ok := seen[ident]; ok && last < depth {
				panic(cycle{v.Type()})
			}
			seen[ident] = depth
			defer delete(seen, ident)
		}
		return redact(v.Elem(), tag, global, depth+1, seen).Addr()
	case reflect.Interface:
		if v.IsNil() {
			return v
		}
		return redact(v.Elem(), tag, global, depth+1, seen)
	case reflect.Array:
		if v.Len() == 0 {
			return v
		}
		r := reflect.New(v.Type()).Elem()
		for i := 0; i < v.Len(); i++ {
			r.Index(i).Set(redact(v.Index(i), tag, global, depth+1, seen))
		}
		return r
	case reflect.Slice:
		if v.Len() == 0 {
			return v
		}
		if depth > tooDeep {
			ident := struct {
				data unsafe.Pointer
				len  int
			}{
				v.UnsafePointer(),
				v.Len(),
			}
			if last, ok := seen[ident]; ok && last < depth {
				panic(cycle{v.Type()})
			}
			seen[ident] = depth
			defer delete(seen, ident)
		}
		r := reflect.MakeSlice(v.Type(), v.Len(), v.Cap())
		for i := 0; i < v.Len(); i++ {
			r.Index(i).Set(redact(v.Index(i), tag, global, depth+1, seen))
		}
		return r
	case reflect.Map:
		if v.IsNil() {
			return v
		}
		if depth > tooDeep {
			ident := v.UnsafePointer()
			if last, ok := seen[ident]; ok && last < depth {
				panic(cycle{v.Type()})
			}
			seen[ident] = depth
			defer delete(seen, ident)
		}
		private := nextStep(global)
		if privateKey.CanConvert(v.Type().Key()) {
			p := v.MapIndex(privateKey.Convert(v.Type().Key()))
			if p.IsValid() && p.CanInterface() {
				switch p := p.Interface().(type) {
				case string:
					private = append(private, p)
				case []string:
					private = append(private, p...)
				case []any:
					for _, s := range p {
						private = append(private, fmt.Sprint(s))
					}
				}
			}
		}
		r := reflect.MakeMap(v.Type())
		it := v.MapRange()
		for it.Next() {
			name := it.Key().String()
			if slices.Contains(private, name) {
				continue
			}
			r.SetMapIndex(it.Key(), redact(it.Value(), tag, nextPath(name, global), depth+1, seen))
		}
		return r
	case reflect.Struct:
		private := nextStep(global)
		rt := v.Type()
		names := make([]string, rt.NumField())
		for i := range names {
			f := rt.Field(i)

			// Look for `private:` tags.
			p, ok := f.Tag.Lookup("private")
			if ok {
				if p != "" {
					private = append(private, strings.Split(p, ",")...)
				} else {
					if tag == "" {
						names[i] = f.Name
						private = append(private, f.Name)
					} else {
						p = f.Tag.Get(tag)
						if p != "" {
							name, _, _ := strings.Cut(p, ",")
							names[i] = name
							private = append(private, name)
						}
					}
				}
			}

			// Look after Private fields if we are not using a tag.
			if tag == "" {
				names[i] = f.Name
				if f.Name == "Private" {
					switch p := v.Field(i).Interface().(type) {
					case string:
						private = append(private, p)
					case []string:
						private = append(private, p...)
					}
				}
				continue
			}

			// If we are using a tag, look for `tag:"<private>"`
			// falling back to fields named Private if no tag is
			// present.
			p = f.Tag.Get(tag)
			var name string
			if p == "" {
				name = f.Name
			} else {
				name, _, _ = strings.Cut(p, ",")
			}
			names[i] = name
			if name == "private" {
				switch p := v.Field(i).Interface().(type) {
				case string:
					private = append(private, p)
				case []string:
					private = append(private, p...)
				}
			}
		}

		r := reflect.New(v.Type()).Elem()
		for i := 0; i < v.NumField(); i++ {
			f := v.Field(i)
			if f.IsZero() || !rt.Field(i).IsExported() {
				continue
			}
			if slices.Contains(private, names[i]) {
				continue
			}
			if r.Field(i).CanSet() {
				r.Field(i).Set(redact(f, tag, nextPath(names[i], global), depth+1, seen))
			}
		}
		return r
	}
	return v
}

func nextStep(global []string) (private []string) {
	if len(global) == 0 {
		return nil
	}
	private = make([]string, 0, len(global))
	for _, s := range global {
		key, _, more := strings.Cut(s, ".")
		if !more {
			private = append(private, key)
		}
	}
	return private
}

func nextPath(step string, global []string) []string {
	if len(global) == 0 {
		return nil
	}
	step += "."
	next := make([]string, 0, len(global))
	for _, s := range global {
		if !strings.HasPrefix(s, step) {
			continue
		}
		next = append(next, s[len(step):])
	}
	return next
}

type cycle struct {
	typ reflect.Type
}

func (e cycle) Error() string {
	return fmt.Sprintf("cycle including %s", e.typ)
}
