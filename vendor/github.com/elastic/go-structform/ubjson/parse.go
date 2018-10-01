// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package ubjson

import (
	"encoding/binary"
	"errors"
	"io"
	"math"

	structform "github.com/elastic/go-structform"
)

type Parser struct {
	visitor    structform.Visitor
	strVisitor structform.StringRefVisitor

	// last fail state
	err error

	// parser state machine
	state      stateStack
	valueState stateStack

	length lengthStack

	buffer  []byte
	buffer0 [64]byte

	// internal parser state
	marker    byte
	valueType structform.BaseType
}

//go:generate stringer -type=stateType
type stateType uint8

//go:generate stringer -type=stateStep
type stateStep uint8

type state struct {
	stateType
	stateStep
}

const (
	stFail stateType = iota
	stNext
	stFixed       // values of fixed size
	stHighPrec    // high precision number
	stString      // string
	stArray       // array
	stArrayDyn    // dynamic array
	stArrayCount  // array with element count
	stArrayTyped  // typed array with element count
	stObject      // object
	stObjectDyn   // dynamic object
	stObjectCount // object with known # of fields
	stObjectTyped // object with all values of same type
)

const (
	stStart stateStep = iota

	// stValue sub-states
	stNil
	stNoop
	stTrue
	stFalse
	stInt8
	stUInt8
	stInt16
	stInt32
	stInt64
	stFloat32
	stFloat64
	stChar

	// variable size primitive value types
	stWithLen

	// array/object states
	stWithType0
	stWithType1
	stCont
	stFieldName
	stFieldNameLen
)

var (
	errUnknownMarker = errors.New("unknown ubjson marker")
	errIncomplete    = errors.New("Incomplete UBJSON input")
	errNegativeLen   = errors.New("negative length encountered")
	errInvalidState  = errors.New("invalid state")
	errMissingArrEnd = errors.New("missing ']'")
	errMissingObjEnd = errors.New("missing '}'")
	errMissingCount  = errors.New("missing count marker")
)

func ParseReader(in io.Reader, vs structform.Visitor) (int64, error) {
	return NewParser(vs).ParseReader(in)
}

func Parse(b []byte, vs structform.Visitor) error {
	return NewParser(vs).Parse(b)
}

func ParseString(str string, vs structform.Visitor) error {
	return NewParser(vs).ParseString(str)
}

func NewParser(vs structform.Visitor) *Parser {
	p := &Parser{}
	p.init(vs)
	return p
}

func (p *Parser) init(vs structform.Visitor) {
	*p = Parser{
		visitor:    vs,
		strVisitor: structform.MakeStringRefVisitor(vs),
	}
	p.buffer = p.buffer0[:0]
	p.length.stack = p.length.stack0[:0]
	p.state.current = state{stNext, stStart}
	p.state.stack = p.state.stack0[:0]
	p.valueState.stack = p.valueState.stack0[:0]
}

func (p *Parser) Parse(b []byte) error {
	p.err = p.feed(b)
	if p.err == nil {
		p.err = p.finalize()
	}
	return p.err
}

func (p *Parser) ParseReader(in io.Reader) (int64, error) {
	n, err := io.Copy(p, in)
	if err == nil {
		err = p.finalize()
	}
	return n, err
}

func (p *Parser) ParseString(s string) error {
	return p.Parse(str2Bytes(s))
}

func (p *Parser) finalize() error {
	for len(p.state.stack) > 0 {
		var err error

		switch p.state.current.stateType {
		case stArrayCount, stArrayTyped:
			if p.length.current != 0 || p.state.current.stateStep != stCont {
				return errMissingArrEnd
			}

			err = p.visitor.OnArrayFinished()
		case stObjectCount, stObjectTyped:
			step := p.state.current.stateStep
			l := p.length.current
			if l != 0 || step != stFieldName {
				return errMissingObjEnd
			}
			err = p.visitor.OnObjectFinished()
		}

		if err != nil {
			return err
		}
		_, err = p.popState()
	}

	st := &p.state.current
	incomplete := len(p.state.stack) > 0 ||
		st.stateStep != stStart ||
		st.stateType != stNext

	if incomplete {
		return errIncomplete
	}
	return nil
}

func (p *Parser) Write(b []byte) (int, error) {
	p.err = p.feed(b)
	if p.err != nil {
		p.state.current = state{stFail, stStart}
		return 0, p.err
	}
	return len(b), nil
}

func (p *Parser) feed(b []byte) error {
	for len(b) > 0 {
		var err error
		n, _, err := p.feedUntil(b)
		if err != nil {
			return err
		}

		b = b[n:]
	}

	return nil
}

func (p *Parser) feedUntil(b []byte) (int, bool, error) {
	var (
		orig = b
		done bool
		err  error
	)

	for {
		b, done, err = p.execStep(b)
		if done || err != nil {
			break
		}

		if len(b) == 0 {
			break
		}
	}
	return len(orig) - len(b), done, err
}

func (p *Parser) execStep(b []byte) ([]byte, bool, error) {
	var (
		err  error
		done bool
	)

	switch p.state.current.stateType {
	case stFail:
		return b, false, p.err
	case stNext:
		b, done, err = p.stepValue(b)
	case stFixed:
		b, done, err = p.stepFixedValue(b)
	case stHighPrec:
		b, done, err = p.stepString(b)
	case stString:
		b, done, err = p.stepString(b)

	case stArray:
		b, err = p.stepArrayInit(b)
	case stArrayDyn:
		b, done, err = p.stepArrayDyn(b)
	case stArrayCount:
		b, done, err = p.stepArrayCount(b)
	case stArrayTyped:
		b, done, err = p.stepArrayTyped(b)

	case stObject:
		b, err = p.stepObjectInit(b)
	case stObjectDyn:
		b, done, err = p.stepObjectDyn(b)
	case stObjectCount:
		b, done, err = p.stepObjectCount(b)
	case stObjectTyped:
		b, done, err = p.stepObjectTyped(b)

	default:
		err = errInvalidState
	}

	if err != nil {
		p.err = err
	}

	return b, done, err
}

func (p *Parser) stepFixedValue(b []byte) ([]byte, bool, error) {
	var (
		tmp  []byte
		err  error
		done bool
	)

	switch p.state.current.stateStep {
	case stNil:
		done, err = true, p.visitor.OnNil()
	case stNoop:

	case stTrue:
		done, err = true, p.visitor.OnBool(true)
	case stFalse:
		done, err = true, p.visitor.OnBool(false)
	case stInt8:
		b, done, err = b[1:], true, p.visitor.OnInt8(int8(b[0]))
	case stUInt8:
		b, done, err = b[1:], true, p.visitor.OnUint8(b[0])
	case stChar:
		b, tmp = p.collect(b, 1)
		if done = tmp != nil; done {
			err = p.visitor.OnByte(tmp[0])
		}
	case stInt16:
		b, tmp = p.collect(b, 2)
		if done = tmp != nil; done {
			err = p.visitor.OnInt16(readInt16(tmp))
		}
	case stInt32:
		b, tmp = p.collect(b, 4)
		if done = tmp != nil; done {
			err = p.visitor.OnInt32(readInt32(tmp))
		}
	case stInt64:
		b, tmp = p.collect(b, 8)
		if done = tmp != nil; done {
			err = p.visitor.OnInt64(readInt64(tmp))
		}
	case stFloat32:
		b, tmp = p.collect(b, 4)
		if done = tmp != nil; done {
			err = p.visitor.OnFloat32(readFloat32(tmp))
		}
	case stFloat64:
		b, tmp = p.collect(b, 8)
		if done = tmp != nil; done {
			err = p.visitor.OnFloat64(readFloat64(tmp))
		}
	default:
		return b, false, err
	}

	if done && err == nil {
		done, err = p.popState()
	}

	return b, done, err
}

func (p *Parser) stepString(b []byte) ([]byte, bool, error) {
	var (
		err  error
		done bool
		st   = &p.state.current
	)

	switch st.stateStep {
	case stStart:
		b, err = p.stepLen(b, st.withStep(stWithLen))
		if !(err == nil && st.stateStep == stWithLen) {
			break
		}
		fallthrough
	case stWithLen:
		L := p.length.current
		if L == 0 {
			done = true
			err = p.visitor.OnString("")
		} else {
			var tmp []byte
			if b, tmp = p.collect(b, int(L)); tmp != nil {
				done = true
				err = p.strVisitor.OnStringRef(tmp)
			}
		}
	}

	if done {
		done, err = p.popLenState()
	}
	return b, done, err
}

func (p *Parser) stepArrayInit(b []byte) ([]byte, error) {
	var (
		err error
		st  = &p.state.current
	)

	switch b[0] {
	case countMarker:
		b, st.stateType = b[1:], stArrayCount
	case typeMarker:
		b, st.stateType = b[1:], stArrayTyped
	default:
		st.stateType = stArrayDyn
		err = p.visitor.OnArrayStart(-1, structform.AnyType)
	}

	return b, err
}

func (p *Parser) stepArrayDyn(b []byte) ([]byte, bool, error) {
	if b[0] == arrEndMarker {
		err := p.visitor.OnArrayFinished()
		done := true
		if err == nil {
			done, err = p.popState()
		}
		return b[1:], done, err
	}

	if st := &p.state.current; st.stateStep == stStart {
		st.stateStep = stCont // ensure continuation state is pushed to stack
		b, _, err := p.stepValue(b)
		return b, false, err
	}
	b, _, err := p.stepValue(b)
	return b, false, err
}

func (p *Parser) stepArrayCount(b []byte) ([]byte, bool, error) {
	var (
		st   = &p.state.current
		step = st.stateStep
	)

	// parse array header
	if step == stStart {
		b, err := p.stepLen(b, st.withStep(stWithLen))
		return b, false, err
	}

	l := int(p.length.current)
	if step == stWithLen {
		p.state.current.stateStep = stCont
		err := p.visitor.OnArrayStart(l, structform.AnyType)
		if err != nil {
			return b, false, err
		}

	}

	if l == 0 {
		err := p.visitor.OnArrayFinished()
		done := true
		if err == nil {
			done, err = p.popLenState()
		}
		return b, done, err
	}

	p.length.current--
	b, _, err := p.stepValue(b)
	return b, false, err
}

func (p *Parser) stepArrayTyped(b []byte) ([]byte, bool, error) {
	step := p.state.current.stateStep

	// parse typed array header
	switch step {
	case stStart, stWithType0, stWithType1:
		b, err := p.stepTypeLenHeader(b, stWithLen)
		return b, false, err
	}

	l := int(p.length.current)
	if step == stWithLen {
		p.state.current.stateStep = stCont
		err := p.visitor.OnArrayStart(l, p.valueType)
		if err != nil {
			return b, false, err
		}
	}

	if l == 0 {
		err := p.visitor.OnArrayFinished()
		done := true
		if err == nil {
			done, err = p.popLenState()
		}
		return b, done, err
	}

	p.length.current--
	vs := p.valueState.current
	p.pushState(vs)
	b, _, err := p.execStep(b)
	return b, false, err
}

func (p *Parser) stepTypeLenHeader(b []byte, cont stateStep) ([]byte, error) {
	st := p.state.current
	step := st.stateStep

	switch step {
	case stStart:
		return p.stepType(b, st.withStep(stWithType0))

	case stWithType0:
		if b[0] != countMarker {
			return b, errMissingCount
		}
		p.state.current = st.withStep(stWithType1)
		return b[1:], nil

	case stWithType1:
		return p.stepLen(b, st.withStep(cont))

	default:
		return b, nil
	}
}

func (p *Parser) stepObjectInit(b []byte) ([]byte, error) {
	var (
		st  = &p.state.current
		err error
	)

	switch b[0] {
	case countMarker:
		b, st.stateType = b[1:], stObjectCount
	case typeMarker:
		b, st.stateType = b[1:], stObjectTyped
	default:
		st.stateType, err = stObjectDyn, p.visitor.OnObjectStart(-1, structform.AnyType)
	}

	return b, err
}

func (p *Parser) stepObjectDyn(b []byte) ([]byte, bool, error) {
	var (
		err  error
		st   = &p.state.current
		step = st.stateStep
	)

	if step == stStart {
		if b[0] == objEndMarker {
			err := p.visitor.OnObjectFinished()
			done := true
			if err == nil {
				done, err = p.popState()
			}
			return b[1:], done, err
		}
	}

	switch step {
	case stStart:
		b, err = p.stepLen(b, st.withStep(stFieldNameLen))
	case stFieldNameLen:
		L := p.length.current
		var tmp []byte
		if b, tmp = p.collect(b, int(L)); tmp != nil {
			p.popLen()
			err = p.strVisitor.OnKeyRef(tmp)
		}
		st.stateStep = stCont
	case stCont:
		st.stateStep = stStart
		b, _, err = p.stepValue(b)
	}

	return b, false, err
}

func (p *Parser) stepObjectCount(b []byte) ([]byte, bool, error) {
	var (
		st   = &p.state.current
		step = st.stateStep
	)

	if step == stStart {
		b, err := p.stepLen(b, st.withStep(stWithLen))
		return b, false, err
	}

	done, b, err := p.stepObjectCountedContent(b, false)
	if done {
		done, err = p.popLenState()
	}
	return b, done, err
}

func (p *Parser) stepObjectTyped(b []byte) ([]byte, bool, error) {
	st := &p.state.current
	step := st.stateStep

	switch step {
	case stStart, stWithType0, stWithType1:
		b, err := p.stepTypeLenHeader(b, stWithLen)
		return b, false, err
	}

	done, b, err := p.stepObjectCountedContent(b, true)
	if done {
		p.valueState.pop()
		done, err = p.popLenState()
	}
	return b, done, err
}

func (p *Parser) stepObjectCountedContent(b []byte, typed bool) (bool, []byte, error) {
	var (
		err  error
		st   = &p.state.current
		step = st.stateStep
		end  = false
	)

	switch step {
	case stWithLen:
		L := p.length.current
		err := p.visitor.OnObjectStart(int(L), structform.AnyType)
		if err != nil {
			return end, b, err
		}

		if L == 0 {
			end = p.length.current == 0
			break
		}

		st.stateStep = stFieldName
		fallthrough

	case stFieldName:
		end = p.length.current == 0
		if end {
			break
		}
		b, err = p.stepLen(b, st.withStep(stFieldNameLen))

	case stFieldNameLen:
		L := p.length.current
		var tmp []byte
		if b, tmp = p.collect(b, int(L)); tmp != nil {
			p.popLen()
			err = p.strVisitor.OnKeyRef(tmp)
		}
		st.stateStep = stCont

	case stCont:
		p.length.current--
		st.stateStep = stFieldName
		// handle object field value
		if typed {
			p.pushState(p.valueState.current)
		} else {
			b, _, err = p.stepValue(b)
		}
	}

	if end {
		err = p.visitor.OnObjectFinished()
	}
	return end, b, err
}

func (p *Parser) stepType(b []byte, cont state) ([]byte, error) {
	marker := b[0]
	b = b[1:]
	p.state.current = cont

	// TODO: analyze marker
	state, err := markerToStartState(marker)
	if err != nil {
		return nil, err
	}
	p.valueState.push(state)
	p.valueType = markerToBaseType(marker)

	return b, nil
}

func (p *Parser) stepLen(b []byte, cont state) ([]byte, error) {
	if p.marker == noMarker {
		p.marker = b[0]
		b = b[1:]
		if len(b) == 0 {
			return nil, nil
		}
	}

	var tmp []byte
	complete := false
	L := int64(-1)

	switch p.marker {
	case int8Marker:
		complete, L, b = true, int64(int8(b[0])), b[1:]
	case uint8Marker:
		complete, L, b = true, int64(b[0]), b[1:]
	case int16Marker:
		if b, tmp = p.collect(b, 2); tmp != nil {
			complete, L = true, int64(readInt16(tmp))
		}
	case int32Marker:
		if b, tmp = p.collect(b, 4); tmp != nil {
			complete, L = true, int64(readInt32(tmp))
		}
	case int64Marker:
		if b, tmp = p.collect(b, 8); tmp != nil {
			complete, L = true, readInt64(tmp)
		}
	}

	if !complete {
		return b, nil
	}

	if L < 0 {
		return nil, errNegativeLen
	}

	p.marker = noMarker
	p.state.current = cont
	p.pushLen(L)
	return b, nil
}

func (p *Parser) collect(b []byte, count int) ([]byte, []byte) {
	if len(p.buffer) > 0 {
		delta := count - len(p.buffer)
		if delta > 0 {
			N := delta
			complete := true
			if N > len(b) {
				complete = false
				N = len(b)
			}

			p.buffer = append(p.buffer, b[:N]...)
			if !complete {
				return nil, nil
			}

			// advance read buffer
			b = b[N:]
		}

		if len(p.buffer) >= count {
			tmp := p.buffer[:count]
			if len(p.buffer) == count {
				p.buffer = p.buffer0[:0]
			} else {
				p.buffer = p.buffer[count:]
			}
			return b, tmp
		}
	}

	if len(b) >= count {
		return b[count:], b[:count]
	}

	p.buffer = append(p.buffer, b...)
	return nil, nil
}

func (p *Parser) stepValue(b []byte) ([]byte, bool, error) {
	state, err := markerToStartState(b[0])
	if err != nil {
		return nil, false, err
	}

	done := true
	switch state.stateStep {
	case stNil:
		b, err = b[1:], p.visitor.OnNil()
	case stNoop:
		done = false
		b, err = b[1:], nil
	case stTrue:
		b, err = b[1:], p.visitor.OnBool(true)
	case stFalse:
		b, err = b[1:], p.visitor.OnBool(false)
	default:
		done = false
		b, err = p.advanceMarker(state, b)
	}

	return b, done, err
}

func (p *Parser) advanceMarker(s state, b []byte) ([]byte, error) {
	p.pushState(s)
	return b[1:], nil
}

func (p *Parser) pushLen(l int64) { p.length.push(l) }
func (p *Parser) popLen()         { p.length.pop() }

func (p *Parser) pushState(next state) { p.state.push(next) }
func (p *Parser) popState() (bool, error) {
	p.state.pop()
	return len(p.state.stack) == 0, nil
}

func (p *Parser) popLenState() (bool, error) {
	p.popLen()
	return p.popState()
}

func readInt16(b []byte) int16 {
	return int16(binary.BigEndian.Uint16(b))
}

func readInt32(b []byte) int32 {
	return int32(binary.BigEndian.Uint32(b))
}

func readInt64(b []byte) int64 {
	return int64(binary.BigEndian.Uint64(b))
}

func readFloat32(b []byte) float32 {
	bits := binary.BigEndian.Uint32(b)
	return math.Float32frombits(bits)
}

func readFloat64(b []byte) float64 {
	bits := binary.BigEndian.Uint64(b)
	return math.Float64frombits(bits)
}

func markerToStartState(marker byte) (state, error) {
	switch marker {
	case nullMarker:
		return state{stFixed, stNil}, nil
	case noopMarker:
		return state{stFixed, stNoop}, nil
	case trueMarker:
		return state{stFixed, stTrue}, nil
	case falseMarker:
		return state{stFixed, stFalse}, nil
	case int8Marker:
		return state{stFixed, stInt8}, nil
	case uint8Marker:
		return state{stFixed, stUInt8}, nil
	case int16Marker:
		return state{stFixed, stInt16}, nil
	case int32Marker:
		return state{stFixed, stInt32}, nil
	case int64Marker:
		return state{stFixed, stInt64}, nil
	case float32Marker:
		return state{stFixed, stFloat32}, nil
	case float64Marker:
		return state{stFixed, stFloat64}, nil
	case highPrecMarker:
		return state{stHighPrec, stStart}, nil
	case charMarker:
		return state{stFixed, stChar}, nil
	case stringMarker:
		return state{stString, stStart}, nil
	case objStartMarker:
		return state{stObject, stStart}, nil
	case arrStartMarker:
		return state{stArray, stStart}, nil
	default:
		return state{stFail, stStart}, errUnknownMarker
	}
}

func markerToBaseType(marker byte) structform.BaseType {
	switch marker {
	case falseMarker, trueMarker:
		return structform.BoolType
	case charMarker:
		return structform.ByteType
	case int8Marker:
		return structform.Int8Type
	case uint8Marker:
		return structform.Uint8Type
	case int16Marker:
		return structform.Int16Type
	case int32Marker:
		return structform.Int32Type
	case int64Marker:
		return structform.Int64Type
	case float32Marker:
		return structform.Float32Type
	case float64Marker:
		return structform.Float64Type
	case highPrecMarker, stringMarker:
		return structform.StringType
	default:
		return structform.AnyType
	}
}

func (st state) withStep(s stateStep) state {
	st.stateStep = s
	return st
}
