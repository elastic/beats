// +build !go1.12

// Copyright 2019 Tamás Gulácsi
//
//
// SPDX-License-Identifier: UPL-1.0 OR Apache-2.0

package goracle

import "bytes"

func bytesReplaceAll(s, old, new []byte) []byte { return bytes.Replace(s, old, new, -1) }
