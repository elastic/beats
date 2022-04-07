// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package javascript

import (
	"github.com/dop251/goja"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v8/libbeat/beat"
	"github.com/elastic/beats/v8/libbeat/common"
)

// IMPORTANT:
// This is the user facing API within Javascript processors. Do not make
// breaking changes to the JS methods. If you must make breaking changes then
// create a new version and require the user to specify an API version in their
// configuration (e.g. api_version: 2).

type beatEventV0 struct {
	vm        *goja.Runtime
	obj       *goja.Object
	inner     *beat.Event
	cancelled bool
}

func newBeatEventV0(s Session) (Event, error) {
	e := &beatEventV0{
		vm:  s.Runtime(),
		obj: s.Runtime().NewObject(),
	}
	e.init()
	return e, nil
}

func newBeatEventV0Constructor(s Session) func(call goja.ConstructorCall) *goja.Object {
	return func(call goja.ConstructorCall) *goja.Object {
		if len(call.Arguments) != 1 {
			panic(errors.New("Event constructor requires one argument"))
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

		evt := &beatEventV0{
			vm:  s.Runtime(),
			obj: call.This,
		}
		evt.init()
		evt.reset(&beat.Event{Fields: fields})
		return nil
	}
}

func (e *beatEventV0) init() {
	e.obj.Set("Get", e.get)
	e.obj.Set("Put", e.put)
	e.obj.Set("Rename", e.rename)
	e.obj.Set("Delete", e.delete)
	e.obj.Set("Cancel", e.cancel)
	e.obj.Set("Tag", e.tag)
	e.obj.Set("AppendTo", e.appendTo)
}

// reset the event so that it can be reused to wrap another event.
func (e *beatEventV0) reset(b *beat.Event) error {
	e.inner = b
	e.cancelled = false
	e.obj.Set("_private", e)
	e.obj.Set("fields", e.vm.ToValue(e.inner.Fields))
	return nil
}

// Wrapped returns the wrapped beat.Event.
func (e *beatEventV0) Wrapped() *beat.Event {
	return e.inner
}

// JSObject returns the goja.Value that represents the event within the
// Javascript runtime.
func (e *beatEventV0) JSObject() goja.Value {
	return e.obj
}

// get returns the specified field. If the field does not exist then null is
// returned. If no field is specified then it returns entire object.
//
//	// javascript
// 	var dataset = evt.Get("event.dataset");
//
func (e *beatEventV0) get(call goja.FunctionCall) goja.Value {
	a0 := call.Argument(0)
	if goja.IsUndefined(a0) {
		// event.Get() is the same as event.fields (but slower).
		return e.vm.ToValue(e.inner.Fields)
	}

	v, err := e.inner.GetValue(a0.String())
	if err != nil {
		return goja.Null()
	}

	return e.vm.ToValue(v)
}

// put writes a value to the event. If there was a previous value assigned to
// the given field then the old object is returned. It throws an exception if
// you try to write a to a field where one of the intermediate values is not
// an object.
//
//	// javascript
// 	evt.Put("event.action", "process-created");
// 	evt.Put("geo.location", {"lon": -73.614830, "lat": 45.505918});
//
func (e *beatEventV0) put(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) != 2 {
		panic(errors.New("Put requires two arguments (key and value)"))
	}

	key := call.Argument(0).String()
	value := call.Argument(1).Export()

	old, err := e.inner.PutValue(key, value)
	if err != nil {
		panic(err)
	}
	return e.vm.ToValue(old)
}

// rename moves a value from one key to another. It returns true on success.
//
//	// javascript
// 	evt.Rename("src_ip", "source.ip");
//
func (e *beatEventV0) rename(call goja.FunctionCall) goja.Value {
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

	if _, err = e.inner.PutValue(to, fromValue); err != nil {
		// Undo
		e.inner.PutValue(from, fromValue)
		return e.vm.ToValue(false)
	}

	return e.vm.ToValue(true)
}

// delete deletes a key from the object. If returns true on success.
//
//	// javascript
// 	evt.Delete("http.request.headers.authorization");
//
func (e *beatEventV0) delete(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) != 1 {
		panic(errors.New("Delete requires one argument"))
	}

	key := call.Argument(0).String()

	if err := e.inner.Delete(key); err != nil {
		return e.vm.ToValue(false)
	}
	return e.vm.ToValue(true)
}

// IsCancelled returns true if the event has been canceled.
func (e *beatEventV0) IsCancelled() bool {
	return e.cancelled
}

// Cancel marks the event as cancelled. When the processor returns the event
// will be dropped.
func (e *beatEventV0) Cancel() {
	e.cancelled = true
}

// cancel marks the event as cancelled.
func (e *beatEventV0) cancel(call goja.FunctionCall) goja.Value {
	e.cancelled = true
	return goja.Undefined()
}

// tag adds a new value to the tags field if it is not already contained in the
// set.
//
//	// javascript
//	evt.Tag("_parse_failure");
//
func (e *beatEventV0) tag(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) != 1 {
		panic(errors.New("Tag requires one argument"))
	}

	tag := call.Argument(0).String()

	if err := appendString(e.inner.Fields, "tags", tag, true); err != nil {
		panic(err)
	}
	return goja.Undefined()
}

// appendTo is a specialized Put method that converts any existing value to
// an array and appends the value if it does not already exist. If there is an
// existing value that's not a string or array of strings then an exception is
// thrown.
//
//	// javascript
//	evt.AppendTo("error.message", "invalid file hash");
//
func (e *beatEventV0) appendTo(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) != 2 {
		panic(errors.New("AppendTo requires two arguments (field and value)"))
	}

	field := call.Argument(0).String()
	value := call.Argument(1).String()

	if err := appendString(e.inner.Fields, field, value, false); err != nil {
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
