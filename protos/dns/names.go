// This file contains the name mapping data used to convert various DNS IDs to
// their string values.
package dns

import (
	"fmt"
	"strconv"

	"github.com/tsg/gopacket/layers"
)

// opCodeToStringMap contains a mapping of DNS op code values to strings.
var opCodeToStringMap = map[uint8]string{
	0: "QUERY",
	1: "IQUERY",
	2: "STATUS",
	4: "NOTIFY",
	5: "UPDATE",
}

// classToStringMap contains a mapping of DNS class values to strings.
var classToStringMap = map[uint16]string{
	1:   "IN",
	2:   "CS",
	3:   "CH",
	4:   "HS",
	254: "NONE",
	255: "ANY",
}

// typeToStringMap contains a mapping of DNS type values to strings.
// This list was created from data maintained by IANA at
// https://www.iana.org/assignments/dns-parameters/dns-parameters.xhtml.
var typeToStringMap = map[uint16]string{
	1:     "A",          // a host address
	2:     "NS",         // an authoritative name server
	3:     "MD",         // a mail destination (OBSOLETE - use MX)
	4:     "MF",         // a mail forwarder (OBSOLETE - use MX)
	5:     "CNAME",      // the canonical name for an alias
	6:     "SOA",        // marks the start of a zone of authority
	7:     "MB",         // a mailbox domain name (EXPERIMENTAL)
	8:     "MG",         // a mail group member (EXPERIMENTAL)
	9:     "MR",         // a mail rename domain name (EXPERIMENTAL)
	10:    "NULL",       // a null RR (EXPERIMENTAL)
	11:    "WKS",        // a well known service description
	12:    "PTR",        // a domain name pointer
	13:    "HINFO",      // host information
	14:    "MINFO",      // mailbox or mail list information
	15:    "MX",         // mail exchange
	16:    "TXT",        // text strings
	17:    "RP",         // for Responsible Person
	18:    "AFSDB",      // for AFS Data Base location
	19:    "X25",        // for X.25 PSDN address
	20:    "ISDN",       // for ISDN address
	21:    "RT",         // for Route Through
	22:    "NSAP",       // for NSAP address, NSAP style A record
	23:    "NSAP-PTR",   // for domain name pointer, NSAP style
	24:    "SIG",        // for security signature
	25:    "KEY",        // for security key
	26:    "PX",         // X.400 mail mapping information
	27:    "GPOS",       // Geographical Position
	28:    "AAAA",       // IP6 Address
	29:    "LOC",        // Location Information
	30:    "NXT",        // Next Domain (OBSOLETE)
	31:    "EID",        // Endpoint Identifier
	32:    "NIMLOC",     // Nimrod Locator
	33:    "SRV",        // Server Selection
	34:    "ATMA",       // ATM Address
	35:    "NAPTR",      // Naming Authority Pointer
	36:    "KX",         // Key Exchanger
	37:    "CERT",       // CERT
	38:    "A6",         // A6 (OBSOLETE - use AAAA)
	39:    "DNAME",      // DNAME
	40:    "SINK",       // SINK
	41:    "OPT",        // OPT
	42:    "APL",        // APL
	43:    "DS",         // Delegation Signer
	44:    "SSHFP",      // SSH Key Fingerprint
	45:    "IPSECKEY",   // IPSECKEY
	46:    "RRSIG",      // RRSIG
	47:    "NSEC",       // NSEC
	48:    "DNSKEY",     // DNSKEY
	49:    "DHCID",      // DHCID
	50:    "NSEC3",      // NSEC3
	51:    "NSEC3PARAM", // NSEC3PARAM
	52:    "TLSA",       // TLSA
	55:    "HIP",        // Host Identity Protocol
	56:    "NINFO",      // NINFO
	57:    "RKEY",       // RKEY
	58:    "TALINK",     // Trust Anchor LINK
	59:    "CDS",        // Child DS
	60:    "CDNSKEY",    // DNSKEY(s) the Child wants reflected in DS
	61:    "OPENPGPKEY", // OpenPGP Key
	62:    "CSYNC",      // Child-To-Parent Synchronization
	99:    "SPF",
	100:   "UINFO",
	101:   "UID",
	102:   "GID",
	103:   "UNSPEC",
	104:   "NID",
	105:   "L32",
	106:   "L64",
	107:   "LP",
	108:   "EUI48", // an EUI-48 address
	109:   "EUI64", // an EUI-64 address
	249:   "TKEY",  // Transaction Key
	250:   "TSIG",  // Transaction Signature
	251:   "IXFR",  // incremental transfer
	252:   "AXFR",  // transfer of an entire zone
	253:   "MAILB", // mailbox-related RRs (MB, MG or MR)
	254:   "MAILA", // mail agent RRs (OBSOLETE - see MX)
	255:   "ANY",   // A request for all records the server/cache has available
	256:   "URI",   // URI
	257:   "CAA",   // Certification Authority Restriction
	32768: "TA",    // DNSSEC Trust Authorities
	32769: "DLV",   // DNSSEC Lookaside Validation
}

var rcodeToStringMap = map[uint8]string{
	0:  "NOERROR",  // Success
	1:  "FORMERR",  // Format Error                       [RFC1035]
	2:  "SERVFAIL", // Server Failure                     [RFC1035]
	3:  "NXDOMAIN", // Non-Existent Domain                [RFC1035]
	4:  "NOTIMPL",  // Not Implemented                    [RFC1035]
	5:  "REFUSED",  // Query Refused                      [RFC1035]
	6:  "YXDOMAIN", // Name Exists when it should not     [RFC2136]
	7:  "YXRRSET",  // RR Set Exists when it should not   [RFC2136]
	8:  "NXRRSET",  // RR Set that should exist does not  [RFC2136]
	9:  "NOTAUTH",  // Server Not Authoritative for zone  [RFC2136]
	10: "NOTZONE",  // Name not contained in zone         [RFC2136]
	16: "BADSIG",   // TSIG Signature Failure             [RFC2845]
	//  "BADVERS",  // Bad OPT Version (also 16)          [RFC2671]
	17: "BADKEY",   // Key not recognized                 [RFC2845]
	18: "BADTIME",  // Signature out of time window       [RFC2845]
	19: "BADMODE",  // Bad TKEY Mode                      [RFC2930]
	20: "BADNAME",  // Duplicate key name                 [RFC2930]
	21: "BADALG",   // Algorithm not supported            [RFC2930]
	22: "BADTRUNC", // Bad Truncation                     [RFC4635]
}

// dnsOpCodeToString converts a DNSOpCode value to a string. If the type's
// string representation is unknown then the numeric value will be returned as
// a string.
func dnsOpCodeToString(opCode layers.DNSOpCode) string {
	s, exists := opCodeToStringMap[uint8(opCode)]
	if !exists {
		return strconv.Itoa(int(opCode))
	}
	return s
}

// dnsResponseCodeToString converts a DNSResponseCode value to a string. If
// the type's string representation is unknown then "Unknown <rcode value>"
// will be returned.
func dnsResponseCodeToString(rcode layers.DNSResponseCode) string {
	s, exists := rcodeToStringMap[uint8(rcode)]
	if !exists {
		return fmt.Sprintf("Unknown %d", int(rcode))
	}
	return s
}

// dnsTypeToString converts a DNSType value to a string. If the type's
// string representation is unknown then the numeric value will be returned
// as a string.
func dnsTypeToString(t layers.DNSType) string {
	s, exists := typeToStringMap[uint16(t)]
	if !exists {
		return strconv.Itoa(int(t))
	}
	return s
}

// dnsClassToString converts a DNSClass value to a string. If the class'es
// string representation is unknown then the numeric value will be returned
// as a string.
func dnsClassToString(c layers.DNSClass) string {
	s, exists := classToStringMap[uint16(c)]
	if !exists {
		return strconv.Itoa(int(c))
	}
	return s
}
