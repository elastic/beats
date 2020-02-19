// Copyright 2017 Santhosh Kumar Tekuri. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package jsonschema

import (
	"net"
	"net/mail"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Formats is a registry of functions, which know how to validate
// a specific format.
//
// New Formats can be registered by adding to this map. Key is format name,
// value is function that knows how to validate that format.
var Formats = map[string]func(interface{}) bool{
	"date-time":             isDateTime,
	"date":                  isDate,
	"time":                  isTime,
	"hostname":              isHostname,
	"email":                 isEmail,
	"ip-address":            isIPV4,
	"ipv4":                  isIPV4,
	"ipv6":                  isIPV6,
	"uri":                   isURI,
	"iri":                   isURI,
	"uri-reference":         isURIReference,
	"uriref":                isURIReference,
	"iri-reference":         isURIReference,
	"uri-template":          isURITemplate,
	"regex":                 isRegex,
	"json-pointer":          isJSONPointer,
	"relative-json-pointer": isRelativeJSONPointer,
}

// isDateTime tells whether given string is a valid date representation
// as defined by RFC 3339, section 5.6.
//
// Note: this is unable to parse UTC leap seconds. See https://github.com/golang/go/issues/8728.
func isDateTime(v interface{}) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	if _, err := time.Parse(time.RFC3339, s); err == nil {
		return true
	}
	if _, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return true
	}
	return false
}

// isDate tells whether given string is a valid full-date production
// as defined by RFC 3339, section 5.6.
func isDate(v interface{}) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	_, err := time.Parse("2006-01-02", s)
	return err == nil
}

// isTime tells whether given string is a valid full-time production
// as defined by RFC 3339, section 5.6.
func isTime(v interface{}) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	if _, err := time.Parse("15:04:05Z07:00", s); err == nil {
		return true
	}
	if _, err := time.Parse("15:04:05.999999999Z07:00", s); err == nil {
		return true
	}
	return false
}

// isHostname tells whether given string is a valid representation
// for an Internet host name, as defined by RFC 1034 section 3.1 and
// RFC 1123 section 2.1.
//
// See https://en.wikipedia.org/wiki/Hostname#Restrictions_on_valid_host_names, for details.
func isHostname(v interface{}) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	// entire hostname (including the delimiting dots but not a trailing dot) has a maximum of 253 ASCII characters
	s = strings.TrimSuffix(s, ".")
	if len(s) > 253 {
		return false
	}

	// Hostnames are composed of series of labels concatenated with dots, as are all domain names
	for _, label := range strings.Split(s, ".") {
		// Each label must be from 1 to 63 characters long
		if labelLen := len(label); labelLen < 1 || labelLen > 63 {
			return false
		}

		// labels must not start with a hyphen
		// RFC 1123 section 2.1: restriction on the first character
		// is relaxed to allow either a letter or a digit
		if first := s[0]; first == '-' {
			return false
		}

		// must not end with a hyphen
		if label[len(label)-1] == '-' {
			return false
		}

		// labels may contain only the ASCII letters 'a' through 'z' (in a case-insensitive manner),
		// the digits '0' through '9', and the hyphen ('-')
		for _, c := range label {
			if valid := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || (c == '-'); !valid {
				return false
			}
		}
	}

	return true
}

// isEmail tells whether given string is a valid Internet email address
// as defined by RFC 5322, section 3.4.1.
//
// See https://en.wikipedia.org/wiki/Email_address, for details.
func isEmail(v interface{}) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	// entire email address to be no more than 254 characters long
	if len(s) > 254 {
		return false
	}

	// email address is generally recognized as having two parts joined with an at-sign
	at := strings.LastIndexByte(s, '@')
	if at == -1 {
		return false
	}
	local := s[0:at]
	domain := s[at+1:]

	// local part may be up to 64 characters long
	if len(local) > 64 {
		return false
	}

	// domain must match the requirements for a hostname
	if !isHostname(domain) {
		return false
	}

	_, err := mail.ParseAddress(s)
	return err == nil
}

// isIPV4 tells whether given string is a valid representation of an IPv4 address
// according to the "dotted-quad" ABNF syntax as defined in RFC 2673, section 3.2.
func isIPV4(v interface{}) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	groups := strings.Split(s, ".")
	if len(groups) != 4 {
		return false
	}
	for _, group := range groups {
		n, err := strconv.Atoi(group)
		if err != nil {
			return false
		}
		if n < 0 || n > 255 {
			return false
		}
	}
	return true
}

// isIPV6 tells whether given string is a valid representation of an IPv6 address
// as defined in RFC 2373, section 2.2.
func isIPV6(v interface{}) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	if !strings.Contains(s, ":") {
		return false
	}
	return net.ParseIP(s) != nil
}

// isURI tells whether given string is valid URI, according to RFC 3986.
func isURI(v interface{}) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	u, err := url.Parse(s)
	return err == nil && u.IsAbs()
}

// isURIReference tells whether given string is a valid URI Reference
// (either a URI or a relative-reference), according to RFC 3986.
func isURIReference(v interface{}) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	_, err := url.Parse(s)
	return err == nil
}

// isURITemplate tells whether given string is a valid URI Template
// according to RFC6570.
//
// Current implementation does minimal validation.
func isURITemplate(v interface{}) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	for _, item := range strings.Split(u.RawPath, "/") {
		depth := 0
		for _, ch := range item {
			switch ch {
			case '{':
				depth++
				if depth != 1 {
					return false
				}
			case '}':
				depth--
				if depth != 0 {
					return false
				}
			}
		}
		if depth != 0 {
			return false
		}
	}
	return true
}

// isRegex tells whether given string is a valid regular expression,
// according to the ECMA 262 regular expression dialect.
//
// The implementation uses go-lang regexp package.
func isRegex(v interface{}) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	_, err := regexp.Compile(s)
	return err == nil
}

// isJSONPointer tells whether given string is a valid JSON Pointer.
//
// Note: It returns false for JSON Pointer URI fragments.
func isJSONPointer(v interface{}) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	if s != "" && !strings.HasPrefix(s, "/") {
		return false
	}
	for _, item := range strings.Split(s, "/") {
		for i := 0; i < len(item); i++ {
			if item[i] == '~' {
				if i == len(item)-1 {
					return false
				}
				switch item[i+1] {
				case '~', '0', '1':
					// valid
				default:
					return false
				}
			}
		}
	}
	return true
}

// isRelativeJSONPointer tells whether given string is a valid Relative JSON Pointer.
//
// see https://tools.ietf.org/html/draft-handrews-relative-json-pointer-01#section-3
func isRelativeJSONPointer(v interface{}) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	if s == "" {
		return false
	}
	if s[0] == '0' {
		s = s[1:]
	} else if s[0] >= '0' && s[0] <= '9' {
		for s != "" && s[0] >= '0' && s[0] <= '9' {
			s = s[1:]
		}
	} else {
		return false
	}
	return s == "#" || isJSONPointer(s)
}
