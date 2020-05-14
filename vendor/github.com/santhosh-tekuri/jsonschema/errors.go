// Copyright 2017 Santhosh Kumar Tekuri. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package jsonschema

import (
	"fmt"
	"strings"
)

// InvalidJSONTypeError is the error type returned by ValidateInteface.
// this tells that specified go object is not valid jsonType.
type InvalidJSONTypeError string

func (e InvalidJSONTypeError) Error() string {
	return fmt.Sprintf("invalid jsonType: %s", string(e))
}

// SchemaError is the error type returned by Compile.
type SchemaError struct {
	// SchemaURL is the url to json-schema that filed to compile.
	// This is helpful, if your schema refers to external schemas
	SchemaURL string

	// Err is the error that occurred during compilation.
	// It could be ValidationError, because compilation validates
	// given schema against the json meta-schema
	Err error
}

func (se *SchemaError) Error() string {
	return fmt.Sprintf("json-schema %q compilation failed. Reason:\n%s", se.SchemaURL, se.Err)
}

// ValidationError is the error type returned by Validate.
type ValidationError struct {
	// Message describes error
	Message string

	// InstancePtr is json-pointer which refers to json-fragment in json instance
	// that is not valid
	InstancePtr string

	// SchemaURL is the url to json-schema against which validation failed.
	// This is helpful, if your schema refers to external schemas
	SchemaURL string

	// SchemaPtr is json-pointer which refers to json-fragment in json schema
	// that failed to satisfy
	SchemaPtr string

	// Causes details the nested validation errors
	Causes []*ValidationError
}

func (ve *ValidationError) add(causes ...error) error {
	for _, cause := range causes {
		addContext(ve.InstancePtr, ve.SchemaPtr, cause)
		ve.Causes = append(ve.Causes, cause.(*ValidationError))
	}
	return ve
}

func (ve *ValidationError) Error() string {
	msg := fmt.Sprintf("I[%s] S[%s] %s", ve.InstancePtr, ve.SchemaPtr, ve.Message)
	for _, c := range ve.Causes {
		for _, line := range strings.Split(c.Error(), "\n") {
			msg += "\n  " + line
		}
	}
	return msg
}

func validationError(schemaPtr string, format string, a ...interface{}) *ValidationError {
	return &ValidationError{fmt.Sprintf(format, a...), "", "", schemaPtr, nil}
}

func addContext(instancePtr, schemaPtr string, err error) error {
	ve := err.(*ValidationError)
	ve.InstancePtr = joinPtr(instancePtr, ve.InstancePtr)
	if len(ve.SchemaURL) == 0 {
		ve.SchemaPtr = joinPtr(schemaPtr, ve.SchemaPtr)
	}
	for _, cause := range ve.Causes {
		addContext(instancePtr, schemaPtr, cause)
	}
	return ve
}

func finishSchemaContext(err error, s *Schema) {
	ve := err.(*ValidationError)
	if len(ve.SchemaURL) == 0 {
		ve.SchemaURL = s.URL
		ve.SchemaPtr = s.Ptr + "/" + ve.SchemaPtr
		for _, cause := range ve.Causes {
			finishSchemaContext(cause, s)
		}
	}
}

func finishInstanceContext(err error) {
	ve := err.(*ValidationError)
	if len(ve.InstancePtr) == 0 {
		ve.InstancePtr = "#"
	} else {
		ve.InstancePtr = "#/" + ve.InstancePtr
	}
	for _, cause := range ve.Causes {
		finishInstanceContext(cause)
	}
}

func joinPtr(ptr1, ptr2 string) string {
	if len(ptr1) == 0 {
		return ptr2
	}
	if len(ptr2) == 0 {
		return ptr1
	}
	return ptr1 + "/" + ptr2
}
