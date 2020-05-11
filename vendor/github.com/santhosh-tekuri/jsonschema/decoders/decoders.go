// Copyright 2017 Santhosh Kumar Tekuri. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package decoders provides functions to decode encoded-string.
//
// It allows developers to register custom encodings, that can be used
// in json-schema for validation.
package decoders

import (
	"encoding/base64"
)

// The Decoder type is a function, that returns
// the bytes represented by encoded string.
type Decoder func(string) ([]byte, error)

var decoders = map[string]Decoder{
	"base64": base64.StdEncoding.DecodeString,
}

// Register registers Decoder object for given encoding.
func Register(name string, d Decoder) {
	decoders[name] = d
}

// Get returns Decoder object for given encoding, if found.
func Get(name string) (Decoder, bool) {
	d, ok := decoders[name]
	return d, ok
}
