package tls

import (
	"fmt"
	"strconv"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// ExtensionID is the 16-bit identifier for an extension
type ExtensionID uint16

// Extensions stores the data from parsed extensions
type Extensions struct {
	Parsed  common.MapStr
	Raw     map[ExtensionID][]byte
	InOrder []ExtensionID
}

type extensionParser func(reader bufferView) interface{}
type extension struct {
	label   string
	parser  extensionParser
	saveRaw bool
}

const (
	// ExtensionSupportedGroups identifies the supported group extension
	ExtensionSupportedGroups ExtensionID = 10
	// ExtensionEllipticCurvePointsFormats identifies the points formats extension
	ExtensionEllipticCurvePointsFormats = 11
)

var extensionMap = map[uint16]extension{
	0:      {"server_name_indication", parseSni, false},
	1:      {"max_fragment_length", parseMaxFragmentLen, false},
	2:      {"client_certificate_url", expectEmpty, false},
	3:      {"trusted_ca_keys", ignoreContent, false},
	4:      {"truncated_hmac", expectEmpty, false},
	5:      {"status_request", ignoreContent, false},
	6:      {"user_mapping", ignoreContent, false},
	7:      {"client_authz", ignoreContent, false},
	8:      {"server_authz", ignoreContent, false},
	9:      {"cert_type", parseCertType, false},
	10:     {"supported_groups", parseSupportedGroups, true},
	11:     {"ec_points_formats", parseEcPoints, true},
	12:     {"srp", parseSrp, false},
	13:     {"signature_algorithms", parseSignatureSchemes, false},
	16:     {"application_layer_protocol_negotiation", parseALPN, false},
	35:     {"session_ticket", parseTicket, false},
	0xff01: {"renegotiation_info", ignoreContent, false},
}

// ParseExtensions returns an Extensions object parsed from the supplied buffer
func ParseExtensions(buffer bufferView) Extensions {

	var extensionsLength uint16
	if !buffer.read16Net(0, &extensionsLength) || extensionsLength == 0 {
		// No extensions
		return Extensions{}
	}

	limit := 2 + int(extensionsLength)
	result := Extensions{
		Parsed: common.MapStr{},
		Raw:    make(map[ExtensionID][]byte),
	}

	var unknown []string
	for base := 2; base < limit; {
		var code, length uint16
		if !buffer.read16Net(base, &code) || !buffer.read16Net(base+2, &length) {
			logp.Warn("failed parsing extensions")
			return Extensions{}
		}

		extBuffer := buffer.subview(base+4, int(length))
		base += 4 + int(length)

		// Skip GREASE extensions
		if isGreaseValue(code) {
			continue
		}

		result.InOrder = append(result.InOrder, ExtensionID(code))
		label, parsed, saveRaw := parseExtension(code, extBuffer)
		if parsed != nil {
			result.Parsed[label] = parsed
		} else {
			unknown = append(unknown, label)
		}
		if saveRaw {
			result.Raw[ExtensionID(code)] = extBuffer.readBytes(0, extBuffer.length())
		}
	}
	if len(unknown) != 0 {
		result.Parsed["_unparsed_"] = unknown
	}
	return result
}

func parseExtension(code uint16, buffer bufferView) (string, interface{}, bool) {
	if ext, ok := extensionMap[code]; ok {
		parsed := ext.parser(buffer)
		return ext.label, parsed, ext.saveRaw
	}
	return strconv.Itoa(int(code)), nil, false
}

func parseSni(buffer bufferView) interface{} {
	var listLength uint16
	if !buffer.read16Net(0, &listLength) {
		return nil
	}
	var hosts []string
	for pos, limit := 2, 2+int(listLength); pos+3 <= limit; {
		var nameType uint8
		var nameLen uint16
		var host string
		if !buffer.read8(pos, &nameType) || !buffer.read16Net(pos+1, &nameLen) ||
			limit < pos+3+int(nameLen) || !buffer.readString(pos+3, int(nameLen), &host) {
			logp.Warn("SNI hostname list truncated")
			break
		}
		if nameType == 0 {
			hosts = append(hosts, host)
		}
		pos += 3 + int(nameLen)
	}
	return hosts
}

func parseMaxFragmentLen(buffer bufferView) interface{} {
	var val uint8
	if buffer.length() == 1 && buffer.read8(0, &val) {
		if val > 0 && val < 5 {
			return fmt.Sprintf("2^%d", 8+val)
		}
		return fmt.Sprintf("(unknown:%d)", val)
	}
	return nil
}

func ignoreContent(_ bufferView) interface{} {
	return nil
}

func expectEmpty(buffer bufferView) interface{} {
	if buffer.length() != 0 {
		return fmt.Sprintf("(expected empty: found %d bytes)", buffer.length())
	}
	return ""
}

func parseCertType(buffer bufferView) interface{} {
	var value uint8
	var types []string
	pos, limit := 0, buffer.length()
	if limit > 1 {
		buffer.read8(0, &value)
		pos = 1
		if int(value)+1 < limit {
			limit = 1 + int(value)
		}
	}
	for ; pos < limit && buffer.read8(pos, &value); pos++ {
		var label string
		switch value {
		case 0:
			label = "X.509"
		case 1:
			label = "OpenPGP"
		case 2:
			label = "RawPubKey"
		default:
			label = fmt.Sprintf("(unknown:%d)", value)
		}
		types = append(types, label)
	}
	return types
}

func parseSupportedGroups(buffer bufferView) interface{} {
	var value uint16
	if !buffer.read16Net(0, &value) || int(value)+2 != buffer.length() {
		return nil
	}
	var groups []string
	for pos := 0; buffer.read16Net(pos+2, &value); pos += 2 {
		if !isGreaseValue(value) {
			groups = append(groups, pointsGroup(value).String())
		}
	}
	return groups
}

func parseEcPoints(buffer bufferView) interface{} {
	var value, length uint8
	if !buffer.read8(0, &length) || int(length)+1 != buffer.length() {
		return nil
	}
	var formats []string
	for pos := 0; pos < int(length) && buffer.read8(1+pos, &value); pos++ {
		formats = append(formats, ecPointsFormat(value).String())
	}
	return formats
}

func parseSrp(buffer bufferView) interface{} {
	var length uint8
	if !buffer.read8(0, &length) || int(length)+1 > buffer.length() {
		return nil
	}
	var user string
	if !buffer.readString(1, int(length), &user) {
		return nil
	}
	return user
}

func parseSignatureSchemes(buffer bufferView) interface{} {
	var value uint16
	if !buffer.read16Net(0, &value) || int(value)+2 != buffer.length() {
		return nil
	}
	var groups []string
	for pos := 2; buffer.read16Net(pos, &value); pos += 2 {
		groups = append(groups, signatureScheme(value).String())
	}
	return groups
}

func parseTicket(buffer bufferView) interface{} {
	if buffer.length() > 0 {
		return fmt.Sprintf("(%d bytes)", buffer.length())
	}
	return ""
}

func parseALPN(buffer bufferView) interface{} {
	var length uint16
	if !buffer.read16Net(0, &length) || int(length)+2 != buffer.length() {
		return nil
	}
	var strlen uint8
	var proto string
	var protos []string
	for pos := 2; buffer.read8(pos, &strlen); {
		if !buffer.readString(pos+1, int(strlen), &proto) {
			return nil
		}
		protos = append(protos, proto)
		pos += 1 + int(strlen)
	}
	return protos
}
