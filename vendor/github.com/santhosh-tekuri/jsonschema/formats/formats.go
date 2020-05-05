// Copyright 2017 Santhosh Kumar Tekuri. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package formats provides functions to check string against format.
//
// It allows developers to register custom formats, that can be used
// in json-schema for validation.
package formats

import (
	"net"
	"net/mail"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// The Format type is a function, to check
// whether given string is in valid format.
type Format func(string) bool

var formats = map[string]Format{
	"date-time":             IsDateTime,
	"date":                  IsDate,
	"time":                  IsTime,
	"hostname":              IsHostname,
	"email":                 IsEmail,
	"ip-address":            IsIPV4,
	"ipv4":                  IsIPV4,
	"ipv6":                  IsIPV6,
	"uri":                   IsURI,
	"iri":                   IsURI,
	"uri-reference":         IsURIReference,
	"uriref":                IsURIReference,
	"iri-reference":         IsURIReference,
	"uri-template":          IsURITemplate,
	"regex":                 IsRegex,
	"json-pointer":          IsJSONPointer,
	"relative-json-pointer": IsRelativeJSONPointer,
}

func init() {
	formats["format"] = IsFormat
}

// Register registers Format object for given format name.
func Register(name string, f Format) {
	formats[name] = f
}

// Get returns Format object for given format name, if found.
func Get(name string) (Format, bool) {
	f, ok := formats[name]
	return f, ok
}

// IsFormat tells whether given string is a valid format that is registered.
func IsFormat(s string) bool {
	_, ok := formats[s]
	return ok
}

// IsDateTime tells whether given string is a valid date representation
// as defined by RFC 3339, section 5.6.
//
// Note: this is unable to parse UTC leap seconds. See https://github.com/golang/go/issues/8728.
func IsDateTime(s string) bool {
	if _, err := time.Parse(time.RFC3339, s); err == nil {
		return true
	}
	if _, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return true
	}
	return false
}

// IsDate tells whether given string is a valid full-date production
// as defined by RFC 3339, section 5.6.
func IsDate(s string) bool {
	_, err := time.Parse("2006-01-02", s)
	return err == nil
}

// IsTime tells whether given string is a valid full-time production
// as defined by RFC 3339, section 5.6.
func IsTime(s string) bool {
	if _, err := time.Parse("15:04:05Z07:00", s); err == nil {
		return true
	}
	if _, err := time.Parse("15:04:05.999999999Z07:00", s); err == nil {
		return true
	}
	return false
}

// IsHostname tells whether given string is a valid representation
// for an Internet host name, as defined by RFC 1034, section 3.1.
//
// See https://en.wikipedia.org/wiki/Hostname#Restrictions_on_valid_host_names, for details.
func IsHostname(s string) bool {
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

		// labels could not start with a digit or with a hyphen
		if first := s[0]; (first >= '0' && first <= '9') || (first == '-') {
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

// IsEmail tells whether given string is a valid Internet email address
// as defined by RFC 5322, section 3.4.1.
//
// See https://en.wikipedia.org/wiki/Email_address, for details.
func IsEmail(s string) bool {
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
	if !IsHostname(domain) {
		return false
	}

	_, err := mail.ParseAddress(s)
	return err == nil
}

// IsIPV4 tells whether given string is a valid representation of an IPv4 address
// according to the "dotted-quad" ABNF syntax as defined in RFC 2673, section 3.2.
func IsIPV4(s string) bool {
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

// IsIPV6 tells whether given string is a valid representation of an IPv6 address
// as defined in RFC 2373, section 2.2.
func IsIPV6(s string) bool {
	if !strings.Contains(s, ":") {
		return false
	}
	return net.ParseIP(s) != nil
}

// IsURI tells whether given string is valid URI, according to RFC 3986.
func IsURI(s string) bool {
	u, err := url.Parse(s)
	return err == nil && u.IsAbs()
}

// IsURIReference tells whether given string is a valid URI Reference
// (either a URI or a relative-reference), according to RFC 3986.
func IsURIReference(s string) bool {
	_, err := url.Parse(s)
	return err == nil
}

// IsURITemplate tells whether given string is a valid URI Template
// according to RFC6570.
//
// Current implementation does minimal validation.
func IsURITemplate(s string) bool {
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

// IsRegex tells whether given string is a valid regular expression,
// according to the ECMA 262 regular expression dialect.
//
// The implementation uses go-lang regexp package.
func IsRegex(s string) bool {
	_, err := regexp.Compile(s)
	return err == nil
}

// IsJSONPointer tells whether given string is a valid JSON Pointer.
//
// Note: It returns false for JSON Pointer URI fragments.
func IsJSONPointer(s string) bool {
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

// IsRelativeJSONPointer tells whether given string is a valid Relative JSON Pointer.
//
// see https://tools.ietf.org/html/draft-handrews-relative-json-pointer-01#section-3
func IsRelativeJSONPointer(s string) bool {
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
	return s == "#" || IsJSONPointer(s)
}
