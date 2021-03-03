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

package lnk

import (
	"encoding/binary"
	"fmt"
)

type shellbagParser func(data []byte) string

func simpleShellbagParser(name string) shellbagParser {
	return func(data []byte) string {
		return name
	}
}

func checkKnownGUIDs(offset int, data []byte) string {
	if len(data) >= 16+offset {
		uuid := encodeUUID(data[offset : 16+offset])
		if name, known := knownShellbagGuids[uuid]; known {
			return name
		}
	}
	return ""
}

func parseShellbag0x00(data []byte) string {
	return checkKnownGUIDs(0xE, data)
}

func parseShellbag0x01(data []byte) string {
	if data[8] == 0x3A && data[9] == 0x00 {
		return "Hyper-V storage volume"
	}
	signature := binary.LittleEndian.Uint32(data[4:])
	if signature != 0x39de2184 {
		return "Control Panel Category"
	}
	switch data[8] {
	case 0x00:
		return "All Control Panel Items"
	case 0x01:
		return "Appearance and Personalization"
	case 0x02:
		return "Hardware and Sound"
	case 0x03:
		return "Network and Internet"
	case 0x04:
		return "Sound, Speech and Audio Devices"
	case 0x05:
		return "System and Security"
	case 0x06:
		return "Clock, Language, and Region"
	case 0x07:
		return "Ease of Access"
	case 0x08:
		return "Programs"
	case 0x09:
		return "User Accounts"
	case 0x10:
		return "Security Center"
	case 0x11:
		return "Mobile PC"
	default:
		return fmt.Sprintf("Unknown Control Panel Category: %d", data[8])
	}
}

func parseShellbag0x2e(data []byte) string {
	if known := checkKnownGUIDs(0x4, data); known != "" {
		return known
	}

	if len(data) == 0x16 && data[3] == 0x80 {
		return "Root folder: GUID"
	}
	signature := binary.LittleEndian.Uint64(data[len(data)-8:])
	if signature == 0x0000ee306bfe9555 || signature == 0xee306bfe9555c589 {
		return "User profile"
	}
	shortSignature := binary.LittleEndian.Uint32(data[5:])
	if shortSignature >= 0x15032601 {
		return "Control panel category"
	}
	return "Users property view"
}

func parseShellbag0x1f(data []byte) string {
	if known := checkKnownGUIDs(4, data); known != "" {
		return known
	}
	if data[0] == 0x14 || data[0] == 0x32 || data[0] == 0x3A {
		return "Root folder: GUID"
	}
	if data[4] == 0x2f {
		return "Users property view: Drive letter"
	}
	maskedBit := data[3] & 0x70
	switch maskedBit {
	// https://github.com/williballenthin/shellbags/blob/fee76eb25c2b80c33caf8ab9013de5cba113dcd2/ShellItems.py#L54
	case 0x00:
		return "INTERNET_EXPLORER"
	case 0x42:
		return "LIBRARIES"
	case 0x44:
		return "USERS"
	case 0x48:
		return "MY_DOCUMENTS"
	case 0x50:
		return "MY_COMPUTER"
	case 0x58:
		return "NETWORK"
	case 0x60:
		return "RECYCLE_BIN"
	case 0x68:
		return "INTERNET_EXPLORER"
	case 0x80:
		return "MY_GAMES"
		// unknown
	case 0x40:
		fallthrough
	case 0x70:
		return "Root folder: GUID"
	}
	signature := binary.LittleEndian.Uint32(data[6:])
	if signature == 0xbeebee00 {
		return "Variable: Users property view"
	}
	if signature == 0x4c644970 {
		return "Windows Backup"
	}
	return "Users property view"
}

func parseShellbag0x40(data []byte) string {
	switch data[2] {
	case 0x47:
		return "Entire Network"
	case 0x46:
		return "Microsoft Windows Network"
	case 0x41:
		return "Domain/Workgroup name"
	case 0x42:
		return "Server UNC path"
	case 0x43:
		return "Share UNC path"
	default:
		return "Network location"
	}
}

func parseShellbag0x71(data []byte) string {
	return checkKnownGUIDs(0xE, data)
}

// Have a better look at
// https://github.com/williballenthin/shellbags/blob/fee76eb25c2b80c33caf8ab9013de5cba113dcd2/ShellItems.py
var knownShellbags = map[byte]shellbagParser{
	// https://github.com/EricZimmerman/Lnk/blob/master/Lnk/ShellItems/ShellBag0X23.cs
	0x23: simpleShellbagParser("Drive letter"),
	// https://github.com/EricZimmerman/Lnk/blob/master/Lnk/ShellItems/ShellBag0X4C.cs
	0x4C: simpleShellbagParser("Sharepoint directory"),
	// https://github.com/EricZimmerman/Lnk/blob/master/Lnk/ShellItems/ShellBag0x00.cs
	0x00: parseShellbag0x00,
	// https://github.com/EricZimmerman/Lnk/blob/master/Lnk/ShellItems/ShellBag0x01.cs
	0x01: parseShellbag0x01,
	// https://github.com/EricZimmerman/Lnk/blob/master/Lnk/ShellItems/ShellBag0x1f.cs
	0x1f: parseShellbag0x1f,
	// https://github.com/EricZimmerman/Lnk/blob/master/Lnk/ShellItems/ShellBag0x2e.cs
	0x2e: parseShellbag0x2e,
	// https://github.com/EricZimmerman/Lnk/blob/master/Lnk/ShellItems/ShellBag0x2f.cs
	0x2f: simpleShellbagParser("Drive letter"),
	// https://github.com/EricZimmerman/Lnk/blob/master/Lnk/ShellItems/ShellBag0x31.cs
	0x31: simpleShellbagParser("Directory"),
	// https://github.com/EricZimmerman/Lnk/blob/master/Lnk/ShellItems/ShellBag0x32.cs
	0x32: simpleShellbagParser("File"),
	// https://github.com/EricZimmerman/Lnk/blob/master/Lnk/ShellItems/ShellBag0x40.cs
	0x40: parseShellbag0x40,
	// https://github.com/EricZimmerman/Lnk/blob/master/Lnk/ShellItems/ShellBag0x61.cs
	0x61: simpleShellbagParser("URI"),
	// https://github.com/EricZimmerman/Lnk/blob/master/Lnk/ShellItems/ShellBag0x71.cs
	0x71: parseShellbag0x71,
	// https://github.com/EricZimmerman/Lnk/blob/master/Lnk/ShellItems/ShellBag0x74.cs
	// 0x74:
	// https://github.com/EricZimmerman/Lnk/blob/master/Lnk/ShellItems/ShellBag0xc3.cs
	0xc3: simpleShellbagParser("Network location"),
}

func getShellbagName(shellbagType byte, data []byte) string {
	if parser, known := knownShellbags[shellbagType]; known {
		return parser(data)
	}
	return ""
}
