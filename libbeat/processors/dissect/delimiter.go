package dissect

import (
	"fmt"
	"strings"
)

//delimiter represents a text section after or before a key, it keeps track of the needle and allows
// to retrieve the position where it starts from a haystack.
type delimiter interface {
	// IndexOf receives the haystack and a offset position and will return the absolute position where
	// the needle is found.
	IndexOf(haystack string, offset int) int

	// Len returns the length of the needle used to calculate boundaries.
	Len() int

	// String displays debugging information.
	String() string

	// Delimiter returns the actual delimiter string.
	Delimiter() string

	// IsGreedy return true if the next key should be greedy (end of string) or when explicitely
	// configured.
	IsGreedy() bool

	// MarkGreedy marks this delimiter as greedy.
	MarkGreedy()

	// Next returns the next delimiter in the chain.
	Next() delimiter

	//SetNext sets the next delimiter or nil if current delimiter is the last.
	SetNext(d delimiter)
}

// zeroByte represents a zero string delimiter its usually start of the line.
type zeroByte struct {
	needle string
	greedy bool
	next   delimiter
}

func (z *zeroByte) IndexOf(haystack string, offset int) int {
	return offset
}

func (z *zeroByte) Len() int {
	return 0
}

func (z *zeroByte) String() string {
	return "delimiter: zerobyte"
}

func (z *zeroByte) Delimiter() string {
	return z.needle
}

func (z *zeroByte) IsGreedy() bool {
	return z.greedy
}

func (z *zeroByte) MarkGreedy() {
	z.greedy = true
}

func (z *zeroByte) Next() delimiter {
	return z.next
}

func (z *zeroByte) SetNext(d delimiter) {
	z.next = d
}

// multiByte represents a delimiter with at least one byte.
type multiByte struct {
	needle string
	greedy bool
	next   delimiter
}

func (m *multiByte) IndexOf(haystack string, offset int) int {
	i := strings.Index(haystack[offset:], m.needle)
	if i != -1 {
		return i + offset
	}
	return -1
}

func (m *multiByte) Len() int {
	return len(m.needle)
}

func (m *multiByte) IsGreedy() bool {
	return m.greedy
}

func (m *multiByte) MarkGreedy() {
	m.greedy = true
}

func (m *multiByte) String() string {
	return fmt.Sprintf("delimiter: multibyte (match: '%s', len: %d)", string(m.needle), m.Len())
}

func (m *multiByte) Delimiter() string {
	return m.needle
}

func (m *multiByte) Next() delimiter {
	return m.next
}

func (m *multiByte) SetNext(d delimiter) {
	m.next = d
}

func newDelimiter(needle string) delimiter {
	if len(needle) == 0 {
		return &zeroByte{}
	}
	return &multiByte{needle: needle}
}
