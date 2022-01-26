// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cef

import (
	"strings"

	"github.com/pkg/errors"
	"go.uber.org/multierr"
)

// Parser is generated from a ragel state machine using the following command:
//go:generate ragel -Z -G1 cef.rl -o parser.go
//go:generate goimports -l -w parser.go
//
// Run go vet and remove any unreachable code in the generated parser.go.
// The go generator outputs duplicated goto statements sometimes.
//
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

func (e *Event) init(data string) {
	e.Version = -1
	e.DeviceVendor = ""
	e.DeviceProduct = ""
	e.DeviceVersion = ""
	e.DeviceEventClassID = ""
	e.Name = ""
	e.Severity = ""
	e.Extensions = nil

	// Estimate length of the extensions. But limit the allocation because
	// it's based on user input. This doesn't account for escaped equals.
	if n := strings.Count(data, "="); n > 0 {
		const maxLen = 50
		if n <= maxLen {
			e.Extensions = make(map[string]*Field, n)
		} else {
			e.Extensions = make(map[string]*Field, maxLen)
		}
	}
}

func (e *Event) pushExtension(key, value string) {
	if e.Extensions == nil {
		e.Extensions = map[string]*Field{}
	}
	field := &Field{String: value}
	e.Extensions[key] = field
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
func (e *Event) Unpack(data string, opts ...Option) error {
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
		mapping, found := extensionMappingLowerCase[strings.ToLower(key)]
		if !found {
			continue
		}

		// Mark the data type and do the actual conversion.
		field.Type = mapping.Type
		field.Interface, err = toType(field.String, mapping.Type, &settings)
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

type escapePosition struct {
	start, end int
}

// replaceEscapes replaces the escaped characters contained in v with their
// unescaped value.
func replaceEscapes(v string, startOffset int, escapes []escapePosition) string {
	if len(escapes) == 0 {
		return v
	}

	// Adjust escape offsets relative to the start offset of v.
	for i := 0; i < len(escapes); i++ {
		escapes[i].start = escapes[i].start - startOffset
		escapes[i].end = escapes[i].end - startOffset
	}

	var buf strings.Builder
	var prevEnd int

	// Iterate over escapes and replace them.
	for _, escape := range escapes {
		buf.WriteString(v[prevEnd:escape.start])

		value := v[escape.start:escape.end]

		switch value {
		case `\n`:
			buf.WriteByte('\n')
		case `\r`:
			buf.WriteByte('\r')
		default:
			// Remove leading slash.
			if len(value) > 0 && value[0] == '\\' {
				buf.WriteString(value[1:])
			}
		}

		prevEnd = escape.end
	}
	buf.WriteString(v[prevEnd:])

	return buf.String()
}
