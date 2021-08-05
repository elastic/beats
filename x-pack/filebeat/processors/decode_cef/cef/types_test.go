// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cef

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToTimestamp(t *testing.T) {
	var times = []string{
		// Unix epoch in milliseconds.
		"1322004689000",

		// MMM dd HH:mm:ss.SSS zzz
		"Jun 23 17:37:24.000 Z",
		"Jun 23 17:37:24.000 EST",
		"Jun 23 17:37:24.000 +05",
		"Jun 23 17:37:24.000 +0500",
		"Jun 23 17:37:24.000 +05:00",
		"Jun 23 17:37:24.000 GMT+05:00",

		// MMM dd HH:mm:sss.SSS
		"Jun 23 17:37:24.000",

		// MMM dd HH:mm:ss zzz
		"Jun 23 17:37:24 Z",
		"Jun 23 17:37:24 EST",
		"Jun 23 17:37:24 +05",
		"Jun 23 17:37:24 +0500",
		"Jun 23 17:37:24 +05:00",
		"Jun 23 17:37:24 GMT+05:00",

		// MMM dd HH:mm:ss
		"Jun 23 17:37:24",

		// MMM dd yyyy HH:mm:ss.SSS zzz
		"Jun 23 2020 17:37:24.000 Z",
		"Jun 23 2020 17:37:24.000 EST",
		"Jun 23 2020 17:37:24.000 +05",
		"Jun 23 2020 17:37:24.000 +0500",
		"Jun 23 2020 17:37:24.000 +05:00",
		"Jun 23 2020 17:37:24.000 GMT+05:00",

		// MMM dd yyyy HH:mm:ss.SSS
		"Jun 23 2020 17:37:24.000",

		// MMM dd yyyy HH:mm:ss zzz
		"Jun 23 2020 17:37:24 Z",
		"Jun 23 2020 17:37:24 EST",
		"Jun 23 2020 17:37:24 +05",
		"Jun 23 2020 17:37:24 +0500",
		"Jun 23 2020 17:37:24 +05:00",
		"Jun 23 2020 17:37:24 GMT+05:00",

		// MMM dd yyyy HH:mm:ss
		"Jun 23 2020 17:37:24",
	}

	for _, timeValue := range times {
		_, err := toTimestamp(timeValue)
		assert.NoError(t, err, timeValue)
	}
}

func TestToMACAddress(t *testing.T) {
	var macs = []string{
		// EUI-48 (with and without separators).
		"00:0D:60:AF:1B:61",
		"00-0D-60-AF-1B-61",
		"000D.60AF.1B61",
		"000D60AF1B61",

		// EUI-64 (with and without separators).
		"00:0D:60:FF:FE:AF:1B:61",
		"00-0D-60-FF-FE-AF-1B-61",
		"000D.60FF.FEAF.1B61",
		"000D60FFEEAF1B61",
	}

	for _, mac := range macs {
		_, err := toMACAddress(mac)
		assert.NoError(t, err, mac)
	}
}
