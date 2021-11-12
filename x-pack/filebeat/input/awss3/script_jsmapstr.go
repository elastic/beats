// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"github.com/dop251/goja"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
)

type jsMapStr struct {
	vm    *goja.Runtime
	obj   *goja.Object
	inner common.MapStr
}

func newJSMapStr(s *session, m common.MapStr) (*jsMapStr, error) {
	e := &jsMapStr{
		vm:    s.vm,
		obj:   s.vm.NewObject(),
		inner: m,
	}
	e.init()
	return e, nil
}

func newJSMapStrConstructor(s *session) func(call goja.ConstructorCall) *goja.Object {
	return func(call goja.ConstructorCall) *goja.Object {
		if len(call.Arguments) != 1 {
			panic(errors.New("MapStr constructor requires one argument"))
		}

		a0 := call.Argument(0).Export()

		var fields common.MapStr
		switch v := a0.(type) {
		case map[string]interface{}:
			fields = v
		case common.MapStr:
			fields = v
		default:
			panic(errors.Errorf("Event constructor requires a "+
				"map[string]interface{} argument but got %T", a0))
		}

		evt := &jsMapStr{
			vm:    s.vm,
			obj:   call.This,
			inner: fields,
		}
		evt.init()
		return nil
	}
}

func (e *jsMapStr) init() {
	e.obj.Set("Get", e.get)
	e.obj.Set("Put", e.put)
	e.obj.Set("Rename", e.rename)
	e.obj.Set("Delete", e.delete)
	e.obj.Set("AppendTo", e.appendTo)
}

// get returns the specified field. If the field does not exist then null is
// returned. If no field is specified then it returns entire object.
//
//	// javascript
// 	var v = evt.Get("key1.key2");
//
func (e *jsMapStr) get(call goja.FunctionCall) goja.Value {
	a0 := call.Argument(0)
	if goja.IsUndefined(a0) {
		// event.Get() is the same as event.fields (but slower).
		return e.vm.ToValue(e.inner)
	}

	v, err := e.inner.GetValue(a0.String())
	if err != nil {
		return goja.Null()
	}

	return e.vm.ToValue(v)
}

// put writes a value to the map. If there was a previous value assigned to
// the given field then the old object is returned. It throws an exception if
// you try to write a to a field where one of the intermediate values is not
// an object.
//
//	// javascript
// 	evt.Put("key1.key2", "value");
// 	evt.Put("key", {"a": 1, "b": 2});
//
func (e *jsMapStr) put(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) != 2 {
		panic(errors.New("Put requires two arguments (key and value)"))
	}

	key := call.Argument(0).String()
	value := call.Argument(1).Export()

	old, err := e.inner.Put(key, value)
	if err != nil {
		panic(err)
	}
	return e.vm.ToValue(old)
}

// rename moves a value from one key to another. It returns true on success.
//
//	// javascript
// 	evt.Rename("key1", "key2");
//
func (e *jsMapStr) rename(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) != 2 {
		panic(errors.New("Rename requires two arguments (from and to)"))
	}

	from := call.Argument(0).String()
	to := call.Argument(1).String()

	if _, err := e.inner.GetValue(to); err == nil {
		// Fields cannot be overwritten. Either the target field has to be
		// deleted or renamed.
		return e.vm.ToValue(false)
	}

	fromValue, err := e.inner.GetValue(from)
	if err != nil {
		return e.vm.ToValue(false)
	}

	// Deletion must happen first to support cases where a becomes a.b.
	if err = e.inner.Delete(from); err != nil {
		return e.vm.ToValue(false)
	}

	if _, err = e.inner.Put(to, fromValue); err != nil {
		// Undo
		e.inner.Put(from, fromValue)
		return e.vm.ToValue(false)
	}

	return e.vm.ToValue(true)
}

// delete deletes a key from the object. If returns true on success.
//
//	// javascript
// 	evt.Delete("key1");
//
func (e *jsMapStr) delete(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) != 1 {
		panic(errors.New("Delete requires one argument"))
	}

	key := call.Argument(0).String()

	if err := e.inner.Delete(key); err != nil {
		return e.vm.ToValue(false)
	}
	return e.vm.ToValue(true)
}

// appendTo is a specialized Put method that converts any existing value to
// an array and appends the value if it does not already exist. If there is an
// existing value that's not a string or array of strings then an exception is
// thrown.
//
//	// javascript
//	evt.AppendTo("arr", "val");
//
func (e *jsMapStr) appendTo(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) != 2 {
		panic(errors.New("AppendTo requires two arguments (field and value)"))
	}

	field := call.Argument(0).String()
	value := call.Argument(1).String()

	if err := appendString(e.inner, field, value, false); err != nil {
		panic(err)
	}
	return goja.Undefined()
}

func appendString(m common.MapStr, field, value string, alwaysArray bool) error {
	list, _ := m.GetValue(field)
	switch v := list.(type) {
	case nil:
		if alwaysArray {
			m.Put(field, []string{value})
		} else {
			m.Put(field, value)
		}
	case string:
		if value != v {
			m.Put(field, []string{v, value})
		}
	case []string:
		for _, existingTag := range v {
			if value == existingTag {
				// Duplicate
				return nil
			}
		}
		m.Put(field, append(v, value))
	case []interface{}:
		for _, existingTag := range v {
			if value == existingTag {
				// Duplicate
				return nil
			}
		}
		m.Put(field, append(v, value))
	default:
		return errors.Errorf("unexpected type %T found for %v field", list, field)
	}
	return nil
}
