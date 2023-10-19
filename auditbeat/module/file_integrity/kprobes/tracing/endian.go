// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux

package tracing

import (
	"encoding/binary"
)

// MachineEndian is either binary.BigEndian or binary.LittleEndian, depending
// on the current architecture.
var MachineEndian = binary.NativeEndian
