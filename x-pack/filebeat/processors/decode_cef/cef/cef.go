// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cef

import (
	"bytes"

	"github.com/pkg/errors"
	"go.uber.org/multierr"
)

// Parser is generated from a ragel state machine using the following command:
//go:generate ragel -Z -G1 cef.rl -o parser.go
//go:generate imports -l -w parser.go

// An SVG rendering of the state machine can be viewed by opening cef.svg in
// Chrome / Firefox.
//go:generate ragel -V -p cef.rl -o cef.dot
//go:generate dot -T svg cef.dot -o cef.svg

// Field is CEF extension field value.
type Field struct {
	String    string      // Raw value.
	Type      DataType    // Data type from CEF guide.
	Interface interface{} // Converted value.
}

// Event is a single CEF message.
type Event struct {
	// CEF version.
	Version int `json:"version"`

	// Vendor of the sending device.
	DeviceVendor string `json:"device_vendor"`

	// Product of the sending device.
	DeviceProduct string `json:"device_product"`

	// Version of the sending device.
	DeviceVersion string `json:"device_version"`

	// Device Event Class ID identifies the type of event reported
	DeviceEventClassID string `json:"device_event_class_id"`

	// Human-readable and understandable description of the event.
	Name string `json:"name"`

	// Importance of the event. The valid string values are Unknown, Low,
	// Medium, High, and Very-High. The valid integer values are 0-3=Low,
	// 4-6=Medium, 7- 8=High, and 9-10=Very-High.
	Severity string `json:"severity"`

	// Extensions is a collection of key-value pairs. The keys are part of a
	// predefined set. The standard allows for including additional keys as
	// outlined in "ArcSight Extension Directory". An event can contain any
	// number of key-value pairs in any order.
	Extensions map[string]*Field `json:"extensions,omitempty"`
}

func (e *Event) init() {
	e.Version = -1
	e.DeviceVendor = ""
	e.DeviceProduct = ""
	e.DeviceVersion = ""
	e.DeviceEventClassID = ""
	e.Name = ""
	e.Severity = ""
	e.Extensions = nil
}

func (e *Event) pushExtension(key []byte, value []byte) {
	if e.Extensions == nil {
		e.Extensions = map[string]*Field{}
	}
	e.Extensions[string(key)] = &Field{String: string(value)}
}

// Unpack unpacks a common event format (CEF) message. The data is expected to
// be UTF-8 encoded and must begin with the CEF message header (e.g. starts
// with "CEF:").
//
// The CEF message consists of a header followed by a series of key-value pairs.
//
//    CEF:Version|Device Vendor|Device Product|Device Version|Device Event Class ID|Name|Severity|[Extension]
//
// The header is a series of pipe delimited values. If a pipe (|) is used in a
// header value, it has to be escaped with a backslash (\). If a backslash is
// used is must be escaped with another backslash.
//
// The extension contains key-value pairs. The equals sign (=) separates each
// key from value. And key-value pairs are separated by a single space
// (e.g. "src=1.2.3.4 dst=8.8.8.8"). If an equals sign is used as part of the
// value then it must be escaped with a backslash (\). If a backslash is used is
// must be escaped with another backslash.
//
// Extension keys must begin with an alphanumeric or underscore (_) character
// and may contain alphanumeric, underscore (_), period (.), comma (,), and
// brackets ([) (]). This is less strict than the CEF specification, but aligns
// the key names used in practice.
func (e *Event) Unpack(data []byte, opts ...Option) error {
	var settings Settings
	for _, opt := range opts {
		opt.Apply(&settings)
	}

	var errs []error
	var err error
	if err = e.unpack(data); err != nil {
		errs = append(errs, err)
	}

	for key, field := range e.Extensions {
		mapping, found := extensionMapping[key]
		if !found {
			continue
		}

		// Mark the data type and do the actual conversion.
		field.Type = mapping.Type
		field.Interface, err = ToType(field.String, mapping.Type)
		if err != nil {
			// Drop the key because the field value is invalid.
			delete(e.Extensions, key)
			errs = append(errs, errors.Wrapf(err, "error in field '%v'", key))
			continue
		}

		// Rename extension.
		if settings.fullExtensionNames && key != mapping.Target {
			e.Extensions[mapping.Target] = field
			delete(e.Extensions, key)
		}
	}

	return multierr.Combine(errs...)
}

var (
	backslash        = []byte(`\`)
	escapedBackslash = []byte(`\\`)

	pipe        = []byte(`|`)
	escapedPipe = []byte(`\|`)

	equalsSign        = []byte(`=`)
	escapedEqualsSign = []byte(`\=`)
)

func replaceHeaderEscapes(b []byte) []byte {
	if bytes.IndexByte(b, '\\') != -1 {
		b = bytes.ReplaceAll(b, escapedBackslash, backslash)
		b = bytes.ReplaceAll(b, escapedPipe, pipe)
	}
	return b
}

func replaceExtensionEscapes(b []byte) []byte {
	if bytes.IndexByte(b, '\\') != -1 {
		b = bytes.ReplaceAll(b, escapedBackslash, backslash)
		b = bytes.ReplaceAll(b, escapedEqualsSign, equalsSign)
	}
	return b
}
