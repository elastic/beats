package rpm

import (
	"time"
)

// Header index value data types.
const (
	IndexDataTypeNull int = iota
	IndexDataTypeChar
	IndexDataTypeInt8
	IndexDataTypeInt16
	IndexDataTypeInt32
	IndexDataTypeInt64
	IndexDataTypeString
	IndexDataTypeBinary
	IndexDataTypeStringArray
	IndexDataTypeI8NString
)

// An IndexEntry is a rpm key/value tag stored in the package header.
type IndexEntry struct {
	Tag       int
	Type      int
	Offset    int
	ItemCount int
	Value     interface{}
}

// IndexEntries is an array of IndexEntry structs.
type IndexEntries []IndexEntry

// IndexByTag returns a pointer to an IndexEntry with the given tag ID or nil if
// the tag is not found.
func (c IndexEntries) IndexByTag(tag int) *IndexEntry {
	for _, e := range c {
		if e.Tag == tag {
			return &e
		}
	}

	return nil
}

// StringByTag returns the string value of an IndexEntry or an empty string if
// the tag is not found or has no value.
func (c IndexEntries) StringByTag(tag int) string {
	i := c.IndexByTag(tag)
	if i == nil || i.Value == nil {
		return ""
	}

	s := i.Value.([]string)

	return s[0]
}

// StringsByTag returns the slice of string values of an IndexEntry or nil if
// the tag is not found or has no value.
func (c IndexEntries) StringsByTag(tag int) []string {
	i := c.IndexByTag(tag)
	if i == nil || i.Value == nil {
		return nil
	}

	return i.Value.([]string)
}

// IntsByTag returns the int64 values of an IndexEntry or nil if the tag is not
// found or has no value. Values with a lower range (E.g. int8) are cast as an
// int64.
func (c IndexEntries) IntsByTag(tag int) []int64 {
	ix := c.IndexByTag(tag)
	if ix != nil && ix.Value != nil {
		vals := make([]int64, ix.ItemCount)

		for i := 0; i < int(ix.ItemCount); i++ {
			switch ix.Type {
			case IndexDataTypeChar, IndexDataTypeInt8:
				vals[i] = int64(ix.Value.([]int8)[i])

			case IndexDataTypeInt16:
				vals[i] = int64(ix.Value.([]int16)[i])

			case IndexDataTypeInt32:
				vals[i] = int64(ix.Value.([]int32)[i])

			case IndexDataTypeInt64:
				vals[i] = ix.Value.([]int64)[i]
			}
		}

		return vals
	}

	return nil
}

// IntByTag returns the int64 value of an IndexEntry or 0 if the tag is not found
// or has no value. Values with a lower range (E.g. int8) are cast as an int64.
func (c IndexEntries) IntByTag(tag int) int64 {
	i := c.IndexByTag(tag)
	if i != nil && i.Value != nil {
		switch i.Type {
		case IndexDataTypeChar, IndexDataTypeInt8:
			return int64(i.Value.([]int8)[0])

		case IndexDataTypeInt16:
			return int64(i.Value.([]int16)[0])

		case IndexDataTypeInt32:
			return int64(i.Value.([]int32)[0])

		case IndexDataTypeInt64:
			return int64(i.Value.([]int64)[0])
		}
	}

	return 0
}

// BytesByTag returns the raw value of an IndexEntry or nil if the tag is not
// found or has no value.
func (c IndexEntries) BytesByTag(tag int) []byte {
	i := c.IndexByTag(tag)
	if i == nil || i.Value == nil {
		return nil
	}

	return i.Value.([]byte)
}

// TimesByTag returns the value of an IndexEntry as a slice of Go native
// timestamps or nil if the tag is not found or has no value.
func (c IndexEntries) TimesByTag(tag int) []time.Time {
	ix := c.IndexByTag(tag)

	if ix == nil || ix.Value == nil {
		return nil
	}

	vals := make([]time.Time, ix.ItemCount)
	for i := 0; i < ix.ItemCount; i++ {
		vals[i] = time.Unix(int64(ix.Value.([]int32)[i]), 0)
	}

	return vals
}

// TimeByTag returns the value of an IndexEntry as a Go native timestamp or
// zero-time if the tag is not found or has no value.
func (c IndexEntries) TimeByTag(tag int) time.Time {
	vals := c.TimesByTag(tag)
	if vals == nil || len(vals) == 0 {
		return time.Time{}
	}

	return vals[0]
}
