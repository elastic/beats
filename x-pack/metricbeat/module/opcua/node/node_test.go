// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package node

import (
	"math"
	"testing"
	"time"

	"github.com/gopcua/opcua/ua"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/pkg/mapstr"
)

func TestConfigValidate(t *testing.T) {
	valid := func() config {
		c := defaultConfig()
		c.Nodes = []nodeConfig{{ID: "ns=3;s=StepUp"}}
		return c
	}

	cases := map[string]struct {
		modify  func(*config)
		wantErr string
	}{
		"defaults":        {modify: func(*config) {}},
		"no nodes":        {modify: func(c *config) { c.Nodes = nil }, wantErr: "at least one node"},
		"invalid node id": {modify: func(c *config) { c.Nodes[0].ID = "ns=x;y" }, wantErr: "invalid node id"},
		"unknown policy":  {modify: func(c *config) { c.SecurityPolicy = "Rot13" }, wantErr: "unsupported security_policy"},
		"unknown mode":    {modify: func(c *config) { c.SecurityMode = "Maybe" }, wantErr: "unsupported security_mode"},
		"policy without mode": {
			modify:  func(c *config) { c.SecurityPolicy = "Basic256Sha256" },
			wantErr: "both be None or both be set",
		},
		"secure without certificate": {
			modify: func(c *config) {
				c.SecurityPolicy = "Basic256Sha256"
				c.SecurityMode = "SignAndEncrypt"
			},
			wantErr: "client_certificate and client_key are required",
		},
		"secure with certificate": {
			modify: func(c *config) {
				c.SecurityPolicy = "basic256sha256"
				c.SecurityMode = "signandencrypt"
				c.ClientCertificate = "cert.pem"
				c.ClientKey = "key.pem"
			},
		},
		"certificate without key": {
			modify:  func(c *config) { c.ClientCertificate = "cert.pem" },
			wantErr: "must be set together",
		},
		"password without username": {
			modify:  func(c *config) { c.Password = "secret" },
			wantErr: "username is empty",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			c := valid()
			tc.modify(&c)
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

func TestTypedValue(t *testing.T) {
	ts := time.Date(2026, 11, 24, 10, 0, 0, 0, time.UTC)
	cases := []struct {
		in        any
		wantKey   string
		wantValue any
		wantOK    bool
	}{
		{in: true, wantKey: "boolean", wantValue: true, wantOK: true},
		{in: int8(-5), wantKey: "long", wantValue: int64(-5), wantOK: true},
		{in: uint16(7), wantKey: "long", wantValue: int64(7), wantOK: true},
		{in: uint32(math.MaxUint32), wantKey: "long", wantValue: int64(math.MaxUint32), wantOK: true},
		{in: int64(-42), wantKey: "long", wantValue: int64(-42), wantOK: true},
		{in: uint64(42), wantKey: "long", wantValue: int64(42), wantOK: true},
		{in: uint64(math.MaxUint64), wantKey: "double", wantValue: float64(math.MaxUint64), wantOK: true},
		{in: float32(1.5), wantKey: "double", wantValue: 1.5, wantOK: true},
		{in: 2.25, wantKey: "double", wantValue: 2.25, wantOK: true},
		{in: math.NaN(), wantOK: false},
		{in: math.Inf(1), wantOK: false},
		{in: "running", wantKey: "keyword", wantValue: "running", wantOK: true},
		{in: ts, wantKey: "date", wantValue: ts, wantOK: true},
		{in: time.Time{}, wantOK: false},
		{in: &ua.LocalizedText{Text: "Pump 1"}, wantKey: "keyword", wantValue: "Pump 1", wantOK: true},
		{in: ua.NewStringNodeID(3, "StepUp"), wantKey: "keyword", wantValue: "ns=3;s=StepUp", wantOK: true},
		{in: []byte{1, 2}, wantOK: false},
	}

	for _, tc := range cases {
		key, value, ok := typedValue(tc.in)
		assert.Equal(t, tc.wantOK, ok, "%T %v", tc.in, tc.in)
		if tc.wantOK {
			assert.Equal(t, tc.wantKey, key, "%T %v", tc.in, tc.in)
			assert.Equal(t, tc.wantValue, value, "%T %v", tc.in, tc.in)
		}
	}
}

func TestSeverity(t *testing.T) {
	assert.Equal(t, "good", severity(ua.StatusOK))
	assert.Equal(t, "uncertain", severity(ua.StatusUncertain))
	assert.Equal(t, "bad", severity(ua.StatusBadNodeIDUnknown))
}

func TestEventFields(t *testing.T) {
	source := time.Date(2026, 11, 24, 10, 0, 0, 0, time.UTC)
	server := source.Add(time.Millisecond)
	n := node{id: ua.NewStringNodeID(3, "StepUp"), rawID: "ns=3;s=StepUp", label: "step", name: "StepUp"}

	t.Run("scalar value", func(t *testing.T) {
		dv := &ua.DataValue{
			Value:           ua.MustVariant(uint32(17)),
			Status:          ua.StatusOK,
			SourceTimestamp: source,
			ServerTimestamp: server,
		}
		assert.Equal(t, mapstr.M{
			"id":               "ns=3;s=StepUp",
			"label":            "step",
			"name":             "StepUp",
			"data_type":        "Uint32",
			"value":            mapstr.M{"long": int64(17)},
			"status":           mapstr.M{"code": int64(0), "name": "Good", "severity": "good"},
			"source_timestamp": source,
			"server_timestamp": server,
		}, eventFields(n, dv))
	})

	t.Run("bad status without value", func(t *testing.T) {
		dv := &ua.DataValue{Status: ua.StatusBadNodeIDUnknown}
		fields := eventFields(n, dv)
		assert.Equal(t, mapstr.M{
			"code":     int64(ua.StatusBadNodeIDUnknown),
			"name":     "StatusBadNodeIDUnknown",
			"severity": "bad",
		}, fields["status"])
		assert.NotContains(t, fields, "value")
		assert.NotContains(t, fields, "data_type")
	})

	t.Run("array is not indexed", func(t *testing.T) {
		dv := &ua.DataValue{Value: ua.MustVariant([]int32{1, 2, 3}), Status: ua.StatusOK}
		fields := eventFields(n, dv)
		assert.Equal(t, "Int32[]", fields["data_type"])
		assert.NotContains(t, fields, "value")
	})

	t.Run("byte string is a scalar", func(t *testing.T) {
		dv := &ua.DataValue{Value: ua.MustVariant([]byte{1, 2}), Status: ua.StatusOK}
		fields := eventFields(n, dv)
		assert.Equal(t, "ByteString", fields["data_type"])
		assert.NotContains(t, fields, "value")
	})
}
