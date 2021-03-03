package lnk

import (
	"encoding/binary"
	"fmt"
)

type targetParser func(data []byte) string

func simpleTargetParser(name string) targetParser {
	return func(data []byte) string {
		return name
	}
}

func parseTarget0x01(data []byte) string {
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

func parseTarget0x2e(data []byte) string {
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

func parseTarget0x1f(data []byte) string {
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

func parseTarget0x40(data []byte) string {
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

var knownTargets = map[byte]targetParser{
	// https://github.com/EricZimmerman/Lnk/blob/master/Lnk/ShellItems/ShellBag0X23.cs
	0x23: simpleTargetParser("Drive letter"),
	// https://github.com/EricZimmerman/Lnk/blob/master/Lnk/ShellItems/ShellBag0X4C.cs
	0x4C: simpleTargetParser("Sharepoint directory"),
	// https://github.com/EricZimmerman/Lnk/blob/master/Lnk/ShellItems/ShellBag0x00.cs
	// 0x00:
	// https://github.com/EricZimmerman/Lnk/blob/master/Lnk/ShellItems/ShellBag0x01.cs
	0x01: parseTarget0x01,
	// https://github.com/EricZimmerman/Lnk/blob/master/Lnk/ShellItems/ShellBag0x1f.cs
	0x1f: parseTarget0x1f,
	// https://github.com/EricZimmerman/Lnk/blob/master/Lnk/ShellItems/ShellBag0x2e.cs
	0x2e: parseTarget0x2e,
	// https://github.com/EricZimmerman/Lnk/blob/master/Lnk/ShellItems/ShellBag0x2f.cs
	0x2f: simpleTargetParser("Drive letter"),
	// https://github.com/EricZimmerman/Lnk/blob/master/Lnk/ShellItems/ShellBag0x31.cs
	0x31: simpleTargetParser("Directory"),
	// https://github.com/EricZimmerman/Lnk/blob/master/Lnk/ShellItems/ShellBag0x32.cs
	0x32: simpleTargetParser("File"),
	// https://github.com/EricZimmerman/Lnk/blob/master/Lnk/ShellItems/ShellBag0x40.cs
	0x40: parseTarget0x40,
	// https://github.com/EricZimmerman/Lnk/blob/master/Lnk/ShellItems/ShellBag0x61.cs
	0x61: simpleTargetParser("URI"),
	// https://github.com/EricZimmerman/Lnk/blob/master/Lnk/ShellItems/ShellBag0x71.cs
	// 0x71:
	// https://github.com/EricZimmerman/Lnk/blob/master/Lnk/ShellItems/ShellBag0x74.cs
	// 0x74:
	// https://github.com/EricZimmerman/Lnk/blob/master/Lnk/ShellItems/ShellBag0xc3.cs
	0xc3: simpleTargetParser("Network location"),
}

func getTargetName(targetType byte, data []byte) string {
	if len(data) >= 20 {
		uuid := encodeUUID(data[4:20])
		if name, known := knownShellbagGuids[uuid]; known {
			return name
		}
	}
	if parser, known := knownTargets[targetType]; known {
		return parser(data)
	}
	return ""
}
