// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package decode_cef

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

type testCase struct {
	config  func() config
	message string
	fields  common.MapStr
}

var testCases = map[string]testCase{
	"default": {
		message: "CEF:0|Trend Micro|Deep Security Manager|1.2.3|600|User Signed In|3|src=10.52.116.160 suser=admin target=admin msg=User signed in from 2001:db8::5",
		fields: common.MapStr{
			"cef.version":                   0,
			"cef.device.event_class_id":     "600",
			"cef.device.product":            "Deep Security Manager",
			"cef.device.vendor":             "Trend Micro",
			"cef.device.version":            "1.2.3",
			"cef.name":                      "User Signed In",
			"cef.severity":                  "3",
			"cef.extensions.message":        "User signed in from 2001:db8::5",
			"cef.extensions.sourceAddress":  "10.52.116.160",
			"cef.extensions.sourceUserName": "admin",
			"cef.extensions.target":         "admin",
			"message":                       "User signed in from 2001:db8::5",
			"source.ip":                     "10.52.116.160",
			"source.user.name":              "admin",
		},
	},
	"custom_target": {
		config: func() config {
			c := defaultConfig()
			c.Target = ""
			return c
		},
		message: "CEF:0|Trend Micro|Deep Security Manager|1.2.3|600|User Signed In|3|src=10.52.116.160 suser=admin target=admin msg=User signed in from 2001:db8::5",
		fields: common.MapStr{
			"version":                   0,
			"device.event_class_id":     "600",
			"device.product":            "Deep Security Manager",
			"device.vendor":             "Trend Micro",
			"device.version":            "1.2.3",
			"name":                      "User Signed In",
			"severity":                  "3",
			"extensions.message":        "User signed in from 2001:db8::5",
			"extensions.sourceAddress":  "10.52.116.160",
			"extensions.sourceUserName": "admin",
			"extensions.target":         "admin",
			"message":                   "User signed in from 2001:db8::5",
			"source.ip":                 "10.52.116.160",
			"source.user.name":          "admin",
		},
	},
	"parse_errors": {
		message: "CEF:0|Trend Micro|Deep Security Manager|1.2.3|600|User Signed In|3|msg=User signed in with =xyz",
		fields: common.MapStr{
			"cef.version":               0,
			"cef.device.event_class_id": "600",
			"cef.device.product":        "Deep Security Manager",
			"cef.device.vendor":         "Trend Micro",
			"cef.device.version":        "1.2.3",
			"cef.name":                  "User Signed In",
			"cef.severity":              "3",
			"message":                   "CEF:0|Trend Micro|Deep Security Manager|1.2.3|600|User Signed In|3|msg=User signed in with =xyz",
			"error.message": []string{
				"malformed value for msg at pos 92",
				"unexpected end of CEF event",
			},
		},
	},
	"ecs_disabled": {
		config: func() config {
			c := defaultConfig()
			c.ECS = false
			return c
		},
		message: "CEF:0|Trend Micro|Deep Security Manager|1.2.3|600|User Signed In|3|src=10.52.116.160 suser=admin target=admin msg=User signed in from 2001:db8::5",
		fields: common.MapStr{
			"cef.version":                   0,
			"cef.device.event_class_id":     "600",
			"cef.device.product":            "Deep Security Manager",
			"cef.device.vendor":             "Trend Micro",
			"cef.device.version":            "1.2.3",
			"cef.name":                      "User Signed In",
			"cef.severity":                  "3",
			"cef.extensions.message":        "User signed in from 2001:db8::5",
			"cef.extensions.sourceAddress":  "10.52.116.160",
			"cef.extensions.sourceUserName": "admin",
			"cef.extensions.target":         "admin",
			"message":                       "CEF:0|Trend Micro|Deep Security Manager|1.2.3|600|User Signed In|3|src=10.52.116.160 suser=admin target=admin msg=User signed in from 2001:db8::5",
		},
	},
}

func TestProcessorRun(t *testing.T) {
	dec, err := newDecodeCEF(defaultConfig())
	if err != nil {
		t.Fatal(err)
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			dec := dec
			if tc.config != nil {
				dec, err = newDecodeCEF(tc.config())
				if err != nil {
					t.Fatal(err)
				}
			}

			evt := &beat.Event{
				Fields: common.MapStr{
					"message": tc.message,
				},
			}

			evt, err = dec.Run(evt)
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, tc.fields, evt.Fields.Flatten())
		})
	}

	t.Run("not_cef", func(t *testing.T) {
		evt := &beat.Event{
			Fields: common.MapStr{
				"message": "hello world!",
			},
		}

		evt, err = dec.Run(evt)
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "header start not found")
		}
	})

	t.Run("leading_garbage", func(t *testing.T) {
		tc := testCases["default"]

		evt := &beat.Event{
			Fields: common.MapStr{
				"message": "leading garbage" + tc.message,
			},
		}

		evt, err = dec.Run(evt)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, tc.fields, evt.Fields.Flatten())
	})
}
