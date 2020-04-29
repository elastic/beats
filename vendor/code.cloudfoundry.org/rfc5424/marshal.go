package rfc5424

import (
	"bytes"
	"fmt"
	"unicode/utf8"
)

// allowLongSdNames is true to allow names longer than the RFC-specified limit
// of 32-characters. (When true, this violates RFC-5424).
const allowLongSdNames = true

// RFC5424TimeOffsetNum is the timestamp defined by RFC-5424 with the
// NUMOFFSET instead of Z.
const RFC5424TimeOffsetNum = "2006-01-02T15:04:05.999999-07:00"

// RFC5424TimeOffsetUTC is the timestamp defined by RFC-5424 with the offset
// set to 0 for UTC.
const RFC5424TimeOffsetUTC = "2006-01-02T15:04:05.999999Z"

// ErrInvalidValue is returned when a log message cannot be emitted because one
// of the values is invalid.
type ErrInvalidValue struct {
	Property string
	Value    interface{}
}

func (e ErrInvalidValue) Error() string {
	return fmt.Sprintf("Message cannot be serialized because %s is invalid: %v",
		e.Property, e.Value)
}

// invalidValue returns an invalid value error with the given property
func invalidValue(property string, value interface{}) error {
	return ErrInvalidValue{Property: property, Value: value}
}

func nilify(x string) string {
	if x == "" {
		return "-"
	}
	return x
}

func escapeSDParam(s string) string {
	escapeCount := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\\', '"', ']':
			escapeCount++
		}
	}
	if escapeCount == 0 {
		return s
	}

	t := make([]byte, len(s)+escapeCount)
	j := 0
	for i := 0; i < len(s); i++ {
		switch c := s[i]; c {
		case '\\', '"', ']':
			t[j] = '\\'
			t[j+1] = c
			j += 2
		default:
			t[j] = s[i]
			j++
		}
	}
	return string(t)
}

func isPrintableUsASCII(s string) bool {
	for _, ch := range s {
		if ch < 33 || ch > 126 {
			return false
		}
	}
	return true
}

func isValidSdName(s string) bool {
	if !allowLongSdNames && len(s) > 32 {
		return false
	}
	for _, ch := range s {
		if ch < 33 || ch > 126 {
			return false
		}
		if ch == '=' || ch == ']' || ch == '"' {
			return false
		}
	}
	return true
}

func (m Message) assertValid() error {

	// HOSTNAME        = NILVALUE / 1*255PRINTUSASCII
	if !isPrintableUsASCII(m.Hostname) {
		return invalidValue("Hostname", m.Hostname)
	}
	if len(m.Hostname) > 255 {
		return invalidValue("Hostname", m.Hostname)
	}

	// APP-NAME        = NILVALUE / 1*48PRINTUSASCII
	if !isPrintableUsASCII(m.AppName) {
		return invalidValue("AppName", m.AppName)
	}
	if len(m.AppName) > 48 {
		return invalidValue("AppName", m.AppName)
	}

	// PROCID          = NILVALUE / 1*128PRINTUSASCII
	if !isPrintableUsASCII(m.ProcessID) {
		return invalidValue("ProcessID", m.ProcessID)
	}
	if len(m.ProcessID) > 128 {
		return invalidValue("ProcessID", m.ProcessID)
	}

	// MSGID           = NILVALUE / 1*32PRINTUSASCII
	if !isPrintableUsASCII(m.MessageID) {
		return invalidValue("MessageID", m.MessageID)
	}
	if len(m.MessageID) > 32 {
		return invalidValue("MessageID", m.MessageID)
	}

	for _, sdElement := range m.StructuredData {
		if !isValidSdName(sdElement.ID) {
			return invalidValue("StructuredData/ID", sdElement.ID)
		}
		for _, sdParam := range sdElement.Parameters {
			if !isValidSdName(sdParam.Name) {
				return invalidValue("StructuredData/Name", sdParam.Name)
			}
			if !utf8.ValidString(sdParam.Value) {
				return invalidValue("StructuredData/Value", sdParam.Value)
			}
		}
	}
	return nil
}

// MarshalBinary marshals the message to a byte slice, or returns an error
func (m Message) MarshalBinary() ([]byte, error) {
	if err := m.assertValid(); err != nil {
		return nil, err
	}

	b := bytes.NewBuffer(nil)
	fmt.Fprintf(b, "<%d>1 %s %s %s %s %s ",
		m.Priority,
		m.Timestamp.Format(RFC5424TimeOffsetNum),
		nilify(m.Hostname),
		nilify(m.AppName),
		nilify(m.ProcessID),
		nilify(m.MessageID))

	if len(m.StructuredData) == 0 {
		fmt.Fprint(b, "-")
	}
	for _, sdElement := range m.StructuredData {
		fmt.Fprintf(b, "[%s", sdElement.ID)
		for _, sdParam := range sdElement.Parameters {
			fmt.Fprintf(b, " %s=\"%s\"", sdParam.Name,
				escapeSDParam(sdParam.Value))
		}
		fmt.Fprintf(b, "]")
	}

	if len(m.Message) > 0 {
		fmt.Fprint(b, " ")
		b.Write(m.Message)
	}
	return b.Bytes(), nil
}
