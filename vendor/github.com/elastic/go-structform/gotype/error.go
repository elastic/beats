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

package gotype

import "errors"

var (
	errNotInitialized           = errors.New("Unfolder is not initialized")
	errInvalidState             = errors.New("invalid state")
	errUnsupported              = errors.New("unsupported")
	errMapRequiresStringKey     = errors.New("map requires string key")
	errSquashNeedObject         = errors.New("require map or struct when using squash/inline")
	errNilInput                 = errors.New("nil input")
	errRequiresPointer          = errors.New("requires pointer")
	errKeyIntoNonStruct         = errors.New("key for non-structure target")
	errUnexpectedObjectKey      = errors.New("unexpected object key")
	errRequiresPrimitive        = errors.New("requires primitive target to set a boolean value")
	errRequiresBoolReceiver     = errors.New("requires bool receiver")
	errIncompatibleTypes        = errors.New("can not assign to incompatible go type")
	errStartArrayWaitingForKey  = errors.New("start array while waiting for object field name")
	errStartObjectWaitingForKey = errors.New("start object while waiting for object field name")
	errExpectedArrayNotObject   = errors.New("expected array but received object")
	errExpectedObjectNotArray   = errors.New("expected object but received array")
	errUnexpectedArrayStart     = errors.New("unexpected array start")
	errUnexpectedObjectStart    = errors.New("unexpected object start")
	errExpectedObjectKey        = errors.New("waiting for object key or object end marker")
	errExpectedArray            = errors.New("expected array")
	errExpectedObject           = errors.New("expected object")
	errExpectedObjectValue      = errors.New("expected object value")
	errExpectedObjectClose      = errors.New("missing object close")
	errInlineAndOmitEmpty       = errors.New("inline and omitempty must not be set at the same time")

	errUnexpectedNil       = errors.New("unexpected nil value received")
	errUnexpectedBool      = errors.New("unexpected bool value received")
	errUnexpectedNum       = errors.New("unexpected numeric value received")
	errUnexpectedString    = errors.New("unexpected string value received")
	errUnexpectedArrayEnd  = errors.New("array closed early")
	errUnexpectedObjectEnd = errors.New("unexpected object close")
)

func errTODO() error {
	panic(errors.New("TODO"))
}

func visitErrTODO(V visitor, v interface{}) error {
	return errTODO()
}
