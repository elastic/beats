// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package tag

import (
	"math"

	"github.com/apache/plc4x/plc4go/pkg/api/model"
	"github.com/apache/plc4x/plc4go/pkg/api/values"

	"github.com/elastic/beats/v7/pkg/mapstr"
)

// eventFields builds the metricset fields for one tag of a read response.
func eventFields(t tagConfig, code model.PlcResponseCode, value values.PlcValue) mapstr.M {
	fields := mapstr.M{
		"address":       t.Address,
		"response_code": code.GetName(),
	}
	if t.Name != "" {
		fields["name"] = t.Name
	}
	if code != model.PlcResponseCode_OK || value == nil {
		return fields
	}

	valueType := value.GetPlcValueType()
	fields["data_type"] = valueType.String()
	if key, v, ok := typedValue(valueType, value); ok {
		fields["value"] = mapstr.M{key: v}
	}
	return fields
}

// typedValue maps a PLC4X value to one of the typed value fields, based on its
// IEC 61131-3 data type. PLC4X values answer several Is* checks at once, so the
// data type decides which getter is used.
func typedValue(valueType values.PlcValueType, v values.PlcValue) (string, any, bool) {
	switch valueType {
	case values.BOOL:
		return "boolean", v.GetBool(), true
	case values.BYTE, values.WORD, values.DWORD, values.USINT, values.UINT, values.UDINT:
		return "long", int64(v.GetUint32()), true
	case values.LWORD, values.ULINT:
		u := v.GetUint64()
		if u > math.MaxInt64 {
			return "double", float64(u), true
		}
		return "long", int64(u), true
	case values.SINT, values.INT, values.DINT, values.LINT:
		return "long", v.GetInt64(), true
	case values.REAL, values.LREAL:
		f := v.GetFloat64()
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return "", nil, false
		}
		return "double", f, true
	case values.CHAR, values.WCHAR, values.STRING, values.WSTRING:
		return "keyword", v.GetString(), true
	case values.DATE, values.LDATE, values.DATE_AND_TIME, values.DATE_AND_LTIME, values.LDATE_AND_TIME:
		ts := v.GetDateTime()
		if ts.IsZero() {
			return "", nil, false
		}
		return "date", ts, true
	default:
		// Lists, structs, raw byte arrays, durations and times of day are not
		// mapped. The data type is still reported.
		return "", nil, false
	}
}
