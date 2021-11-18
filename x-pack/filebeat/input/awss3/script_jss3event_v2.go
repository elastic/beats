// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"strings"

	"github.com/dop251/goja"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common/encoding/xml"
)

func newJSS3EventV2Constructor(s *session) func(call goja.ConstructorCall) *goja.Object {
	return func(call goja.ConstructorCall) *goja.Object {
		if len(call.Arguments) != 0 {
			panic(errors.New("Event constructor don't accept arguments"))
		}
		return s.vm.ToValue(&s3EventV2{}).(*goja.Object)
	}
}

func (e *s3EventV2) SetAWSRegion(v string) {
	e.AWSRegion = v
}

func (e *s3EventV2) SetProvider(v string) {
	e.Provider = v
}

func (e *s3EventV2) SetEventName(v string) {
	e.EventName = v
}

func (e *s3EventV2) SetEventSource(v string) {
	e.EventSource = v
}

func (e *s3EventV2) SetS3BucketName(v string) {
	e.S3.Bucket.Name = v
}

func (e *s3EventV2) SetS3BucketARN(v string) {
	e.S3.Bucket.ARN = v
}

func (e *s3EventV2) SetS3ObjectKey(v string) {
	e.S3.Object.Key = v
}

func newXMLDecoderConstructor(s *session) func(call goja.ConstructorCall) *goja.Object {
	return func(call goja.ConstructorCall) *goja.Object {
		if len(call.Arguments) != 1 {
			panic(errors.New("Event constructor requires one argument"))
		}

		a0 := call.Argument(0).Export()
		s0, ok := a0.(string)

		if !ok {
			panic(errors.Errorf("Event constructor requires a "+
				"string argument but got %T", a0))
		}

		return s.vm.ToValue(xml.NewDecoder(strings.NewReader(s0))).(*goja.Object)
	}
}
