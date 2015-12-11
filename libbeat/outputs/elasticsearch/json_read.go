package elasticsearch

import (
	"errors"

	"github.com/elastic/beats/libbeat/common/streambuf"
)

// SAX like json parser. But instead of relying on callbacks, state machine
// returns raw item plus entity. On top of state machine additional helper methods
// like expectDict, expectArray, nextFieldName and nextInt are available for
// low-level parsing/stepping through a json document.
//
// Due to parser simply stepping through the input buffer, almost no additional
// allocations are required.
type jsonReader struct {
	streambuf.Buffer

	// parser state machine
	states       []entity // state stack for nested arrays/objects
	currentState entity

	// preallocate stack memory for up to 32 nested arrays/objects
	statesBuf [32]entity
}

type entity uint8

var (
	errUnknownChar         = errors.New("unknown character")
	errQuoteMissing        = errors.New("missing closing quote")
	errExpectColon         = errors.New("expected ':' after map key")
	errExpectStringKey     = errors.New("expected string key")
	errUnexpectedComma     = errors.New("unexpected ','")
	errUnexpectedDictClose = errors.New("unexpected '}'")
	errUnexpectedArrClose  = errors.New("unexpected ']'")
	errExpectedDigit       = errors.New("expected a digit")
	errExpectedObject      = errors.New("expected JSON object")
	errExpectedArray       = errors.New("expected JSON array")
	errExpectedFieldName   = errors.New("expected JSON object field name")
	errExpectedInteger     = errors.New("expected integer value")
)

const (
	unknownState entity = iota
	dictStart
	dictEnd
	dictField
	arrStart
	arrEnd
	stringEntity
	mapKeyEntity
	intEntity
	doubleEntity
)

func newJSONReader(in []byte) *jsonReader {
	r := &jsonReader{}
	r.init(in)
	return r
}

func (r *jsonReader) init(in []byte) {
	r.Buffer.Init(in, true)
	r.currentState = unknownState
	r.states = r.statesBuf[:0]
}

var whitespace = []byte(" \t\r\n")

func (r *jsonReader) skipWS() {
	r.IgnoreSymbols(whitespace)
}

func (r *jsonReader) pushState(next entity) {
	if r.currentState != unknownState {
		r.states = append(r.states, r.currentState)
	}
	r.currentState = next
}

func (r *jsonReader) popState() {
	if len(r.states) == 0 {
		r.currentState = unknownState
	} else {
		last := len(r.states) - 1
		r.currentState = r.states[last]
		r.states = r.states[:last]
	}
}

func (r *jsonReader) expectDict() error {
	entity, _, err := r.step()
	if err != nil {
		return err
	}

	if entity != dictStart {
		return r.SetError(errExpectedObject)
	}

	return nil
}

func (r *jsonReader) expectArray() error {
	entity, _, err := r.step()
	if err != nil {
		return err
	}

	if entity != arrStart {
		return r.SetError(errExpectedArray)
	}

	return nil
}

func (r *jsonReader) nextFieldName() (entity, []byte, error) {
	entity, raw, err := r.step()
	if err != nil {
		return entity, raw, err
	}

	if entity != mapKeyEntity && entity != dictEnd {
		return entity, nil, r.SetError(errExpectedFieldName)
	}

	return entity, raw, err
}

func (r *jsonReader) nextInt() (int, error) {
	entity, raw, err := r.step()
	if err != nil {
		return 0, err
	}

	if entity != intEntity {
		return 0, errExpectedInteger
	}

	tmp := streambuf.NewFixed(raw)
	i, err := tmp.AsciiInt(false)
	return int(i), err
}

// ignore type of next element and return raw content.
func (r *jsonReader) ignoreNext() (raw []byte, err error) {
	r.skipWS()

	snapshot := r.Snapshot()
	before := r.Len()

	var ignoreKind func(*jsonReader, entity) error
	ignoreKind = func(r *jsonReader, kind entity) error {

		for {
			entity, _, err := r.step()
			if err != nil {
				return err
			}

			switch entity {
			case kind:
				return nil
			case arrStart:
				return ignoreKind(r, arrEnd)
			case dictStart:
				return ignoreKind(r, dictEnd)
			}
		}
	}

	entity, _, err := r.step()
	if err != nil {
		return nil, err
	}

	switch entity {
	case dictStart:
		err = ignoreKind(r, dictEnd)
	case arrStart:
		err = ignoreKind(r, arrEnd)
	}
	if err != nil {
		return nil, err
	}

	after := r.Len()
	r.Restore(snapshot)

	bytes, _ := r.Collect(before - after)
	return bytes, nil
}

// step continues the JSON parser state machine until next entity has been parsed.
func (r *jsonReader) step() (entity, []byte, error) {
	for r.Len() > 0 {
		r.skipWS()

		c, _ := r.PeekByte()
		if r.currentState == dictStart && c != '"' {
			return unknownState, nil, r.SetError(errExpectStringKey)
		}

		switch c {
		case '{': // start dictionary
			r.Advance(1)
			r.pushState(dictStart)
			return dictStart, nil, nil
		case '}': // end dictionary
			// validate dictionary end (+ allow for trailing comma)
			if r.currentState != dictStart && r.currentState != dictField {
				return unknownState, nil, r.SetError(errUnexpectedDictClose)
			}

			r.Advance(1)
			r.popState()
			return dictEnd, nil, nil
		case '[': // start array
			r.Advance(1)
			r.pushState(arrStart)
			return arrStart, nil, nil
		case ']': // end array
			// validate array end (+ allow for trailing comma)
			if r.currentState != arrStart {
				return unknownState, nil, r.SetError(errUnexpectedArrClose)
			}

			r.Advance(1)
			r.popState()
			return arrEnd, nil, nil
		case ',':
			if r.currentState != arrStart && r.currentState != dictField {
				return unknownState, nil, r.SetError(errUnexpectedComma)
			}

			// next dictionary/array entry
			if r.currentState == dictField {
				r.currentState = dictStart
			}
			r.Advance(1)
		case '"':
			if r.currentState == dictStart {
				r.currentState = dictField
				return r.stepMapKey()
			}
			return r.stepString()
		default:
			// parse number?
			if c == '-' || c == '+' || c == '.' || ('0' <= c && c <= '9') {
				return r.stepNumber()
			}

			err := r.Err()
			if err == nil {
				err = errUnknownChar
				r.SetError(err)
			}
			return unknownState, nil, err
		}
	}

	return unknownState, nil, r.Err()
}

func (r *jsonReader) stepMapKey() (entity, []byte, error) {
	entity, key, err := r.stepString()
	if err != nil {
		return entity, key, err
	}

	r.skipWS()
	c, err := r.ReadByte()
	if err != nil {
		return unknownState, nil, err
	}

	if c != ':' {
		return unknownState, nil, r.SetError(errExpectColon)
	}

	return mapKeyEntity, key, r.Err()
}

func (r *jsonReader) stepString() (entity, []byte, error) {
	start := 1
	for {
		idxQuote := r.IndexByteFrom(start, '"')
		if idxQuote == -1 {
			return unknownState, nil, r.SetError(errQuoteMissing)
		}

		if b, _ := r.PeekByteFrom(idxQuote - 1); b == '\\' { // escaped quote?
			start = idxQuote + 1
			continue
		}

		// found string end
		str, err := r.Collect(idxQuote + 1)
		str = str[1 : len(str)-1]
		return stringEntity, str, err
	}
}

func (r *jsonReader) stepNumber() (entity, []byte, error) {
	snapshot := r.Snapshot()
	lenBefore := r.Len()
	isDouble := false

	if err := r.Err(); err != nil {
		return unknownState, nil, err
	}

	// parse '+', '-' or '.'
	if b, _ := r.PeekByte(); b == '-' || b == '+' {
		r.Advance(1)
	}
	if b, _ := r.PeekByte(); b == '.' {
		r.Advance(1)
		isDouble = true
	}

	// parse digits
	buf, _ := r.CollectWhile(isDigit)
	if len(buf) == 0 {
		return unknownState, nil, r.SetError(errExpectedDigit)
	}

	if !isDouble {
		// parse optional '.'
		if b, _ := r.PeekByte(); b == '.' {
			r.Advance(1)
			isDouble = true

			// parse optional digits
			r.CollectWhile(isDigit)
		}
	}

	lenAfter := r.Len()
	r.Restore(snapshot)
	total := lenBefore - lenAfter - 1
	if total == 0 {
		return unknownState, nil, r.SetError(errExpectedDigit)
	}

	raw, _ := r.Collect(total)
	state := intEntity
	if isDouble {
		state = doubleEntity
	}

	return state, raw, nil
}

func isDigit(c byte) bool {
	return '0' <= c && c <= '9'
}
