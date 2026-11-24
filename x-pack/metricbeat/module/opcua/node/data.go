// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package node

import (
	"math"
	"reflect"
	"strings"
	"time"

	"github.com/gopcua/opcua/ua"

	"github.com/elastic/beats/v7/pkg/mapstr"
)

// node is a node from the configuration, resolved for reading.
type node struct {
	id    *ua.NodeID
	rawID string
	label string
	name  string // display name, read from the server
}

// eventFields builds the metricset fields for one read result.
func eventFields(n node, dv *ua.DataValue) mapstr.M {
	fields := mapstr.M{
		"id": n.rawID,
	}
	if n.label != "" {
		fields["label"] = n.label
	}
	if n.name != "" {
		fields["name"] = n.name
	}
	if dv == nil {
		return fields
	}

	fields["status"] = statusFields(dv.Status)
	if !dv.SourceTimestamp.IsZero() {
		fields["source_timestamp"] = dv.SourceTimestamp
	}
	if !dv.ServerTimestamp.IsZero() {
		fields["server_timestamp"] = dv.ServerTimestamp
	}

	if dv.Value == nil || dv.Value.Value() == nil {
		return fields
	}
	dataType := strings.TrimPrefix(dv.Value.Type().String(), "TypeID")
	if isArray(dv.Value) {
		// Arrays are not indexed: their elements can't be mapped to one typed
		// field, and they are rare for the scalar process values this module
		// targets. The data type is still reported.
		fields["data_type"] = dataType + "[]"
		return fields
	}
	fields["data_type"] = dataType

	if key, value, ok := typedValue(dv.Value.Value()); ok {
		fields["value"] = mapstr.M{key: value}
	}
	return fields
}

// typedValue maps a scalar OPC UA value to one of the typed value fields, so
// that each field keeps a single type across nodes.
func typedValue(v any) (string, any, bool) {
	switch x := v.(type) {
	case bool:
		return "boolean", x, true
	case int8:
		return "long", int64(x), true
	case int16:
		return "long", int64(x), true
	case int32:
		return "long", int64(x), true
	case int64:
		return "long", x, true
	case uint8:
		return "long", int64(x), true
	case uint16:
		return "long", int64(x), true
	case uint32:
		return "long", int64(x), true
	case uint64:
		if x > math.MaxInt64 {
			return "double", float64(x), true
		}
		return "long", int64(x), true
	case float32:
		f := float64(x)
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return "", nil, false
		}
		return "double", f, true
	case float64:
		if math.IsNaN(x) || math.IsInf(x, 0) {
			return "", nil, false
		}
		return "double", x, true
	case string:
		return "keyword", x, true
	case time.Time:
		if x.IsZero() {
			return "", nil, false
		}
		return "date", x, true
	case *ua.LocalizedText:
		if x == nil {
			return "", nil, false
		}
		return "keyword", x.Text, true
	case *ua.QualifiedName:
		if x == nil {
			return "", nil, false
		}
		return "keyword", x.Name, true
	case *ua.NodeID:
		if x == nil {
			return "", nil, false
		}
		return "keyword", x.String(), true
	case *ua.GUID:
		if x == nil {
			return "", nil, false
		}
		return "keyword", x.String(), true
	case ua.StatusCode:
		return "long", int64(x), true
	default:
		// ByteString, XmlElement, ExtensionObject and other structured values
		// are not mapped.
		return "", nil, false
	}
}

func isArray(v *ua.Variant) bool {
	if v.ArrayLength() > 0 || len(v.ArrayDimensions()) > 0 {
		return true
	}
	kind := reflect.TypeOf(v.Value()).Kind()
	if kind != reflect.Slice && kind != reflect.Array {
		return false
	}
	// A ByteString is a []byte scalar, not an array of values.
	return v.Type() != ua.TypeIDByteString
}

// statusFields describes an OPC UA status code. The two most significant bits
// of a status code give its severity.
func statusFields(code ua.StatusCode) mapstr.M {
	status := mapstr.M{
		"code":     int64(code),
		"severity": severity(code),
	}
	if code == ua.StatusOK {
		status["name"] = "Good"
	} else if desc, ok := ua.StatusCodes[code]; ok {
		status["name"] = desc.Name
	}
	return status
}

func severity(code ua.StatusCode) string {
	switch uint32(code) >> 30 {
	case 0:
		return "good"
	case 1:
		return "uncertain"
	default:
		return "bad"
	}
}
