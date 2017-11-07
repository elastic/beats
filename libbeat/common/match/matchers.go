package match

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"unsafe"
)

type equalsMatcher struct {
	s  string
	bs []byte
}

type substringMatcher struct {
	s  string
	bs []byte
}

type altSubstringMatcher struct {
	literals [][]byte
}

type oneOfMatcher struct {
	literals [][]byte
}

type prefixMatcher struct {
	s []byte
}

type altPrefixMatcher struct {
	literals [][]byte
}

type prefixNumDate struct {
	minLen int
	prefix []byte
	digits []int
	seps   [][]byte
	suffix []byte
}

type emptyStringMatcher struct{}

type emptyWhiteStringMatcher struct{}

type matchAny struct{}

func (m *equalsMatcher) MatchString(s string) bool {
	return m.s == s
}

func (m *equalsMatcher) Match(bs []byte) bool {
	return bytes.Equal(bs, m.bs)
}

func (m *equalsMatcher) String() string {
	return fmt.Sprintf("<string '%v'>", m.s)
}

func (m *substringMatcher) MatchString(s string) bool {
	return strings.Contains(s, m.s)
}

func (m *substringMatcher) Match(bs []byte) bool {
	return bytes.Contains(bs, m.bs)
}

func (m *substringMatcher) String() string {
	return fmt.Sprintf("<substring '%v'>", m.s)
}

func (m *altSubstringMatcher) MatchString(s string) bool {
	return m.Match(stringToBytes(s))
}

func (m *altSubstringMatcher) Match(in []byte) bool {
	for _, literal := range m.literals {
		if bytes.Contains(in, literal) {
			return true
		}
	}
	return false
}

func (m *altSubstringMatcher) String() string {
	return fmt.Sprintf("<alt substring '%s'>", bytes.Join(m.literals, []byte(",")))
}

func (m *oneOfMatcher) MatchString(s string) bool {
	return m.Match(stringToBytes(s))
}

func (m *oneOfMatcher) Match(in []byte) bool {
	for _, literal := range m.literals {
		if bytes.Equal(in, literal) {
			return true
		}
	}
	return false
}

func (m *oneOfMatcher) String() string {
	return fmt.Sprintf("<one of '%s'>", bytes.Join(m.literals, []byte(",")))
}

func (m *prefixMatcher) MatchString(s string) bool {
	return len(s) >= len(m.s) && s[0:len(m.s)] == string(m.s)
}

func (m *prefixMatcher) Match(bs []byte) bool {
	return len(bs) >= len(m.s) && bytes.Equal(bs[0:len(m.s)], m.s)
}

func (m *prefixMatcher) String() string {
	return fmt.Sprintf("<prefix string '%v'>", string(m.s))
}

func (m *altPrefixMatcher) MatchString(in string) bool {
	for _, s := range m.literals {
		if len(in) >= len(s) && in[0:len(s)] == string(s) {
			return true
		}
	}
	return false
}

func (m *altPrefixMatcher) Match(bs []byte) bool {
	for _, s := range m.literals {
		if len(bs) >= len(s) && bytes.Equal(bs[0:len(s)], s) {
			return true
		}
	}
	return false
}

func (m *altPrefixMatcher) String() string {
	return fmt.Sprintf("<alt prefix string '%s'>", bytes.Join(m.literals, []byte(",")))
}

func (m *prefixNumDate) MatchString(in string) bool {
	return m.Match(stringToBytes(in))
}

func (m *prefixNumDate) Match(in []byte) bool {
	if len(in) < m.minLen {
		return false
	}

	pos := 0
	if m.prefix != nil {
		end := len(m.prefix)
		if !bytes.Equal(in[0:end], m.prefix) {
			return false
		}

		pos += end
	}

	for cnt := m.digits[0]; cnt > 0; cnt-- {
		v := in[pos]
		pos++
		if !('0' <= v && v <= '9') {
			return false
		}
	}

	for i := 1; i < len(m.digits); i++ {
		sep := m.seps[i-1]
		if !bytes.Equal(in[pos:pos+len(sep)], sep) {
			return false
		}

		pos += len(sep)
		for cnt := m.digits[i]; cnt > 0; cnt-- {
			v := in[pos]
			pos++
			if !('0' <= v && v <= '9') {
				return false
			}
		}
	}

	if sfx := m.suffix; len(sfx) > 0 {
		if !bytes.HasPrefix(in[pos:], sfx) {
			return false
		}
	}

	return true
}

func (m *prefixNumDate) String() string {
	return "<prefix num date>"
}

func (m *emptyStringMatcher) MatchString(s string) bool {
	return len(s) == 0
}

func (m *emptyStringMatcher) Match(bs []byte) bool {
	return len(bs) == 0
}

func (m *emptyStringMatcher) String() string {
	return "<empty>"
}

func (m *emptyWhiteStringMatcher) MatchString(s string) bool {
	for _, r := range s {
		if !(r == 0x9 || r == 0xa || r == 0xc || r == 0xd || r == 0x20 || r == '\t') {
			return false
		}
	}
	return true
}

func (m *emptyWhiteStringMatcher) Match(bs []byte) bool {
	for _, r := range bytesToString(bs) {
		if !(r == 0x9 || r == 0xa || r == 0xc || r == 0xd || r == 0x20 || r == '\t') {
			return false
		}
	}
	return true
}

func (m *emptyWhiteStringMatcher) String() string {
	return "<empty whitespace>"
}

func (m *matchAny) Match(_ []byte) bool       { return true }
func (m *matchAny) MatchString(_ string) bool { return true }
func (m *matchAny) String() string            { return "<any>" }

func bytesToString(b []byte) string {
	bh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	sh := reflect.StringHeader{Data: bh.Data, Len: bh.Len}
	return *(*string)(unsafe.Pointer(&sh))
}

func stringToBytes(s string) []byte {
	sh := (*reflect.StringHeader)(unsafe.Pointer(&s))
	bh := reflect.SliceHeader{Data: sh.Data, Len: sh.Len, Cap: sh.Len}
	return *(*[]byte)(unsafe.Pointer(&bh))
}
