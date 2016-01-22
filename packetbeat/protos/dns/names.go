// This file contains the name mapping data used to convert various DNS IDs to
// their string values.
package dns

import (
	"encoding/hex"
	"fmt"
	"strconv"

	mkdns "github.com/miekg/dns"
)

// dnsOpCodeToString converts a Opcode value to a string. If the type's
// string representation is unknown then the numeric value will be returned as
// a string.
func dnsOpCodeToString(opCode int) string {
	s, exists := mkdns.OpcodeToString[opCode]
	if !exists {
		return strconv.Itoa(int(opCode))
	}
	return s
}

// dnsResponseCodeToString converts a Rcode value to a string. If
// the type's string representation is unknown then "Unknown <rcode value>"
// will be returned.
func dnsResponseCodeToString(rcode int) string {
	s, exists := mkdns.RcodeToString[rcode]
	if !exists {
		return fmt.Sprintf("Unknown %d", int(rcode))
	}
	return s
}

// dnsTypeToString converts a RR type value to a string. If the type's
// string representation is unknown then the numeric value will be returned
// as a string.
func dnsTypeToString(t uint16) string {
	s, exists := mkdns.TypeToString[uint16(t)]
	if !exists {
		return strconv.Itoa(int(t))
	}
	return s
}

// dnsClassToString converts a RR class value to a string. If the class'es
// string representation is unknown then the numeric value will be returned
// as a string.
func dnsClassToString(c uint16) string {
	s, exists := mkdns.ClassToString[uint16(c)]
	if !exists {
		return strconv.Itoa(int(c))
	}
	return s
}

// hexStringToString converts an hexadeciaml string to string. Bytes
// below 32 or above 126 are represented as an escaped base10 integer (\DDD).
// Back slashes and quotes are escaped. Tabs, carriage returns, and line feeds
// will be converted to \t, \r and \n respectively.
// Example:
func hexStringToString(hexString string) (string, error) {
	bytes, err := hex.DecodeString(hexString)
	if err != nil {
		return hexString, err
	}

	var s []byte
	for _, value := range bytes {
		switch value {
		default:
			if value < 32 || value >= 127 {
				// Unprintable characters are written as \\DDD (e.g. \\012).
				s = append(s, []byte(fmt.Sprintf("\\%03d", int(value)))...)
			} else {
				s = append(s, value)
			}
		case '"', '\\':
			s = append(s, '\\', value)
		case '\t':
			s = append(s, '\\', 't')
		case '\r':
			s = append(s, '\\', 'r')
		case '\n':
			s = append(s, '\\', 'n')
		}
	}
	return string(s), nil
}
