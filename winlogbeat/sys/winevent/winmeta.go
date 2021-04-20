// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package winevent

type WinMeta struct {
	Keywords map[int64]string  // Keyword bit mask to keyword name.
	Opcodes  map[uint8]string  // Opcode value to name.
	Levels   map[uint8]string  // Level value to name.
	Tasks    map[uint16]string // Task value to name.
}

// defaultWinMeta contains the static values that are a common across Windows. These
// values are from winmeta.xml inside the Windows SDK.
var defaultWinMeta = &WinMeta{
	Keywords: map[int64]string{
		0:                "AnyKeyword",
		0x1000000000000:  "Response Time",
		0x4000000000000:  "WDI Diag",
		0x8000000000000:  "SQM",
		0x10000000000000: "Audit Failure",
		0x20000000000000: "Audit Success",
		0x40000000000000: "Correlation Hint",
		0x80000000000000: "Classic",
	},
	Opcodes: map[uint8]string{
		0: "Info",
		1: "Start",
		2: "Stop",
		3: "DCStart",
		4: "DCStop",
		5: "Extension",
		6: "Reply",
		7: "Resume",
		8: "Suspend",
		9: "Send",
	},
	Levels: map[uint8]string{
		0: "Information", // "Log Always", but Event Viewer shows Information.
		1: "Critical",
		2: "Error",
		3: "Warning",
		4: "Information",
		5: "Verbose",
	},
	Tasks: map[uint16]string{
		0: "None",
	},
}
