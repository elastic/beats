// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cef

import (
	"crypto/sha1"
	"encoding/hex"
	"flag"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

var generateCorpus = flag.Bool("corpus", false, "generate fuzz corpus from test cases")

const (
	standardMessage = `CEF:26|security|threatmanager|1.0|100|trojan successfully stopped|10|src=10.0.0.192 dst=12.121.122.82 spt=1232 eventId=1`

	headerOnly = `CEF:26|security|threatmanager|1.0|100|trojan successfully stopped|10|`

	emptyDeviceFields = `CEF:0|||1.0|100|trojan successfully stopped|10|src=10.0.0.192 dst=12.121.122.82 spt=1232`

	escapedPipeInHeader = `CEF:26|security|threat\|->manager|1.0|100|trojan successfully stopped|10|src=10.0.0.192 dst=12.121.122.82 spt=1232`

	equalsSignInHeader = `CEF:26|security|threat=manager|1.0|100|trojan successfully stopped|10|src=10.0.0.192 dst=12.121.122.82 spt=1232`

	emptyExtensionValue = `CEF:26|security|threatmanager|1.0|100|trojan successfully stopped|10|src=10.0.0.192 dst= spt=1232`

	leadingWhitespace = `CEF:0|security|threatmanager|1.0|100|trojan successfully stopped|10| src=10.0.0.192 dst=12.121.122.82 spt=1232`

	escapedPipeInExtension = `CEF:0|security|threatmanager|1.0|100|trojan successfully stopped|10|moo=this\|has an escaped pipe`

	pipeInMessage = `CEF:0|security|threatmanager|1.0|100|trojan successfully stopped|10|moo=this|has an pipe`

	equalsInMessage = `CEF:0|security|threatmanager|1.0|100|trojan successfully stopped|10|moo=this =has = equals\=`

	escapesInExtension = `CEF:0|security|threatmanager|1.0|100|trojan successfully stopped|10|msg=a+b\=c x=c\\d\=z`

	malformedExtensionEscape = `CEF:0|FooBar|Web Gateway|1.2.3.45.67|200|Success|2|rt=Sep 07 2018 14:50:39 cat=Access Log dst=1.1.1.1 dhost=foo.example.com suser=redacted src=2.2.2.2 requestMethod=POST request='https://foo.example.com/bar/bingo/1' requestClientApplication='Foo-Bar/2018.1.7; =Email:user@example.com; Guid:test=' cs1= cs1Label=Foo Bar`

	multipleMalformedExtensionValues = `CEF:0|vendor|product|version|event_id|name|Very-High| msg=Hello World error=Failed because id==old_id user=root angle=106.7<=180`
)

var testMessages = []string{
	standardMessage,
	headerOnly,
	emptyDeviceFields,
	escapedPipeInHeader,
	equalsSignInHeader,
	emptyExtensionValue,
	leadingWhitespace,
	escapedPipeInExtension,
	pipeInMessage,
	equalsInMessage,
	escapesInExtension,
	malformedExtensionEscape,
	multipleMalformedExtensionValues,
}

func TestGenerateFuzzCorpus(t *testing.T) {
	if !*generateCorpus {
		t.Skip("-corpus is not enabled")
	}

	for _, m := range testMessages {
		h := sha1.New()
		h.Write([]byte(m))
		name := hex.EncodeToString(h.Sum(nil))

		ioutil.WriteFile(filepath.Join("fuzz/corpus", name), []byte(m), 0644)
	}
}

func TestEventUnpack(t *testing.T) {
	t.Run("standardMessage", func(t *testing.T) {
		var e Event
		err := e.Unpack([]byte(standardMessage))
		assert.NoError(t, err)
		assert.Equal(t, 26, e.Version)
		assert.Equal(t, "security", e.DeviceVendor)
		assert.Equal(t, "threatmanager", e.DeviceProduct)
		assert.Equal(t, "1.0", e.DeviceVersion)
		assert.Equal(t, "100", e.DeviceEventClassID)
		assert.Equal(t, "trojan successfully stopped", e.Name)
		assert.Equal(t, "10", e.Severity)
		assert.Equal(t, map[string]string{
			"src":     "10.0.0.192",
			"dst":     "12.121.122.82",
			"spt":     "1232",
			"eventId": "1",
		}, e.Extensions)
	})

	t.Run("headerOnly", func(t *testing.T) {
		var e Event
		err := e.Unpack([]byte(headerOnly))
		assert.NoError(t, err)
		assert.Equal(t, 26, e.Version)
		assert.Equal(t, "security", e.DeviceVendor)
		assert.Equal(t, "threatmanager", e.DeviceProduct)
		assert.Equal(t, "1.0", e.DeviceVersion)
		assert.Equal(t, "100", e.DeviceEventClassID)
		assert.Equal(t, "trojan successfully stopped", e.Name)
		assert.Equal(t, "10", e.Severity)
		assert.Nil(t, e.Extensions)
	})

	t.Run("escapedPipeInHeader", func(t *testing.T) {
		var e Event
		err := e.Unpack([]byte(escapedPipeInHeader))
		assert.NoError(t, err)
		assert.Equal(t, 26, e.Version)
		assert.Equal(t, "security", e.DeviceVendor)
		assert.Equal(t, "threat|->manager", e.DeviceProduct)
		assert.Equal(t, "1.0", e.DeviceVersion)
		assert.Equal(t, "100", e.DeviceEventClassID)
		assert.Equal(t, "trojan successfully stopped", e.Name)
		assert.Equal(t, "10", e.Severity)
		assert.Equal(t, map[string]string{
			"src": "10.0.0.192",
			"dst": "12.121.122.82",
			"spt": "1232",
		}, e.Extensions)
	})

	t.Run("equalsSignInHeader", func(t *testing.T) {
		var e Event
		err := e.Unpack([]byte(equalsSignInHeader))
		assert.NoError(t, err)
		assert.Equal(t, 26, e.Version)
		assert.Equal(t, "security", e.DeviceVendor)
		assert.Equal(t, "threat=manager", e.DeviceProduct)
		assert.Equal(t, "1.0", e.DeviceVersion)
		assert.Equal(t, "100", e.DeviceEventClassID)
		assert.Equal(t, "trojan successfully stopped", e.Name)
		assert.Equal(t, "10", e.Severity)
		assert.Equal(t, map[string]string{
			"src": "10.0.0.192",
			"dst": "12.121.122.82",
			"spt": "1232",
		}, e.Extensions)
	})

	t.Run("emptyExtensionValue", func(t *testing.T) {
		var e Event
		err := e.Unpack([]byte(emptyExtensionValue))
		assert.NoError(t, err)
		assert.Equal(t, 26, e.Version)
		assert.Equal(t, "security", e.DeviceVendor)
		assert.Equal(t, "threatmanager", e.DeviceProduct)
		assert.Equal(t, "1.0", e.DeviceVersion)
		assert.Equal(t, "100", e.DeviceEventClassID)
		assert.Equal(t, "trojan successfully stopped", e.Name)
		assert.Equal(t, "10", e.Severity)
		assert.Equal(t, map[string]string{
			"src": "10.0.0.192",
			"dst": "",
			"spt": "1232",
		}, e.Extensions)
	})

	t.Run("emptyDeviceFields", func(t *testing.T) {
		var e Event
		err := e.Unpack([]byte(emptyDeviceFields))
		assert.NoError(t, err)
		assert.Equal(t, 0, e.Version)
		assert.Equal(t, "", e.DeviceVendor)
		assert.Equal(t, "", e.DeviceProduct)
		assert.Equal(t, "1.0", e.DeviceVersion)
		assert.Equal(t, "100", e.DeviceEventClassID)
		assert.Equal(t, "trojan successfully stopped", e.Name)
		assert.Equal(t, "10", e.Severity)
		assert.Equal(t, map[string]string{
			"src": "10.0.0.192",
			"dst": "12.121.122.82",
			"spt": "1232",
		}, e.Extensions)
	})

	t.Run("errorEscapedPipeInExtension", func(t *testing.T) {
		var e Event
		err := e.Unpack([]byte(escapedPipeInExtension))
		assert.Equal(t, 0, e.Version)
		assert.Equal(t, "security", e.DeviceVendor)
		assert.Equal(t, "threatmanager", e.DeviceProduct)
		assert.Equal(t, "1.0", e.DeviceVersion)
		assert.Equal(t, "100", e.DeviceEventClassID)
		assert.Equal(t, "trojan successfully stopped", e.Name)
		assert.Equal(t, "10", e.Severity)
		assert.Nil(t, e.Extensions)

		// Pipes in extensions should not be escaped.
		assert.Error(t, err)
	})

	t.Run("leadingWhitespace", func(t *testing.T) {
		var e Event
		err := e.Unpack([]byte(leadingWhitespace))
		assert.NoError(t, err)
		assert.Equal(t, 0, e.Version)
		assert.Equal(t, "security", e.DeviceVendor)
		assert.Equal(t, "threatmanager", e.DeviceProduct)
		assert.Equal(t, "1.0", e.DeviceVersion)
		assert.Equal(t, "100", e.DeviceEventClassID)
		assert.Equal(t, "trojan successfully stopped", e.Name)
		assert.Equal(t, "10", e.Severity)
		assert.Equal(t, map[string]string{
			"src": "10.0.0.192",
			"dst": "12.121.122.82",
			"spt": "1232",
		}, e.Extensions)
	})

	t.Run("pipeInMessage", func(t *testing.T) {
		var e Event
		err := e.Unpack([]byte(pipeInMessage))
		assert.NoError(t, err)
		assert.Equal(t, 0, e.Version)
		assert.Equal(t, "security", e.DeviceVendor)
		assert.Equal(t, "threatmanager", e.DeviceProduct)
		assert.Equal(t, "1.0", e.DeviceVersion)
		assert.Equal(t, "100", e.DeviceEventClassID)
		assert.Equal(t, "trojan successfully stopped", e.Name)
		assert.Equal(t, "10", e.Severity)
		assert.Equal(t, map[string]string{
			"moo": "this|has an pipe",
		}, e.Extensions)
	})

	t.Run("errorEqualsInMessage", func(t *testing.T) {
		var e Event
		err := e.Unpack([]byte(equalsInMessage))
		assert.Equal(t, 0, e.Version)
		assert.Equal(t, "security", e.DeviceVendor)
		assert.Equal(t, "threatmanager", e.DeviceProduct)
		assert.Equal(t, "1.0", e.DeviceVersion)
		assert.Equal(t, "100", e.DeviceEventClassID)
		assert.Equal(t, "trojan successfully stopped", e.Name)
		assert.Equal(t, "10", e.Severity)
		assert.Nil(t, e.Extensions)

		// moo contains unescaped equals signs.
		assert.Error(t, err)
	})

	t.Run("escapesInExtension", func(t *testing.T) {
		var e Event
		err := e.Unpack([]byte(escapesInExtension))
		assert.NoError(t, err)
		assert.Equal(t, 0, e.Version)
		assert.Equal(t, "security", e.DeviceVendor)
		assert.Equal(t, "threatmanager", e.DeviceProduct)
		assert.Equal(t, "1.0", e.DeviceVersion)
		assert.Equal(t, "100", e.DeviceEventClassID)
		assert.Equal(t, "trojan successfully stopped", e.Name)
		assert.Equal(t, "10", e.Severity)
		assert.Equal(t, map[string]string{
			"msg": "a+b=c",
			"x":   `c\d=z`,
		}, e.Extensions)
	})

	t.Run("errorMalformedExtensionEscape", func(t *testing.T) {
		var e Event
		err := e.Unpack([]byte(malformedExtensionEscape))
		assert.Equal(t, 0, e.Version)
		assert.Equal(t, "FooBar", e.DeviceVendor)
		assert.Equal(t, "Web Gateway", e.DeviceProduct)
		assert.Equal(t, "1.2.3.45.67", e.DeviceVersion)
		assert.Equal(t, "200", e.DeviceEventClassID)
		assert.Equal(t, "Success", e.Name)
		assert.Equal(t, "2", e.Severity)
		assert.Equal(t, map[string]string{
			"rt":            "Sep 07 2018 14:50:39",
			"cat":           "Access Log",
			"dst":           "1.1.1.1",
			"dhost":         "foo.example.com",
			"suser":         "redacted",
			"src":           "2.2.2.2",
			"requestMethod": "POST",
			"request":       `'https://foo.example.com/bar/bingo/1'`,
			"cs1":           "",
			"cs1Label":      "Foo Bar",
		}, e.Extensions)

		// requestClientApplication is not valid because it contains an unescaped
		// equals sign.
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "requestClientApplication")
		}
	})

	t.Run("errorMultipleMalformedExtensionValues", func(t *testing.T) {
		var e Event
		err := e.Unpack([]byte(multipleMalformedExtensionValues))
		assert.Equal(t, 0, e.Version)
		assert.Equal(t, "vendor", e.DeviceVendor)
		assert.Equal(t, "product", e.DeviceProduct)
		assert.Equal(t, "version", e.DeviceVersion)
		assert.Equal(t, "event_id", e.DeviceEventClassID)
		assert.Equal(t, "name", e.Name)
		assert.Equal(t, "Very-High", e.Severity)
		assert.Equal(t, map[string]string{
			"msg":   "Hello World",
			"error": "Failed because",
			"user":  "root",
		}, e.Extensions)

		// Both id and angle contain unescaped equals signs.
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "id")
			assert.Contains(t, err.Error(), "malformed")
		}
	})

	t.Run("empty", func(t *testing.T) {
		var e Event
		err := e.Unpack([]byte("CEF:0|||||||a="))
		assert.NoError(t, err)
	})
}

func TestEventUnpackWithFullExtensionNames(t *testing.T) {
	var e Event
	err := e.Unpack([]byte(standardMessage), WithFullExtensionNames())
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{
		"sourceAddress":      "10.0.0.192",
		"destinationAddress": "12.121.122.82",
		"sourcePort":         "1232",
		"eventId":            "1",
	}, e.Extensions)
}

func BenchmarkEventUnpack(b *testing.B) {
	var messages [][]byte
	for _, m := range testMessages {
		messages = append(messages, []byte(m))
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var e Event
		e.Unpack(messages[i%len(messages)])
	}
}
