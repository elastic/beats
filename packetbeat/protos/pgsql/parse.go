package pgsql

import (
	"errors"
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

var (
	errInvalidString     = errors.New("invalid pgsql string")
	errEmptyFieldsBuffer = errors.New("empty fields buffer")
	errNoFieldName       = errors.New("can not read column field")
	errFieldBufferShort  = errors.New("field buffer to small for field count")
	errFieldBufferBig    = errors.New("field count to small for field buffer size")
)

func (pgsql *Pgsql) pgsqlMessageParser(s *PgsqlStream) (bool, bool) {
	debugf("pgsqlMessageParser, off=%v", s.parseOffset)

	var ok, complete bool

	switch s.parseState {
	case PgsqlStartState:
		ok, complete = pgsql.parseMessageStart(s)
	case PgsqlGetDataState:
		ok, complete = pgsql.parseMessageData(s)
	case PgsqlExtendedQueryState:
		ok, complete = pgsql.parseMessageExtendedQuery(s)
	default:
		logp.Critical("Pgsql invalid parser state")
	}

	detailedf("pgsqlMessageParser return: ok=%v, complete=%v, off=%v",
		ok, complete, s.parseOffset)

	return ok, complete
}

func (pgsql *Pgsql) parseMessageStart(s *PgsqlStream) (bool, bool) {
	detailedf("parseMessageStart")

	m := s.message

	for len(s.data[s.parseOffset:]) >= 5 {
		isSpecial, length, command := isSpecialPgsqlCommand(s.data[s.parseOffset:])
		if !isSpecial {
			return pgsql.parseCommand(s)
		}

		// In case of Commands: StartupMessage, SSLRequest, CancelRequest that don't have
		// their type in the first byte

		// check buffer available
		if len(s.data[s.parseOffset:]) <= length {
			detailedf("Wait for more data 1")
			return true, false
		}

		// ignore non SSLRequest commands
		if command != SSLRequest {
			s.parseOffset += length
			continue
		}

		// if SSLRequest is received, expect for one byte reply (S or N)
		m.start = s.parseOffset
		s.parseOffset += length
		m.end = s.parseOffset
		m.isSSLRequest = true
		m.Size = uint64(m.end - m.start)

		return true, true
	}
	return true, false
}

func (pgsql *Pgsql) parseCommand(s *PgsqlStream) (bool, bool) {
	// read type
	typ := byte(s.data[s.parseOffset])

	if s.expectSSLResponse {
		// SSLRequest was received in the other stream
		if typ == 'N' || typ == 'S' {
			m := s.message

			// one byte reply to SSLRequest
			detailedf("Reply for SSLRequest %c", typ)
			m.start = s.parseOffset
			s.parseOffset += 1
			m.end = s.parseOffset
			m.isSSLResponse = true
			m.Size = uint64(m.end - m.start)

			return true, true
		}
	}

	// read length
	length := readLength(s.data[s.parseOffset+1:])
	if length < 4 {
		// length should include the size of itself (int32)
		detailedf("Invalid pgsql command length.")
		return false, false
	}
	if len(s.data[s.parseOffset:]) <= length {
		detailedf("Wait for more data")
		return true, false
	}

	detailedf("Pgsql type %c, length=%d", typ, length)

	switch typ {
	case 'Q':
		return pgsql.parseSimpleQuery(s, length)
	case 'T':
		return pgsql.parseRowDescription(s, length)
	case 'I':
		return pgsql.parseEmptyQueryResponse(s)
	case 'C':
		return pgsql.parseCommandComplete(s, length)
	case 'Z':
		return pgsql.parseReadyForQuery(s, length)
	case 'E':
		return pgsql.parseErrorResponse(s, length)
	case 'P':
		return pgsql.parseExtReq(s, length)
	case '1':
		return pgsql.parseExtResp(s, length)
	default:
		if !pgsqlValidType(typ) {
			detailedf("invalid frame type: '%c'", typ)
			return false, false
		}
		return pgsql.parseSkipMessage(s, length)
	}
}

func (pgsql *Pgsql) parseSimpleQuery(s *PgsqlStream, length int) (bool, bool) {
	m := s.message
	m.start = s.parseOffset
	m.IsRequest = true

	s.parseOffset += 1 //type
	s.parseOffset += length
	m.end = s.parseOffset
	m.Size = uint64(m.end - m.start)

	query, err := pgsqlString(s.data[m.start+5:], length-4)
	if err != nil {
		return false, false
	}

	m.Query = query

	m.toExport = true
	detailedf("Simple Query: %s", m.Query)
	return true, true
}

func (pgsql *Pgsql) parseRowDescription(s *PgsqlStream, length int) (bool, bool) {
	// RowDescription
	m := s.message
	m.start = s.parseOffset
	m.IsRequest = false
	m.IsOK = true
	m.toExport = true

	err := pgsqlFieldsParser(s, s.data[s.parseOffset+5:s.parseOffset+length+1])
	if err != nil {
		detailedf("fields parse failed with: %v", err)
		return false, false
	}
	detailedf("Fields: %s", m.Fields)

	s.parseOffset += 1      //type
	s.parseOffset += length //length
	s.parseState = PgsqlGetDataState
	return pgsql.parseMessageData(s)
}

// Parse a list of commands separated by semicolon from the query
func pgsqlQueryParser(query string) []string {
	array := strings.Split(query, ";")

	queries := []string{}

	for _, q := range array {
		qt := strings.TrimSpace(q)
		if len(qt) > 0 {
			queries = append(queries, qt)
		}
	}
	return queries
}

func (pgsql *Pgsql) parseEmptyQueryResponse(s *PgsqlStream) (bool, bool) {
	// EmptyQueryResponse, appears as a response for empty queries
	// substitutes CommandComplete

	m := s.message

	detailedf("EmptyQueryResponse")
	m.start = s.parseOffset
	m.IsOK = true
	m.IsRequest = false
	m.toExport = true
	s.parseOffset += 5 // type + length
	m.end = s.parseOffset
	m.Size = uint64(m.end - m.start)

	return true, true
}

func (pgsql *Pgsql) parseCommandComplete(s *PgsqlStream, length int) (bool, bool) {
	// CommandComplete -> Successful response

	m := s.message
	m.start = s.parseOffset
	m.IsRequest = false
	m.IsOK = true
	m.toExport = true

	s.parseOffset += 1 //type
	name, err := pgsqlString(s.data[s.parseOffset+4:], length-4)
	if err != nil {
		return false, false
	}

	detailedf("CommandComplete length=%d, tag=%s", length, name)

	s.parseOffset += length
	m.end = s.parseOffset
	m.Size = uint64(m.end - m.start)

	return true, true
}

func (pgsql *Pgsql) parseReadyForQuery(s *PgsqlStream, length int) (bool, bool) {

	// ReadyForQuery -> backend ready for a new query cycle
	m := s.message
	m.start = s.parseOffset
	m.Size = uint64(m.end - m.start)

	s.parseOffset += 1 // type
	s.parseOffset += length
	m.end = s.parseOffset

	return true, true
}

func (pgsql *Pgsql) parseErrorResponse(s *PgsqlStream, length int) (bool, bool) {
	// ErrorResponse
	detailedf("ErrorResponse")

	m := s.message
	m.start = s.parseOffset
	m.IsRequest = false
	m.IsError = true
	m.toExport = true

	s.parseOffset += 1 //type
	pgsqlErrorParser(s, s.data[s.parseOffset+4:s.parseOffset+length])

	s.parseOffset += length //length
	m.end = s.parseOffset
	m.Size = uint64(m.end - m.start)

	return true, true
}

func (pgsql *Pgsql) parseExtReq(s *PgsqlStream, length int) (bool, bool) {
	// Ready for query -> Parse for an extended query request
	detailedf("Parse")

	m := s.message
	m.start = s.parseOffset
	m.IsRequest = true

	s.parseOffset += 1 //type
	s.parseOffset += length
	m.end = s.parseOffset
	m.Size = uint64(m.end - m.start)
	m.toExport = true

	query, err := common.ReadString(s.data[m.start+6:])
	if err != nil {
		detailedf("Invalid extended query request")
		return false, false
	}
	m.Query = query
	detailedf("Parse in an extended query request: %s", m.Query)

	// Ignore SET statement
	if strings.HasPrefix(m.Query, "SET ") {
		m.toExport = false
	}
	s.parseState = PgsqlExtendedQueryState
	return pgsql.parseMessageExtendedQuery(s)
}

func (pgsql *Pgsql) parseExtResp(s *PgsqlStream, length int) (bool, bool) {
	// Sync -> Parse completion for an extended query response
	detailedf("ParseCompletion")

	m := s.message
	m.start = s.parseOffset
	m.IsRequest = false
	m.IsOK = true
	m.toExport = true

	s.parseOffset += 1 //type
	s.parseOffset += length
	detailedf("Parse completion in an extended query response")
	s.parseState = PgsqlGetDataState
	return pgsql.parseMessageData(s)
}

func (pgsql *Pgsql) parseSkipMessage(s *PgsqlStream, length int) (bool, bool) {

	// TODO: add info from NoticeResponse in case there are warning messages for a query
	// ignore command
	s.parseOffset += 1 //type
	s.parseOffset += length

	m := s.message
	m.end = s.parseOffset
	m.Size = uint64(m.end - m.start)

	// ok and complete, but ignore
	m.toExport = false
	return true, true
}

func pgsqlFieldsParser(s *PgsqlStream, buf []byte) error {
	m := s.message

	if len(buf) < 2 {
		return errEmptyFieldsBuffer
	}

	// read field count (int16)
	off := 2
	fieldCount := readCount(buf)
	detailedf("Row Description field count=%d", fieldCount)

	fields := []string{}
	fieldsFormat := []byte{}

	for i := 0; i < fieldCount; i++ {
		if len(buf) <= off {
			return errFieldBufferShort
		}

		// read field name (null terminated string)
		fieldName, err := common.ReadString(buf[off:])
		if err != nil {
			return errNoFieldName
		}
		fields = append(fields, fieldName)
		m.NumberOfFields += 1
		off += len(fieldName) + 1

		// read Table OID (int32)
		off += 4

		// read Column Index (int16)
		off += 2

		// read Type OID (int32)
		off += 4

		// read column length (int16)
		off += 2

		// read type modifier (int32)
		off += 4

		// read format (int16)
		format := common.Bytes_Ntohs(buf[off : off+2])
		off += 2
		fieldsFormat = append(fieldsFormat, byte(format))

		detailedf("Field name=%s, format=%d", fieldName, format)
	}

	if off < len(buf) {
		return errFieldBufferBig
	}

	m.Fields = fields
	m.FieldsFormat = fieldsFormat
	if m.NumberOfFields != fieldCount {
		logp.Err("Missing fields from RowDescription. Expected %d. Received %d",
			fieldCount, m.NumberOfFields)
	}
	return nil
}

func pgsqlErrorParser(s *PgsqlStream, buf []byte) {
	m := s.message
	off := 0
	for off < len(buf) {
		// read field type(byte1)
		typ := buf[off]
		if typ == 0 {
			break
		}

		// read field value(string)
		val, err := common.ReadString(buf[off+1:])
		if err != nil {
			logp.Err("Failed to read the column field")
			break
		}
		off += len(val) + 2

		switch typ {
		case 'M':
			m.ErrorInfo = val
		case 'C':
			m.ErrorCode = val
		case 'S':
			m.ErrorSeverity = val
		}
	}
	detailedf("%s %s %s", m.ErrorSeverity, m.ErrorCode, m.ErrorInfo)
}

func (pgsql *Pgsql) parseMessageData(s *PgsqlStream) (bool, bool) {
	detailedf("parseMessageData")

	// The response to queries that return row sets contains:
	// RowDescription
	// zero or more DataRow
	// CommandComplete
	// ReadyForQuery

	m := s.message

	for len(s.data[s.parseOffset:]) > 5 {
		// read type
		typ := byte(s.data[s.parseOffset])

		// read message length
		length := readLength(s.data[s.parseOffset+1:])
		if length < 4 {
			// length should include the size of itself (int32)
			detailedf("Invalid pgsql command length.")
			return false, false
		}
		if len(s.data[s.parseOffset:]) <= length {
			// wait for more
			detailedf("Wait for more data")
			return true, false
		}

		switch typ {
		case 'D':
			err := pgsql.parseDataRow(s, s.data[s.parseOffset+5:s.parseOffset+length+1])
			if err != nil {
				return false, false
			}
			s.parseOffset += 1
			s.parseOffset += length
		case 'C':
			// CommandComplete

			// skip type
			s.parseOffset += 1

			name, err := pgsqlString(s.data[s.parseOffset+4:], length-4)
			if err != nil {
				detailedf("pgsql string invalid")
				return false, false
			}

			detailedf("CommandComplete length=%d, tag=%s", length, name)
			s.parseOffset += length
			m.end = s.parseOffset
			m.Size = uint64(m.end - m.start)
			s.parseState = PgsqlStartState

			detailedf("Rows: %s", m.Rows)

			return true, true
		case '2':
			// Parse completion -> Bind completion for an extended query response

			// skip type
			s.parseOffset += 1
			s.parseOffset += length
			s.parseState = PgsqlStartState
		case 'T':
			return pgsql.parseRowDescription(s, length)
		default:
			// shouldn't happen -> return error
			logp.Warn("Pgsql parser expected data message, but received command of type %v", typ)
			s.parseState = PgsqlStartState
			return false, false
		}
	}

	return true, false
}

func (pgsql *Pgsql) parseDataRow(s *PgsqlStream, buf []byte) error {
	m := s.message

	// read field count (int16)
	off := 2
	fieldCount := readCount(buf)
	detailedf("DataRow field count=%d", fieldCount)

	rows := []string{}
	rowLength := 0

	for i := 0; i < fieldCount; i++ {
		if len(buf) <= off {
			return errFieldBufferShort
		}

		// read column length (int32)
		columnLength := readLength(buf[off:])
		off += 4

		if columnLength > 0 && columnLength > len(buf[off:]) {
			logp.Err("Pgsql invalid column_length=%v, buffer_length=%v, i=%v",
				columnLength, len(buf[off:]), i)
			return errInvalidLength
		}

		// read column value (byten)
		var columnValue []byte
		if m.FieldsFormat[i] == 0 {
			// field value in text format
			if columnLength > 0 {
				columnValue = buf[off : off+columnLength]
				off += columnLength
			}
		}

		if rowLength < pgsql.maxRowLength {
			if rowLength+len(columnValue) > pgsql.maxRowLength {
				columnValue = columnValue[:pgsql.maxRowLength-rowLength]
			}
			rows = append(rows, string(columnValue))
			rowLength += len(columnValue)
		}

		detailedf("Value %s, length=%d, off=%d", string(columnValue), columnLength, off)
	}

	if off < len(buf) {
		return errFieldBufferBig
	}

	m.NumberOfRows += 1
	if len(m.Rows) < pgsql.maxStoreRows {
		m.Rows = append(m.Rows, rows)
	}

	return nil
}

func (pgsql *Pgsql) parseMessageExtendedQuery(s *PgsqlStream) (bool, bool) {
	detailedf("parseMessageExtendedQuery")

	// An extended query request contains:
	// Parse
	// Bind
	// Describe
	// Execute
	// Sync

	m := s.message

	for len(s.data[s.parseOffset:]) >= 5 {
		// read type
		typ := byte(s.data[s.parseOffset])

		// read message length
		length := readLength(s.data[s.parseOffset+1:])
		if length < 4 {
			// length should include the size of itself (int32)
			detailedf("Invalid pgsql command length.")
			return false, false
		}
		if len(s.data[s.parseOffset:]) <= length {
			// wait for more
			detailedf("Wait for more data")
			return true, false
		}

		switch typ {
		case 'B':
			// Parse -> Bind

			// skip type
			s.parseOffset += 1
			s.parseOffset += length
			//TODO: pgsql.parseBind(s)
		case 'D':
			// Bind -> Describe

			// skip type
			s.parseOffset += 1
			s.parseOffset += length
			//TODO: pgsql.parseDescribe(s)
		case 'E':
			// Bind(or Describe) -> Execute

			// skip type
			s.parseOffset += 1
			s.parseOffset += length
			//TODO: pgsql.parseExecute(s)
		case 'S':
			// Execute -> Sync

			// skip type
			s.parseOffset += 1
			s.parseOffset += length
			m.end = s.parseOffset
			m.Size = uint64(m.end - m.start)
			s.parseState = PgsqlStartState

			return true, true
		default:
			// shouldn't happen -> return error
			logp.Warn("Pgsql parser expected extended query message, but received command of type %v", typ)
			s.parseState = PgsqlStartState
			return false, false
		}
	}

	return true, false
}

func isSpecialPgsqlCommand(data []byte) (bool, int, int) {

	if len(data) < 8 {
		// 8 bytes required
		return false, 0, 0
	}

	// read length
	length := readLength(data[0:])

	// read command identifier
	code := int(common.Bytes_Ntohl(data[4:]))

	if length == 16 && code == 80877102 {
		// Cancel Request
		logp.Debug("pgsqldetailed", "Cancel Request, length=%d", length)
		return true, length, CancelRequest
	} else if length == 8 && code == 80877103 {
		// SSL Request
		logp.Debug("pgsqldetailed", "SSL Request, length=%d", length)
		return true, length, SSLRequest
	} else if code == 196608 {
		// Startup Message
		logp.Debug("pgsqldetailed", "Startup Message, length=%d", length)
		return true, length, StartupMessage
	}
	return false, 0, 0
}

// length field in pgsql counts total length of length field + payload, not
// including the message identifier. => Always check buffer size >= length + 1
func readLength(b []byte) int {
	return int(common.Bytes_Ntohl(b))
}

func readCount(b []byte) int {
	return int(common.Bytes_Ntohs(b))
}

func pgsqlString(b []byte, sz int) (string, error) {
	if sz == 0 {
		return "", nil
	}

	if b[sz-1] != 0 {
		return "", errInvalidString
	}

	return string(b[:sz-1]), nil
}

func pgsqlValidType(t byte) bool {
	switch t {
	case '1', '2', '3',
		'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'K',
		'N', 'P', 'Q', 'R', 'S', 'T', 'V', 'W', 'X', 'Z',
		'c', 'd', 'f', 'n', 'p', 's', 't':
		return true
	default:
		return false
	}
}
