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

// An Action says what RedactFunc does with an element.
type Action int

const (
	// Keep copies the element, visiting the entries within its value.
	Keep Action = iota

	// Replace uses the value returned by the visiting function.
	Replace

	// Remove drops a map element. A struct field or slice or array element
	// cannot be dropped, so it is left as the zero value.
	Remove
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
func Redact[T any](val T, tag string, global []string, replace ...Replacer) (redacted T, err error) {
	reps, err := compileReplacers(replace)
	if err != nil {
		return redacted, err
	}
	switch reflect.ValueOf(val).Kind() {
	case reflect.Map, reflect.Pointer, reflect.Struct:
		return RedactFunc(val, tag, redactVisitor(reps, tag, slices.Clone(global)))
	default:
		return val, nil
	}
}

// RedactFunc returns a copy of val in which fn has had the chance to replace,
// remove or descend into each map element, struct field and slice or array
// element. The names of struct fields are interpreted through the tag
// parameter as for Redact, and zero-valued and unexported struct fields are
// not visited. Values that are not maps, pointers, structs, slices or arrays
// are returned unchanged. The elements of a slice or array at the root are
// visited with an empty path.
//
// For each element fn is given a Visit and returns a replacement value and an
// Action. The replacement is only used if the action is Replace. If the action
// is Keep, RedactFunc visits the entries of the element's value. A replacement
// must be assignable to the element it replaces.
func RedactFunc[T any](val T, tag string, fn func(Visit) (reflect.Value, Action)) (redacted T, err error) {
	defer func() {
		switch r := recover().(type) {
		case nil:
			return
		case cycle:
			// Make the returned type informative in all cases.
			// If RedactFunc[any](v) is called and we use the zero
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
	case reflect.Map, reflect.Pointer, reflect.Struct, reflect.Slice, reflect.Array:
		got := walk(rv, fn, tag, nil, 0, make(map[any]int)).Interface()
		r, ok := got.(T)
		if !ok {
			return val, fmt.Errorf("redacted %T is not a %T", got, val)
		}
		return r, nil
	default:
		return val, nil
	}
}

// A Visit describes an element that RedactFunc is visiting. It is valid only
// for the duration of the call to the function that receives it.
type Visit struct {
	// Path is the names of the map keys and struct fields from the root to
	// the element, ending with the element's own name. Slice and array
	// elements are visited with the path of the slice or array that holds
	// them, since they have no name of their own. Path must not be modified.
	Path []string

	// Parent is the map, struct, slice or array that holds the element.
	Parent reflect.Value

	// Value is the element's value.
	Value reflect.Value
}

// MaskStrings returns a function for RedactFunc that applies fn to each
// string-kinded value, including json.Number, passing the path of the value
// and its text. If fn reports true, the value is replaced by the string it
// returns. A replacement in an interface-typed element is a plain string, so
// replacing a json.Number does not leave text that cannot be marshaled as a
// number. Elements are never removed. Pointers to strings are not masked.
func MaskStrings(fn func(path []string, s string) (string, bool)) func(Visit) (reflect.Value, Action) {
	return func(vis Visit) (reflect.Value, Action) {
		v := vis.Value
		if v.Kind() == reflect.Interface {
			if v.IsNil() {
				return reflect.Value{}, Keep
			}
			v = v.Elem()
		}
		if v.Kind() != reflect.String {
			return reflect.Value{}, Keep
		}
		s, ok := fn(vis.Path, v.String())
		if !ok {
			return reflect.Value{}, Keep
		}
		if vis.Value.Kind() == reflect.Interface {
			return reflect.ValueOf(s), Replace
		}
		return reflect.ValueOf(s).Convert(v.Type()), Replace
	}
}

// Replacer is a function that will return a redaction replacement
// for the provided type. It must be a func(T) T.
type Replacer any

// NewStringReplacer returns a string Replacer that returns s.
func NewStringReplacer(s string) Replacer {
	return func(string) string {
		return s
	}
}

// NewBytesReplacer returns a []byte Replacer that returns the bytes
// representation of s.
func NewBytesReplacer(s string) Replacer {
	return func([]byte) []byte {
		return []byte(s)
	}
}

type replacers map[reflect.Type]func(reflect.Value) reflect.Value

func compileReplacers(replace []Replacer) (replacers, error) {
	reps := make(replacers)
	for _, r := range replace {
		rv := reflect.ValueOf(r)
		rt := rv.Type()
		if rt.Kind() != reflect.Func {
			return nil, fmt.Errorf("replacer is not a function: %T", r)
		}
		if n := rt.NumIn(); n != 1 {
			return nil, fmt.Errorf("incorrect number of arguments for replacer: %d != 1", n)
		}
		if n := rt.NumOut(); n != 1 {
			return nil, fmt.Errorf("incorrect number of return values from replacer: %d != 1", n)
		}
		in, out := rt.In(0), rt.Out(0)
		if in != out {
			return nil, fmt.Errorf("replacer does not preserve type: fn(%s) %s", in, out)
		}
		if _, exists := reps[in]; exists {
			return nil, fmt.Errorf("multiple replacers for %s", in)
		}
		reps[in] = func(v reflect.Value) reflect.Value {
			return rv.Call([]reflect.Value{v})[0]
		}
	}
	if len(reps) == 0 {
		reps = nil
	}
	return reps, nil
}

// redactVisitor returns the function that gives RedactFunc the semantics of
// Redact. An element is private if its name is in the global paths that remain
// at its depth, is listed by a sibling private element of a map, or is listed
// by the tags and fields of its struct. A private element is replaced by the
// result of replacing nested values within it, or removed if no replacement
// is made.
func redactVisitor(reps replacers, tag string, global []string) func(Visit) (reflect.Value, Action) {
	structs := make(map[reflect.Type]structInfo)
	return func(vis Visit) (reflect.Value, Action) {
		var private []string
		switch vis.Parent.Kind() {
		case reflect.Map:
			private = mapPrivate(vis.Parent)
		case reflect.Struct:
			rt := vis.Parent.Type()
			info, ok := structs[rt]
			if !ok {
				info = newStructInfo(rt, tag)
				structs[rt] = info
			}
			private = info.private(vis.Parent)
		default:
			// Slice and array elements have no name of their own.
			return reflect.Value{}, Keep
		}
		name := vis.Path[len(vis.Path)-1]
		private = append(nextStep(globalAt(global, vis.Path[:len(vis.Path)-1])), private...)
		if !slices.Contains(private, name) {
			return reflect.Value{}, Keep
		}
		v := replaceNestedWithin(vis.Value, reps)
		if !v.IsValid() {
			return reflect.Value{}, Remove
		}
		return v, Replace
	}
}

// globalAt returns the global paths that remain after descending through the
// names in path.
func globalAt(global, path []string) []string {
	for _, name := range path {
		global = nextPath(name, global)
	}
	return global
}

// mapPrivate returns the names listed by the map's "private" element.
func mapPrivate(v reflect.Value) []string {
	if !privateKey.CanConvert(v.Type().Key()) {
		return nil
	}
	p := v.MapIndex(privateKey.Convert(v.Type().Key()))
	if !p.IsValid() || !p.CanInterface() {
		return nil
	}
	switch p := p.Interface().(type) {
	case string:
		return []string{p}
	case []string:
		return p
	case []any:
		private := make([]string, 0, len(p))
		for _, s := range p {
			private = append(private, fmt.Sprint(s))
		}
		return private
	}
	return nil
}

// structInfo holds what is needed to find the private names of the structs of
// one type: the names that the `private:` tags list, which depend only on the
// type, and the fields that list names in their values.
type structInfo struct {
	tagged []string
	lists  []int
}

func newStructInfo(rt reflect.Type, tag string) structInfo {
	var info structInfo
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)

		// Look for `private:` tags.
		p, ok := f.Tag.Lookup("private")
		if ok {
			if p != "" {
				info.tagged = append(info.tagged, strings.Split(p, ",")...)
			} else {
				if tag == "" {
					info.tagged = append(info.tagged, f.Name)
				} else {
					p = f.Tag.Get(tag)
					if p != "" {
						name, _, _ := strings.Cut(p, ",")
						info.tagged = append(info.tagged, name)
					}
				}
			}
		}

		// Look after Private fields. If we are using a tag, fall back
		// to the name that the tag gives.
		if fieldName(f, tag) == privateName(tag) {
			info.lists = append(info.lists, i)
		}
	}
	return info
}

// private returns the names listed by the `private:` tags and Private fields
// of the struct v, which must be of the type that info was made for.
func (i structInfo) private(v reflect.Value) []string {
	private := slices.Clone(i.tagged)
	for _, n := range i.lists {
		switch p := v.Field(n).Interface().(type) {
		case string:
			private = append(private, p)
		case []string:
			private = append(private, p...)
		}
	}
	return private
}

// privateName returns the name of the field that lists private fields when
// names are interpreted through tag.
func privateName(tag string) string {
	if tag == "" {
		return "Private"
	}
	return "private"
}

// fieldName returns the name of f, interpreted through tag if it is not empty
// and falling back to the field name if the field has no tag.
func fieldName(f reflect.StructField, tag string) string {
	if tag == "" {
		return f.Name
	}
	p := f.Tag.Get(tag)
	if p == "" {
		return f.Name
	}
	name, _, _ := strings.Cut(p, ",")
	return name
}

// walk returns a copy of v in which fn has been applied to each element.
func walk(v reflect.Value, fn func(Visit) (reflect.Value, Action), tag string, path []string, depth int, seen map[any]int) reflect.Value {
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
		return walk(v.Elem(), fn, tag, path, depth+1, seen).Addr()
	case reflect.Interface:
		if v.IsNil() {
			return v
		}
		return walk(v.Elem(), fn, tag, path, depth+1, seen)
	case reflect.Array:
		if v.Len() == 0 {
			return v
		}
		r := reflect.New(v.Type()).Elem()
		for i := 0; i < v.Len(); i++ {
			r.Index(i).Set(walkElem(v, i, fn, tag, path, depth, seen))
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
			r.Index(i).Set(walkElem(v, i, fn, tag, path, depth, seen))
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
		r := reflect.MakeMap(v.Type())
		it := v.MapRange()
		for it.Next() {
			p := append(slices.Clip(path), it.Key().String())
			repl, act := fn(Visit{Path: p, Parent: v, Value: it.Value()})
			switch act {
			case Replace:
				r.SetMapIndex(it.Key(), repl)
			case Remove:
			default:
				r.SetMapIndex(it.Key(), walk(it.Value(), fn, tag, p, depth+1, seen))
			}
		}
		return r
	case reflect.Struct:
		rt := v.Type()
		r := reflect.New(rt).Elem()
		for i := 0; i < v.NumField(); i++ {
			f := v.Field(i)
			if f.IsZero() || !rt.Field(i).IsExported() {
				continue
			}
			p := append(slices.Clip(path), fieldName(rt.Field(i), tag))
			repl, act := fn(Visit{Path: p, Parent: v, Value: f})
			switch act {
			case Replace:
				r.Field(i).Set(repl)
			case Remove:
			default:
				if r.Field(i).CanSet() {
					r.Field(i).Set(walk(f, fn, tag, p, depth+1, seen))
				}
			}
		}
		return r
	}
	return v
}

// canHoldString reports whether a value of kind k is a string or can contain
// one. Elements of other kinds, such as the bytes of a []byte, are copied
// without being visited.
func canHoldString(k reflect.Kind) bool {
	switch k {
	case reflect.String, reflect.Interface, reflect.Pointer, reflect.Map,
		reflect.Slice, reflect.Array, reflect.Struct:
		return true
	default:
		return false
	}
}

// walkElem returns the copy of the ith element of the slice or array v.
func walkElem(v reflect.Value, i int, fn func(Visit) (reflect.Value, Action), tag string, path []string, depth int, seen map[any]int) reflect.Value {
	e := v.Index(i)
	if !canHoldString(e.Kind()) {
		return e
	}
	repl, act := fn(Visit{Path: path, Parent: v, Value: e})
	switch act {
	case Replace:
		return repl
	case Remove:
		return reflect.Zero(e.Type())
	default:
		return walk(e, fn, tag, path, depth+1, seen)
	}
}

// replaceNestedWithin replaces deeply nested values in pointer, interface and
// array/slice chains. If a replacement is not made an invalid reflect.Value
// is returned. If elements are not replaced by a replacer, it is set to the
// zero value for the type.
func replaceNestedWithin(v reflect.Value, reps replacers) reflect.Value {
	if len(reps) == 0 || !v.IsValid() {
		// No replacer, or an invalid value, so fall back to removal.
		return reflect.Value{}
	}
	if rep, ok := reps[v.Type()]; ok {
		return rep(v)
	}
	switch v.Kind() {
	case reflect.Pointer:
		r := replaceNestedWithin(v.Elem(), reps)
		if !r.IsValid() {
			return r
		}
		return r.Addr()
	case reflect.Interface:
		r := replaceNestedWithin(v.Elem(), reps)
		if !r.IsValid() {
			return r
		}
		i := reflect.New(v.Type()).Elem()
		i.Set(r)
		return i
	case reflect.Array:
		a := reflect.New(v.Type()).Elem()
		wasSet := false
		for i := 0; i < v.Len(); i++ {
			r := replaceNestedWithin(v.Index(i), reps)
			if r.IsValid() {
				wasSet = true
				a.Index(i).Set(r)
			}
		}
		if !wasSet {
			return reflect.Value{}
		}
		return a
	case reflect.Slice:
		s := reflect.MakeSlice(v.Type(), v.Len(), v.Cap())
		wasSet := false
		for i := 0; i < v.Len(); i++ {
			r := replaceNestedWithin(v.Index(i), reps)
			if r.IsValid() {
				wasSet = true
				s.Index(i).Set(r)
			}
		}
		if !wasSet {
			return reflect.Value{}
		}
		return s
	default:
		// Could not catch, fall back to removal.
		return reflect.Value{}
	}
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
