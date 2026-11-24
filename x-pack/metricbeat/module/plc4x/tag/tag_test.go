// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package tag

import (
	"math"
	"testing"
	"time"

	"github.com/apache/plc4x/plc4go/pkg/api/model"
	apivalues "github.com/apache/plc4x/plc4go/pkg/api/values"
	"github.com/apache/plc4x/plc4go/spi/values"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/pkg/mapstr"
)

func TestConfigValidate(t *testing.T) {
	cases := map[string]struct {
		tags    []tagConfig
		wantErr string
	}{
		"register":       {tags: []tagConfig{{Address: "holding-register:1:REAL"}}},
		"no tags":        {wantErr: "at least one tag"},
		"empty address":  {tags: []tagConfig{{Address: " "}}, wantErr: "address is empty"},
		"coil":           {tags: []tagConfig{{Address: "coil:1:BOOL"}}, wantErr: "coil addresses are not supported"},
		"discrete input": {tags: []tagConfig{{Address: "discrete-input:3:BOOL"}}, wantErr: "discrete-input addresses are not supported"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			c := config{Tags: tc.tags}
			err := c.Validate()
			if tc.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

func TestDriverOf(t *testing.T) {
	driver, err := driverOf("modbus-tcp://plc.example.com:502")
	require.NoError(t, err)
	assert.Equal(t, "modbus-tcp", driver)

	_, err = driverOf("s7://10.0.0.1")
	assert.ErrorContains(t, err, "unsupported PLC4X driver")

	_, err = driverOf("plc.example.com:502")
	assert.ErrorContains(t, err, "invalid PLC4X connection string")
}

func TestTypedValue(t *testing.T) {
	ts := time.Date(2026, 11, 24, 10, 0, 0, 0, time.UTC)
	cases := []struct {
		in        apivalues.PlcValue
		wantKey   string
		wantValue any
		wantOK    bool
	}{
		{in: values.NewPlcBOOL(true), wantKey: "boolean", wantValue: true, wantOK: true},
		{in: values.NewPlcINT(-42), wantKey: "long", wantValue: int64(-42), wantOK: true},
		{in: values.NewPlcUINT(1234), wantKey: "long", wantValue: int64(1234), wantOK: true},
		{in: values.NewPlcDINT(-70000), wantKey: "long", wantValue: int64(-70000), wantOK: true},
		{in: values.NewPlcLINT(math.MinInt64), wantKey: "long", wantValue: int64(math.MinInt64), wantOK: true},
		{in: values.NewPlcULINT(7), wantKey: "long", wantValue: int64(7), wantOK: true},
		{in: values.NewPlcULINT(math.MaxUint64), wantKey: "double", wantValue: float64(math.MaxUint64), wantOK: true},
		{in: values.NewPlcREAL(21.5), wantKey: "double", wantValue: 21.5, wantOK: true},
		{in: values.NewPlcLREAL(-0.25), wantKey: "double", wantValue: -0.25, wantOK: true},
		{in: values.NewPlcLREAL(math.NaN()), wantOK: false},
		{in: values.NewPlcSTRING("running"), wantKey: "keyword", wantValue: "running", wantOK: true},
		{in: values.NewPlcDATE_AND_TIME(ts), wantKey: "date", wantValue: ts, wantOK: true},
		{in: values.NewPlcList([]apivalues.PlcValue{values.NewPlcINT(1)}), wantOK: false},
	}
	for _, tc := range cases {
		key, value, ok := typedValue(tc.in.GetPlcValueType(), tc.in)
		assert.Equal(t, tc.wantOK, ok, "%s", tc.in.GetPlcValueType())
		if tc.wantOK {
			assert.Equal(t, tc.wantKey, key, "%s", tc.in.GetPlcValueType())
			assert.Equal(t, tc.wantValue, value, "%s", tc.in.GetPlcValueType())
		}
	}
}

func TestEventFields(t *testing.T) {
	tag := tagConfig{Address: "holding-register:1:REAL", Name: "temperature"}

	assert.Equal(t, mapstr.M{
		"address":       "holding-register:1:REAL",
		"name":          "temperature",
		"response_code": "OK",
		"data_type":     "REAL",
		"value":         mapstr.M{"double": 21.5},
	}, eventFields(tag, model.PlcResponseCode_OK, values.NewPlcREAL(21.5)))

	assert.Equal(t, mapstr.M{
		"address":       "holding-register:1:REAL",
		"name":          "temperature",
		"response_code": "INVALID_ADDRESS",
	}, eventFields(tag, model.PlcResponseCode_INVALID_ADDRESS, nil))
}
