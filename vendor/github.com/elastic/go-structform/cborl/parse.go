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

package cborl

import (
	"encoding/binary"
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
	state stateStack

	length lengthStack

	buffer  []byte
	buffer0 [64]byte
}

type state struct {
	major uint8
	minor uint8
}

// additional parser state 'major' types
const (
	stFail          uint8 = 1
	stValue         uint8 = 2
	stLen           uint8 = 3
	stStartX        uint8 = 4
	stIndef         uint8 = 1
	stStartArr      uint8 = majorArr | stStartX
	stStartMap      uint8 = majorMap | stStartX
	stStartIndefArr uint8 = majorArr | stStartX | stIndef
	stStartIndefMap uint8 = majorMap | stStartX | stIndef
	stKey           uint8 = majorMap | 8
	stElem          uint8 = majorMap | 9
)

const (
	stStart uint8 = iota + 1
	stCont
)

func NewParser(vs structform.Visitor) *Parser {
	p := &Parser{}
	p.init(vs)
	return p
}

func ParseReader(in io.Reader, vs structform.Visitor) (int64, error) {
	p := NewParser(vs)
	i, err := io.Copy(p, in)
	return i, err
}

func Parse(b []byte, vs structform.Visitor) error {
	return NewParser(vs).Parse(b)
}

func ParseString(str string, vs structform.Visitor) error {
	return NewParser(vs).ParseString(str)
}

func (p *Parser) init(vs structform.Visitor) {
	*p = Parser{
		visitor:    vs,
		strVisitor: structform.MakeStringRefVisitor(vs),
	}
	p.buffer = p.buffer0[:0]
	p.length.init()
	p.state.init(state{stValue, stStart})
}

func (p *Parser) Write(b []byte) (int, error) {
	p.err = p.feed(b)
	if p.err != nil {
		return 0, p.err
	}
	return len(b), nil
}

func (p *Parser) ParseString(str string) error {
	return p.Parse(str2Bytes(str))
}

func (p *Parser) Parse(b []byte) error {
	return p.feed(b)
}

func (p *Parser) feed(b []byte) error {
	for len(b) > 0 {
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

		// continue parsing if input buffer is not empty, or structure with length
		// fields must be initialized
		// -> structures with length 0 will be reported immediately
		contParse := len(b) != 0 ||
			(p.state.current.major&(stStartX|stIndef)) == stStartX
		if !contParse {
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

	switch p.state.current.major {
	case stFail:
		return b, false, p.err
	case stValue:
		b, done, err = p.stepValue(b)

	case stLen:
		b = p.stepLen(b)
	case majorUint:
		b, done, err = p.stepUint(b)
	case majorNeg:
		b, done, err = p.stepNeg(b)
	case codeSingleFloat:
		b, done, err = p.stepSingleFloat(b)
	case codeDoubleFloat:
		b, done, err = p.stepDoubleFloat(b)

	case majorBytes | stStartX:
		if p.length.current == 0 {
			err = p.visitor.OnArrayStart(0, structform.ByteType)
			if err == nil {
				err = p.visitor.OnArrayFinished()
				p.length.pop()
				if err == nil {
					done, err = p.popState()
				}
			}

			break
		}

		p.state.current.major &= ^stStartX
		if len(b) == 0 {
			break
		}
		fallthrough
	case majorBytes:
		b, done, err = p.stepBytes(b)

	case majorText | stStartX:
		if p.length.current == 0 {
			p.length.pop()
			err = p.visitor.OnString("")
			if err == nil {
				done, err = p.popState()
			}
			break
		}

		p.state.current.major &= ^stStartX
		if len(b) == 0 {
			break
		}
		fallthrough
	case majorText:
		b, done, err = p.stepText(b)

	case stStartArr:
		err = p.visitor.OnArrayStart(int(p.length.current), structform.AnyType)
		if err != nil {
			break
		}
		p.state.pop()
		fallthrough
	case majorArr:
		b, done, err = p.stepArray(b)

	case stStartIndefArr:
		err = p.visitor.OnArrayStart(-1, structform.AnyType)
		if err != nil {
			break
		}
		p.state.pop()
		fallthrough
	case majorArr | stIndef:
		if b[0] == codeBreak {
			b = b[1:]
			err = p.visitor.OnArrayFinished()
			if err == nil {
				done, err = p.popState()
			}
		} else {
			b, done, err = p.stepValue(b)
		}

	case stStartMap:
		err = p.visitor.OnObjectStart(int(p.length.current), structform.AnyType)
		if err != nil {
			break
		}
		p.state.pop()
		fallthrough
	case majorMap:
		b, done, err = p.stepMap(b)
	case stStartIndefMap:
		err = p.visitor.OnObjectStart(-1, structform.AnyType)
		if err != nil {
			break
		}
		p.state.pop()
		fallthrough
	case majorMap | stIndef:
		if b[0] == codeBreak {
			err = p.visitor.OnObjectFinished()
			b = b[1:]
			if err == nil {
				done, err = p.popState()
			}
		} else {
			b, done, err = p.initMapKey(b)
		}
	case stKey | stStartX:
		if p.length.current == 0 {
			err = errEmptyKey
			break
		}

		p.state.current.major &= (^stStartX)
		fallthrough
	case stKey:
		b, done, err = p.stepKey(b)
	case stElem:
		p.state.pop()
		b, done, err = p.stepValue(b)

	default:
		err = errTODO()
	}

	return b, done, err
}

func (p *Parser) popState() (bool, error) {
	p.state.pop()
	return p.onValue()
}

func (p *Parser) onValue() (bool, error) {
	switch p.state.current.major {
	case majorArr:
		p.length.current--
		_, done, err := p.arrayHandleLen()
		return done, err

	case majorMap:
		p.length.current--
		_, done, err := p.mapHandleLen()
		return done, err

	case majorArr | stIndef, majorMap | stIndef:
		return false, nil
	}
	return true, nil
}

func (p *Parser) stepValue(b []byte) ([]byte, bool, error) {
	if len(b) == 0 {
		return b, false, nil
	}

	major := b[0] & majorMask
	switch major {
	case majorUint:
		if b[0] < len8b {
			err := p.visitor.OnUint8(b[0])
			done := false
			if err == nil {
				done, err = p.onValue()
			}
			return b[1:], done, err
		}

		p.state.push(state{major, b[0] & minorMask})
		return b[1:], false, nil

	case majorNeg:
		minor := b[0] & minorMask
		if v := minor; v < len8b {
			err := p.visitor.OnInt8(int8(^v))
			done := false
			if err == nil {
				done, err = p.onValue()
			}
			return b[1:], done, err
		}

		p.state.push(state{major, minor})
		return b[1:], false, nil

	case majorBytes, majorText:
		minor := b[0] & minorMask
		if minor == lenIndef {
			return nil, false, errIndefByteSeq
		} else {
			return p.initByteSeq(major, minor, b[1:])
		}

	case majorArr, majorMap:
		minor := b[0] & minorMask
		return p.initSub(major, minor, b[1:])

	case majorTag:
		return nil, false, errTODO()

	default:
		var (
			err  error
			done bool
		)

		switch b[0] {
		case codeFalse:
			err = p.visitor.OnBool(false)
			if err == nil {
				done, err = p.onValue()
			}
			return b[1:], done, err
		case codeTrue:
			err = p.visitor.OnBool(true)
			if err == nil {
				done, err = p.onValue()
			}
			return b[1:], done, err
		case codeNull, codeUndef:
			err = p.visitor.OnNil()
			if err == nil {
				done, err = p.onValue()
			}
			return b[1:], done, err
		case codeHalfFloat:
			return b[1:], false, errTODO()
		case codeSingleFloat, codeDoubleFloat:
			p.state.push(state{b[0], stStart})
			return b[1:], false, nil
		}
	}
	return nil, false, errInvalidCode
}

func (p *Parser) stepUint(in []byte) (b []byte, done bool, err error) {
	b = in
	switch p.state.current.minor {
	case len8b:
		b, done, err = b[1:], true, p.visitor.OnUint8(b[0])
	case len16b:
		var v uint16
		if b, done, v = p.getUint16(b); done {
			err = p.visitor.OnUint16(v)
		}
	case len32b:
		var v uint32
		if b, done, v = p.getUint32(b); done {
			err = p.visitor.OnUint32(v)
		}
	case len64b:
		var v uint64
		if b, done, v = p.getUint64(b); done {
			err = p.visitor.OnUint64(v)
		}
	}

	if done && err == nil {
		done, err = p.popState()
	}

	return
}

func (p *Parser) stepBytes(b []byte) ([]byte, bool, error) {
	// stream raw bytes via array visitor

	var (
		st  = &p.state.current
		err error
	)

	if st.minor == stStart {
		err = p.visitor.OnArrayStart(int(p.length.current), structform.ByteType)
		if err != nil {
			return nil, false, err
		}
		st.minor = stCont
	}

	L := int(p.length.current)
	done := len(b) >= L
	if !done {
		L = len(b)
		p.length.current -= int64(L)
	}

	for _, c := range b[:L] {
		if err := p.visitor.OnByte(c); err != nil {
			return nil, false, err
		}
	}

	b = b[L:]
	if done {
		err = p.visitor.OnArrayFinished()
		p.length.pop()
		if err == nil {
			done, err = p.popState()
		}
	}
	return b, done, err
}

func (p *Parser) stepText(b []byte) ([]byte, bool, error) {
	b, tmp := p.collect(b, int(p.length.current))
	if tmp == nil {
		return nil, false, nil
	}

	p.length.pop()

	done := true
	err := p.strVisitor.OnStringRef(tmp)
	if err == nil {
		done, err = p.popState()
	}
	return b, done, err
}

func (p *Parser) stepArray(b []byte) ([]byte, bool, error) {
	val, done, err := p.arrayHandleLen()
	if val {
		b, done, err = p.stepValue(b)
	}
	return b, done, err
}

func (p *Parser) arrayHandleLen() (value, done bool, err error) {
	if p.length.current > 0 {
		return true, false, nil
	}

	err = p.visitor.OnArrayFinished()
	if err == nil {
		p.length.pop()
		done, err = p.popState()
	}

	return false, done, err
}

func (p *Parser) stepMap(b []byte) ([]byte, bool, error) {
	kv, done, err := p.mapHandleLen()
	if kv && len(b) > 0 {
		b, done, err = p.initMapKey(b)
	}
	return b, done, err
}

func (p *Parser) mapHandleLen() (kv, done bool, err error) {
	if p.length.current > 0 {
		return true, false, nil
	}

	err = p.visitor.OnObjectFinished()
	if err == nil {
		p.length.pop()
		done, err = p.popState()
	}
	return false, done, err
}

func (p *Parser) initMapKey(b []byte) ([]byte, bool, error) {
	// parse key:
	major := b[0] & majorMask
	if major != majorText {
		return nil, false, errTextKeyRequired
	}

	minor := b[0] & minorMask
	if minor == lenIndef {
		return nil, false, errIndefByteSeq
	}

	return p.initByteSeq(stKey, minor, b[1:])
}

func (p *Parser) stepKey(b []byte) ([]byte, bool, error) {
	b, tmp := p.collect(b, int(p.length.current))
	if tmp == nil {
		return nil, false, nil
	}

	err := p.strVisitor.OnKeyRef(tmp)
	if err == nil {
		p.length.pop()
		p.state.current.major = stElem
	}
	return b, false, err
}

func (p *Parser) initByteSeq(major, minor uint8, b []byte) ([]byte, bool, error) {
	if v := minor; v < len8b {
		p.state.push(state{major | stStartX, stStart})
		p.length.push(int64(v))
		return b, false, nil
	}

	p.state.push(state{major | stStartX, stStart})
	p.state.push(state{stLen, minor})
	return b, false, nil
}

func (p *Parser) initSub(major, minor uint8, b []byte) ([]byte, bool, error) {
	if minor == lenIndef {
		// TODO: replace 2 state pushes with 1 state push + mask removing startX from current state
		p.state.push(state{major | stIndef, stStart})
		p.state.push(state{major | stStartX | stIndef, stStart})
		return b, false, nil
	}

	if v := minor; v < len8b {
		p.state.push(state{major, stStart})
		p.state.push(state{major | stStartX, stStart})
		p.length.push(int64(v))
		return b, false, nil
	}

	p.state.push(state{major, stStart})
	p.state.push(state{major | stStartX, stStart})
	p.state.push(state{stLen, minor})
	return b, false, nil
}

func (p *Parser) stepLen(b []byte) []byte {
	var done bool

	switch p.state.current.minor {
	case len8b:
		p.length.push(int64(b[0]))
		b, done = b[1:], true
	case len16b:
		var v uint16
		if b, done, v = p.getUint16(b); done {
			p.length.push(int64(v))
		}
	case len32b:
		var v uint32
		if b, done, v = p.getUint32(b); done {
			p.length.push(int64(v))
		}

	case len64b:
		var v uint64
		if b, done, v = p.getUint64(b); done {
			p.length.push(int64(v))
		}
	}

	if done {
		p.state.pop()
	}
	return b
}

func (p *Parser) stepNeg(in []byte) (b []byte, done bool, err error) {
	b = in
	switch p.state.current.minor {
	case len8b:
		b, done, err = b[1:], true, p.visitor.OnInt8(int8(^b[0]))
	case len16b:
		var v uint16
		if b, done, v = p.getUint16(b); done {
			err = p.visitor.OnInt16(int16(^v))
		}
	case len32b:
		var v uint32
		if b, done, v = p.getUint32(b); done {
			err = p.visitor.OnInt32(int32(^v))
		}
	case len64b:
		var v uint64
		if b, done, v = p.getUint64(b); done {
			err = p.visitor.OnInt64(int64(^v))
		}
	}

	if done && err == nil {
		done, err = p.popState()
	}
	return
}

func (p *Parser) stepSingleFloat(in []byte) (b []byte, done bool, err error) {
	var tmp uint32
	if b, done, tmp = p.getUint32(in); done {
		err = p.visitor.OnFloat32(math.Float32frombits(tmp))
		if err == nil {
			done, err = p.popState()
		}
	}
	return
}

func (p *Parser) stepDoubleFloat(in []byte) (b []byte, done bool, err error) {
	var tmp uint64
	if b, done, tmp = p.getUint64(in); done {
		err = p.visitor.OnFloat64(math.Float64frombits(tmp))
		if err == nil {
			done, err = p.popState()
		}
	}
	return
}

func (p *Parser) getUint8(b []byte) ([]byte, bool, uint8) {
	return b[1:], true, b[0]
}

func (p *Parser) getUint16(b []byte) ([]byte, bool, uint16) {
	b, tmp := p.collect(b, 2)
	if tmp == nil {
		return nil, false, 0
	}
	return b, true, binary.BigEndian.Uint16(tmp)
}

func (p *Parser) getUint32(b []byte) ([]byte, bool, uint32) {
	b, tmp := p.collect(b, 4)
	if tmp == nil {
		return b, false, 0
	}

	return b, true, binary.BigEndian.Uint32(tmp)
}

func (p *Parser) getUint64(b []byte) ([]byte, bool, uint64) {
	b, tmp := p.collect(b, 8)
	if tmp == nil {
		return nil, false, 0
	}
	return b, true, binary.BigEndian.Uint64(tmp)
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

func numBytes(code uint8) uint8 {
	return 1 << ((code & minorMask) - len8b)
}

func readInt16(b []byte) int16 { return int16(^readUint16(b)) }
func readInt32(b []byte) int32 { return int32(^readUint32(b)) }
func readInt64(b []byte) int64 { return int64(^readUint64(b)) }

func readUint16(b []byte) uint16 { return binary.BigEndian.Uint16(b) }
func readUint32(b []byte) uint32 { return binary.BigEndian.Uint32(b) }
func readUint64(b []byte) uint64 { return binary.BigEndian.Uint64(b) }
