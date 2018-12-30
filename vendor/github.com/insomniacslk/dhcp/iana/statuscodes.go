package iana

// StatusCode represents a IANA status code for DHCPv6
type StatusCode uint16

// IANA status codes as defined by rfc 3315 par. 24..4
const (
	StatusSuccess      StatusCode = 0
	StatusUnspecFail   StatusCode = 1
	StatusNoAddrsAvail StatusCode = 2
	StatusNoBinding    StatusCode = 3
	StatusNotOnLink    StatusCode = 4
	StatusUseMulticast StatusCode = 5
)

// StatusCodeToString returns a mnemonic name for a given status code
func StatusCodeToString(s StatusCode) string {
	if sc := StatusCodeToStringMap[s]; sc != "" {
		return sc
	}
	return "Unknown"
}

// StatusCodeToStringMap maps status codes to their names
var StatusCodeToStringMap = map[StatusCode]string{
	StatusSuccess:      "Success",
	StatusUnspecFail:   "UnspecFail",
	StatusNoAddrsAvail: "NoAddrsAvail",
	StatusNoBinding:    "NoBinding",
	StatusNotOnLink:    "NotOnLink",
	StatusUseMulticast: "UseMulticast",
}
