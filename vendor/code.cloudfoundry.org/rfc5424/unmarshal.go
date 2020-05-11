package rfc5424

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"time"
	"unicode"
)

// ErrBadFormat is the error that is returned when a log message cannot be parsed
type ErrBadFormat struct {
	Property string
}

func (e ErrBadFormat) Error() string {
	return fmt.Sprintf("Message cannot be unmarshaled because it is not well formed (%s)",
		e.Property)
}

// badFormat returns a bad format error with the given property
func badFormat(property string) error {
	return ErrBadFormat{Property: property}
}

// UnmarshalBinary unmarshals a byte slice into a message
func (m *Message) UnmarshalBinary(inputBuffer []byte) error {
	r := bytes.NewBuffer(inputBuffer)

	// RFC-5424
	// SYSLOG-MSG      = HEADER SP STRUCTURED-DATA [SP MSG]
	if err := m.readHeader(r); err != nil {
		return err
	}

	if err := readSpace(r); err != nil {
		return err // unreachable
	}
	if err := m.readStructuredData(r); err != nil {
		return err
	}

	// MSG is optional
	ch, _, err := r.ReadRune()
	if err == io.EOF {
		return nil
	} else if ch != ' ' {
		return badFormat("MSG") // unreachable
	}

	// TODO(ross): detect and handle UTF-8 BOM (\xef\xbb\xbf)
	//
	// MSG             = MSG-ANY / MSG-UTF8
	// MSG-ANY         = *OCTET ; not starting with BOM
	// MSG-UTF8        = BOM UTF-8-STRING
	// BOM             = %xEF.BB.BF

	// To be on the safe side, remaining stuff is copied over
	m.Message = copyFrom(r.Bytes())
	return nil
}

// readHeader reads a HEADER as defined in RFC-5424
//
// HEADER          = PRI VERSION SP TIMESTAMP SP HOSTNAME
// SP APP-NAME SP PROCID SP MSGID
// PRI             = "<" PRIVAL ">"
// PRIVAL          = 1*3DIGIT ; range 0 .. 191
// VERSION         = NONZERO-DIGIT 0*2DIGIT
// HOSTNAME        = NILVALUE / 1*255PRINTUSASCII
//
// APP-NAME        = NILVALUE / 1*48PRINTUSASCII
// PROCID          = NILVALUE / 1*128PRINTUSASCII
// MSGID           = NILVALUE / 1*32PRINTUSASCII
//
// TIMESTAMP       = NILVALUE / FULL-DATE "T" FULL-TIME
// FULL-DATE       = DATE-FULLYEAR "-" DATE-MONTH "-" DATE-MDAY
// DATE-FULLYEAR   = 4DIGIT
// DATE-MONTH      = 2DIGIT  ; 01-12
// DATE-MDAY       = 2DIGIT  ; 01-28, 01-29, 01-30, 01-31 based on
// ; month/year
// FULL-TIME       = PARTIAL-TIME TIME-OFFSET
// PARTIAL-TIME    = TIME-HOUR ":" TIME-MINUTE ":" TIME-SECOND
// [TIME-SECFRAC]
// TIME-HOUR       = 2DIGIT  ; 00-23
// TIME-MINUTE     = 2DIGIT  ; 00-59
// TIME-SECOND     = 2DIGIT  ; 00-59
// TIME-SECFRAC    = "." 1*6DIGIT
// TIME-OFFSET     = "Z" / TIME-NUMOFFSET
// TIME-NUMOFFSET  = ("+" / "-") TIME-HOUR ":" TIME-MINUTE
//
func (m *Message) readHeader(r io.RuneScanner) error {
	if err := m.readPriority(r); err != nil {
		return err
	}
	if err := m.readVersion(r); err != nil {
		return err
	}
	if err := readSpace(r); err != nil {
		return err // unreachable
	}
	if err := m.readTimestamp(r); err != nil {
		return err
	}
	if err := readSpace(r); err != nil {
		return err // unreachable
	}
	if err := m.readHostname(r); err != nil {
		return err
	}
	if err := readSpace(r); err != nil {
		return err // unreachable
	}
	if err := m.readAppName(r); err != nil {
		return err
	}
	if err := readSpace(r); err != nil {
		return err // unreachable
	}
	if err := m.readProcID(r); err != nil {
		return err
	}
	if err := readSpace(r); err != nil {
		return err // unreachable
	}
	if err := m.readMsgID(r); err != nil {
		return err
	}
	return nil
}

// readPriority reads the PRI as defined in RFC-5424 and assigns Severity and
// Facility accordingly.
func (m *Message) readPriority(r io.RuneScanner) error {
	ch, _, err := r.ReadRune()
	if err != nil {
		return err
	}
	if ch != '<' {
		return badFormat("Priority")
	}

	rv := &bytes.Buffer{}
	for {
		ch, _, err := r.ReadRune()
		if err != nil {
			return err
		}
		if unicode.IsDigit(ch) {
			rv.WriteRune(ch)
			continue
		}
		if ch != '>' {
			return badFormat("Priority")
		}

		// We have a complete integer expression
		priority, err := strconv.ParseInt(string(rv.Bytes()), 10, 32)
		if err != nil {
			return badFormat("Priority")
		}
		m.Priority = Priority(priority)
		return nil
	}
}

// readVersion reads the version string fails if it isn't `1`
func (m *Message) readVersion(r io.RuneScanner) error {
	ch, _, err := r.ReadRune()
	if err != nil {
		return err
	}
	if ch != '1' {
		return badFormat("Version")
	}
	return nil
}

// readTimestamp reads a TIMESTAMP as defined in RFC-5424 and assigns
// m.Timestamp
//
// TIMESTAMP       = NILVALUE / FULL-DATE "T" FULL-TIME
// FULL-DATE       = DATE-FULLYEAR "-" DATE-MONTH "-" DATE-MDAY
// DATE-FULLYEAR   = 4DIGIT
// DATE-MONTH      = 2DIGIT  ; 01-12
// DATE-MDAY       = 2DIGIT  ; 01-28, 01-29, 01-30, 01-31 based on
//                           ; month/year
// FULL-TIME       = PARTIAL-TIME TIME-OFFSET
// PARTIAL-TIME    = TIME-HOUR ":" TIME-MINUTE ":" TIME-SECOND
// [TIME-SECFRAC]
// TIME-HOUR       = 2DIGIT  ; 00-23
// TIME-MINUTE     = 2DIGIT  ; 00-59
// TIME-SECOND     = 2DIGIT  ; 00-59
// TIME-SECFRAC    = "." 1*6DIGIT
// TIME-OFFSET     = "Z" / TIME-NUMOFFSET
// TIME-NUMOFFSET  = ("+" / "-") TIME-HOUR ":" TIME-MINUTE
func (m *Message) readTimestamp(r io.RuneScanner) error {
	timestampString, err := readWord(r)
	if err != nil {
		return err
	}

	m.Timestamp, err = time.Parse(RFC5424TimeOffsetNum, timestampString)
	if err == nil {
		return nil
	}

	m.Timestamp, err = time.Parse(RFC5424TimeOffsetUTC, timestampString)
	if err == nil {
		return nil
	}

	return err
}

func (m *Message) readHostname(r io.RuneScanner) (err error) {
	m.Hostname, err = readWord(r)
	return err
}

func (m *Message) readAppName(r io.RuneScanner) (err error) {
	m.AppName, err = readWord(r)
	return err
}

func (m *Message) readProcID(r io.RuneScanner) (err error) {
	m.ProcessID, err = readWord(r)
	return err
}

func (m *Message) readMsgID(r io.RuneScanner) (err error) {
	m.MessageID, err = readWord(r)
	return err
}

// readStructuredData reads a STRUCTURED-DATA (as defined in RFC-5424)
// from `r` and assigns the StructuredData member.
//
// STRUCTURED-DATA = NILVALUE / 1*SD-ELEMENT
// SD-ELEMENT      = "[" SD-ID *(SP SD-PARAM) "]"
// SD-PARAM        = PARAM-NAME "=" %d34 PARAM-VALUE %d34
// SD-ID           = SD-NAME
// PARAM-NAME      = SD-NAME
// PARAM-VALUE     = UTF-8-STRING ; characters '"', '\' and ']' MUST be escaped.
// SD-NAME         = 1*32PRINTUSASCII except '=', SP, ']', %d34 (")
func (m *Message) readStructuredData(r io.RuneScanner) (err error) {
	m.StructuredData = []StructuredData{}

	ch, _, err := r.ReadRune()
	if err != nil {
		return err
	}
	if ch == '-' {
		return nil
	}
	r.UnreadRune()

	for {
		ch, _, err := r.ReadRune()
		if err == io.EOF {
			return nil
		} else if err != nil {
			return err // hard to reach without underlying IO error
		} else if ch == ' ' {
			r.UnreadRune()
			return nil
		} else if ch == '[' {
			r.UnreadRune()
			sde, err := readSDElement(r)
			if err != nil {
				return err
			}
			m.StructuredData = append(m.StructuredData, sde)
		} else {
			return badFormat("StructuredData")
		}
	}
}

// readSDElement reads an SD-ELEMENT as defined by RFC-5424
//
// SD-ELEMENT      = "[" SD-ID *(SP SD-PARAM) "]"
// SD-PARAM        = PARAM-NAME "=" %d34 PARAM-VALUE %d34
// SD-ID           = SD-NAME
// PARAM-NAME      = SD-NAME
// PARAM-VALUE     = UTF-8-STRING ; characters '"', '\' and ']' MUST be escaped.
// SD-NAME         = 1*32PRINTUSASCII except '=', SP, ']', %d34 (")
func readSDElement(r io.RuneScanner) (element StructuredData, err error) {
	ch, _, err := r.ReadRune()
	if err != nil {
		return element, err // hard to reach without underlying IO error
	}
	if ch != '[' {
		return element, badFormat("StructuredData[]") // unreachable
	}
	element.ID, err = readSdID(r)
	if err != nil {
		return element, err
	}
	for {
		ch, _, err := r.ReadRune()
		if err != nil {
			return element, err
		} else if ch == ']' {
			return element, nil
		} else if ch == ' ' {
			param, err := readSdParam(r)
			if err != nil {
				return element, err
			}
			element.Parameters = append(element.Parameters, *param)
		} else {
			return element, badFormat("StructuredData[]")
		}
	}
}

// readSDID reads an SD-ID as defined by RFC-5424
// SD-ID           = SD-NAME
// SD-NAME         = 1*32PRINTUSASCII except '=', SP, ']', %d34 (")
func readSdID(r io.RuneScanner) (string, error) {
	rv := &bytes.Buffer{}
	for {
		ch, _, err := r.ReadRune()
		if err != nil {
			return "", err
		}
		if ch == ' ' || ch == ']' {
			r.UnreadRune()
			return string(rv.Bytes()), nil
		}
		rv.WriteRune(ch)
	}
}

// readSdParam reads an SD-PARAM as defined by RFC-5424
// SD-PARAM        = PARAM-NAME "=" %d34 PARAM-VALUE %d34
// SD-ID           = SD-NAME
// PARAM-NAME      = SD-NAME
// PARAM-VALUE     = UTF-8-STRING ; characters '"', '\' and ']' MUST be escaped.
// SD-NAME         = 1*32PRINTUSASCII except '=', SP, ']', %d34 (")
func readSdParam(r io.RuneScanner) (sdp *SDParam, err error) {
	sdp = &SDParam{}
	sdp.Name, err = readSdParamName(r)
	if err != nil {
		return nil, err
	}
	ch, _, err := r.ReadRune()
	if err != nil {
		return nil, err // hard to reach
	}
	if ch != '=' {
		return nil, badFormat("StructuredData[].Parameters") // not reachable
	}

	sdp.Value, err = readSdParamValue(r)
	if err != nil {
		return nil, err
	}
	return sdp, nil
}

// readSdParam reads a PARAM-NAME as defined by RFC-5424
// SD-PARAM        = PARAM-NAME "=" %d34 PARAM-VALUE %d34
// PARAM-NAME      = SD-NAME
// SD-NAME         = 1*32PRINTUSASCII except '=', SP, ']', %d34 (")
func readSdParamName(r io.RuneScanner) (string, error) {
	rv := &bytes.Buffer{}
	for {
		ch, _, err := r.ReadRune()
		if err != nil {
			return "", err
		}
		if ch == '=' {
			r.UnreadRune()
			return string(rv.Bytes()), nil
		}
		rv.WriteRune(ch)
	}
}

// readSdParamValue reads an PARAM-VALUE as defined by RFC-5424
// SD-PARAM        = PARAM-NAME "=" %d34 PARAM-VALUE %d34
// PARAM-VALUE     = UTF-8-STRING ; characters '"', '\' and ']' MUST be escaped.
func readSdParamValue(r io.RuneScanner) (string, error) {
	ch, _, err := r.ReadRune()
	if err != nil {
		return "", err
	}
	if ch != '"' {
		return "", badFormat("StructuredData[].Parameters[]") // hard to reach
	}

	rv := &bytes.Buffer{}
	for {
		ch, _, err := r.ReadRune()
		if err != nil {
			return "", err
		}
		if ch == '\\' {
			ch, _, err := r.ReadRune()
			if err != nil {
				return "", err
			}
			rv.WriteRune(ch)
			continue
		}
		if ch == '"' {
			return string(rv.Bytes()), nil
		}
		rv.WriteRune(ch)
	}
}

// readSpace reads a single space
func readSpace(r io.RuneScanner) error {
	ch, _, err := r.ReadRune()
	if err != nil {
		return err
	}
	if ch != ' ' {
		return badFormat("expected space")
	}
	return nil
}

// readWord reads `r` until it encounters a space (0x20)
func readWord(r io.RuneScanner) (string, error) {
	rv := &bytes.Buffer{}
	for {
		ch, _, err := r.ReadRune()
		if err != nil {
			return "", err
		} else if ch != ' ' {
			rv.WriteRune(ch)
		} else {
			r.UnreadRune()
			rvString := string(rv.Bytes())
			if rvString == "-" {
				rvString = ""
			}
			return rvString, nil
		}
	}
}

func copyFrom(in []byte) []byte {
	out := make([]byte, len(in))
	copy(out, in)
	return out
}
