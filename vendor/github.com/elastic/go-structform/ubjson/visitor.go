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
	"io"
	"math"
	"strconv"

	structform "github.com/elastic/go-structform"
)

type Visitor struct {
	w       writer
	scratch [16]byte

	length lengthStack
}

type writer struct {
	out io.Writer
}

var _ structform.ExtVisitor = &Visitor{}

const (
	maxUint  = ^uint(0)
	maxInt   = int(maxUint >> 1)
	isUint64 = maxUint > math.MaxUint32
	isInt64  = maxInt > math.MaxInt32
)

func (w writer) write(b []byte) error {
	_, err := w.out.Write(b)
	return err
}

func NewVisitor(out io.Writer) *Visitor {
	v := &Visitor{w: writer{out}}
	v.length.stack = v.length.stack0[:0]
	return v
}

func (vs *Visitor) writeByte(b byte) error {
	vs.scratch[0] = b
	return vs.w.write(vs.scratch[:1])
}

func (vs *Visitor) optionalCount(l int) error {
	vs.length.push(int64(l))

	if l <= 0 {
		// don't add size if array is empty or size is unknown
		return nil
	}

	if err := vs.writeByte(countMarker); err != nil {
		return err
	}
	return vs.writeLen(l)
}

func (vs *Visitor) OnObjectStart(l int, _ structform.BaseType) error {
	// TODO: add typed object support in case of values being passed one by one

	// if number of elements is known, add size
	if err := vs.writeByte(objStartMarker); err != nil {
		return err
	}

	return vs.optionalCount(l)
}

func (vs *Visitor) OnObjectFinished() error {
	if vs.length.pop() <= 0 {
		return vs.writeByte(objEndMarker)
	}
	return nil
}

func (vs *Visitor) OnKey(s string) error {
	return vs.string(str2Bytes(s), false)
}

func (vs *Visitor) OnKeyRef(s []byte) error {
	return vs.string(s, false)
}

func (vs *Visitor) OnArrayStart(l int, t structform.BaseType) error {
	// TODO: optimize array by computing type tag

	if err := vs.writeByte(arrStartMarker); err != nil {
		return err
	}

	// if array size is known, add at least size
	return vs.optionalCount(l)
}

func (vs *Visitor) OnArrayFinished() error {
	if vs.length.pop() <= 0 {
		return vs.writeByte(arrEndMarker)
	}
	return nil
}

func (vs *Visitor) writeLen(l int) error {
	return vs.onInt(l, true)
}

func (vs *Visitor) OnStringRef(s []byte) error {
	if len(s) == 0 {
		return vs.string(nil, true)
	}
	return vs.string(s, true)
}

func (vs *Visitor) OnString(s string) error {
	if len(s) == 0 {
		return vs.string(nil, true)
	}
	return vs.string(str2Bytes(s), true)
}

func (vs *Visitor) string(s []byte, marker bool) error {
	if marker {
		if err := vs.writeByte(stringMarker); err != nil {
			return err
		}
	}

	L := len(s)
	if err := vs.writeLen(L); err != nil {
		return err
	}
	if L == 0 {
		return nil
	}
	return vs.w.write(s)
}

func (vs *Visitor) OnBool(b bool) error {
	if b {
		return vs.writeByte(trueMarker)
	}
	return vs.writeByte(falseMarker)
}

func (vs *Visitor) OnNil() error {
	return vs.writeByte(nullMarker)
}

// int

func (vs *Visitor) OnInt8(i int8) error {
	return vs.int8(i, true)
}

func (vs *Visitor) int8(i int8, marker bool) error {
	if marker {
		if err := vs.writeByte(int8Marker); err != nil {
			return err
		}
	}
	return vs.writeByte(byte(i))
}

func (vs *Visitor) OnInt16(i int16) error {
	if math.MinInt8 <= i && i <= math.MaxInt8 {
		return vs.int8(int8(i), true)
	}
	return vs.int16(i, true)
}

func (vs *Visitor) int16(i int16, marker bool) error {
	if marker {
		if err := vs.writeByte(int16Marker); err != nil {
			return err
		}
	}
	binary.BigEndian.PutUint16(vs.scratch[:2], uint16(i))
	return vs.w.write(vs.scratch[:2])
}

func (vs *Visitor) OnInt32(i int32) error {
	if math.MinInt16 <= i && i <= math.MaxInt16 {
		return vs.OnInt16(int16(i))
	}
	return vs.int32(i, true)
}

func (vs *Visitor) int32(i int32, marker bool) error {
	if marker {
		if err := vs.writeByte(int32Marker); err != nil {
			return err
		}
	}
	binary.BigEndian.PutUint32(vs.scratch[:4], uint32(i))
	return vs.w.write(vs.scratch[:4])
}

func (vs *Visitor) OnInt64(i int64) error {
	if math.MinInt32 <= i && i <= math.MaxInt32 {
		return vs.OnInt32(int32(i))
	}
	return vs.int64(i, true)
}

func (vs *Visitor) int64(i int64, marker bool) error {
	if marker {
		if err := vs.writeByte(int64Marker); err != nil {
			return err
		}
	}
	binary.BigEndian.PutUint64(vs.scratch[:8], uint64(i))
	return vs.w.write(vs.scratch[:8])
}

func (vs *Visitor) OnInt(i int) error {
	return vs.onInt(i, true)
}

func (vs *Visitor) onInt(i int, marker bool) error {
	switch {
	case math.MinInt8 <= i && i <= math.MaxInt8:
		return vs.int8(int8(i), marker)
	case 0 <= i && i <= math.MaxUint8:
		return vs.uint8(uint8(i), marker)
	case math.MinInt16 <= i && i <= math.MaxInt16:
		return vs.int16(int16(i), marker)
	case math.MinInt32 <= i && i <= math.MaxInt32:
		return vs.int32(int32(i), marker)
	default:
		return vs.int64(int64(i), marker)
	}
}

func (vs *Visitor) OnByte(b byte) error {
	vs.scratch[0] = charMarker
	vs.scratch[1] = b
	return vs.w.write(vs.scratch[:2])
}

// uint

func (vs *Visitor) OnUint8(u uint8) error {
	return vs.uint8(u, true)
}

func (vs *Visitor) uint8(u uint8, marker bool) error {
	if marker {
		vs.scratch[0], vs.scratch[1] = uint8Marker, u
		return vs.w.write(vs.scratch[:2])
	}
	return vs.writeByte(u)
}

func (vs *Visitor) OnUint16(u uint16) error {
	return vs.OnUint64(uint64(u))
}

func (vs *Visitor) OnUint32(u uint32) error {
	return vs.OnUint64(uint64(u))
}

func (vs *Visitor) OnUint64(u uint64) error {
	return vs.uint64(u, uintType(u), true)
}

func (vs *Visitor) uint64(u uint64, t byte, marker bool) error {
	switch t {
	case int8Marker:
		return vs.int8(int8(u), marker)
	case uint8Marker:
		return vs.uint8(uint8(u), marker)
	case int16Marker:
		return vs.int16(int16(u), marker)
	case int32Marker:
		return vs.int32(int32(u), marker)
	case int64Marker:
		return vs.int64(int64(u), marker)
	default:
		return vs.uint64HighPrec(u, marker)
	}
}

func (vs *Visitor) uint64HighPrec(u uint64, marker bool) error {
	if marker {
		if err := vs.writeByte(highPrecMarker); err != nil {
			return err
		}
	}

	b := strconv.AppendUint(vs.scratch[:0], u, 10)
	if err := vs.writeLen(len(b)); err != nil {
		return err
	}
	return vs.w.write(b)
}

func (vs *Visitor) OnUint(u uint) error {
	return vs.OnUint64(uint64(u))
}

// float

func (vs *Visitor) OnFloat32(f float32) error {
	return vs.float32(f, true)
}

func (vs *Visitor) float32(f float32, marker bool) error {
	if marker {
		if err := vs.writeByte(float32Marker); err != nil {
			return err
		}
	}

	bits := math.Float32bits(f)
	binary.BigEndian.PutUint32(vs.scratch[:4], bits)
	return vs.w.write(vs.scratch[:4])
}

func (vs *Visitor) OnFloat64(f float64) error {
	return vs.float64(f, true)
}

func (vs *Visitor) float64(f float64, marker bool) error {
	if marker {
		if err := vs.writeByte(float64Marker); err != nil {
			return err
		}
	}

	bits := math.Float64bits(f)
	binary.BigEndian.PutUint64(vs.scratch[:8], bits)
	return vs.w.write(vs.scratch[:8])
}

// specialize array encoders

func (vs *Visitor) onTypedStruct(s, t byte, count int) error {
	vs.scratch[0] = s
	vs.scratch[1] = typeMarker
	vs.scratch[2] = t
	vs.scratch[3] = countMarker

	if err := vs.w.write(vs.scratch[:4]); err != nil {
		return err
	}
	return vs.writeLen(count)

}

func (vs *Visitor) onArray(t byte, count int) error {
	return vs.onTypedStruct(arrStartMarker, t, count)
}

func (vs *Visitor) OnStringArray(a []string) error {
	if done, err := vs.onEmptyArray(len(a)); done {
		return err
	}

	if err := vs.onArray(stringMarker, len(a)); err != nil {
		return err
	}
	for _, v := range a {
		if err := vs.string(str2Bytes(v), false); err != nil {
			return err
		}
	}
	return nil
}

func (vs *Visitor) OnBoolArray(a []bool) error {
	// no special encoding for boolean arrays => fall back to per element encoding

	if err := vs.OnArrayStart(len(a), structform.AnyType); err != nil {
		return err
	}
	for _, b := range a {
		if err := vs.OnBool(b); err != nil {
			return err
		}
	}
	return nil
}

func (vs *Visitor) OnInt8Array(a []int8) error {
	if done, err := vs.onEmptyArray(len(a)); done {
		return err
	}

	if err := vs.onArray(int8Marker, len(a)); err != nil {
		return err
	}
	for _, v := range a {
		if err := vs.int8(v, false); err != nil {
			return err
		}
	}
	return nil
}

func (vs *Visitor) OnInt16Array(a []int16) error {
	if done, err := vs.onEmptyArray(len(a)); done {
		return err
	}

	if err := vs.onArray(int16Marker, len(a)); err != nil {
		return err
	}
	for _, v := range a {
		if err := vs.int16(v, false); err != nil {
			return err
		}
	}
	return nil
}

func (vs *Visitor) OnInt32Array(a []int32) error {
	if done, err := vs.onEmptyArray(len(a)); done {
		return err
	}

	if err := vs.onArray(int32Marker, len(a)); err != nil {
		return err
	}
	for _, v := range a {
		if err := vs.int32(v, false); err != nil {
			return err
		}
	}
	return nil
}

func (vs *Visitor) OnInt64Array(a []int64) error {
	if done, err := vs.onEmptyArray(len(a)); done {
		return err
	}

	if err := vs.onArray(int64Marker, len(a)); err != nil {
		return err
	}
	for _, v := range a {
		if err := vs.int64(v, false); err != nil {
			return err
		}
	}
	return nil
}

func (vs *Visitor) OnIntArray(a []int) error {
	if done, err := vs.onEmptyArray(len(a)); done {
		return err
	}

	marker := int32Marker
	if isInt64 {
		marker = int64Marker
	}

	if err := vs.onArray(marker, len(a)); err != nil {
		return err
	}
	for _, v := range a {
		var err error

		if isInt64 {
			err = vs.int64(int64(v), false)
		} else {
			err = vs.int32(int32(v), false)
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func (vs *Visitor) OnBytes(a []byte) error {
	if done, err := vs.onEmptyArray(len(a)); done {
		return err
	}

	if err := vs.onArray(uint8Marker, len(a)); err != nil {
		return err
	}

	for _, v := range a {
		if err := vs.uint8(v, false); err != nil {
			return err
		}
	}
	return nil

}

func (vs *Visitor) OnUint8Array(a []uint8) error {
	if done, err := vs.onEmptyArray(len(a)); done {
		return err
	}

	if err := vs.onArray(uint8Marker, len(a)); err != nil {
		return err
	}
	for _, v := range a {
		if err := vs.uint8(v, false); err != nil {
			return err
		}
	}
	return nil
}

func (vs *Visitor) OnUint16Array(a []uint16) error {
	if done, err := vs.onEmptyArray(len(a)); done {
		return err
	}

	// find type:
	minT := int8Marker
	for _, v := range a {
		minT = maxNumType(minT, uintType(uint64(v)))
	}

	// serialize array
	if err := vs.onArray(minT, len(a)); err != nil {
		return err
	}
	for _, v := range a {
		if err := vs.uint64(uint64(v), minT, false); err != nil {
			return err
		}
	}
	return nil
}

func (vs *Visitor) OnUint32Array(a []uint32) error {
	if done, err := vs.onEmptyArray(len(a)); done {
		return err
	}

	// find type:
	minT := int8Marker
	for _, v := range a {
		minT = maxNumType(minT, uintType(uint64(v)))
	}

	// serialize array
	if err := vs.onArray(minT, len(a)); err != nil {
		return err
	}
	for _, v := range a {
		if err := vs.uint64(uint64(v), minT, false); err != nil {
			return err
		}
	}
	return nil
}

func (vs *Visitor) OnUint64Array(a []uint64) error {
	if done, err := vs.onEmptyArray(len(a)); done {
		return err
	}

	// find type:
	minT := int8Marker
	for _, v := range a {
		minT = maxNumType(minT, uintType(v))
	}

	// serialize array
	if err := vs.onArray(minT, len(a)); err != nil {
		return err
	}
	for _, v := range a {
		if err := vs.uint64(v, minT, false); err != nil {
			return err
		}
	}
	return nil
}

func (vs *Visitor) OnUintArray(a []uint) error {
	if done, err := vs.onEmptyArray(len(a)); done {
		return err
	}

	// find type:
	minT := int8Marker
	for _, v := range a {
		minT = maxNumType(minT, uintType(uint64(v)))
	}

	// serialize array
	if err := vs.onArray(minT, len(a)); err != nil {
		return err
	}
	for _, v := range a {
		if err := vs.uint64(uint64(v), minT, false); err != nil {
			return err
		}
	}
	return nil
}

func (vs *Visitor) OnFloat32Array(a []float32) error {
	if done, err := vs.onEmptyArray(len(a)); done {
		return err
	}

	if err := vs.onArray(float32Marker, len(a)); err != nil {
		return err
	}
	for _, v := range a {
		if err := vs.float32(v, false); err != nil {
			return err
		}
	}
	return nil
}

func (vs *Visitor) OnFloat64Array(a []float64) error {
	if done, err := vs.onEmptyArray(len(a)); done {
		return err
	}

	if err := vs.onArray(float64Marker, len(a)); err != nil {
		return err
	}
	for _, v := range a {
		if err := vs.float64(v, false); err != nil {
			return err
		}
	}
	return nil
}

func (vs *Visitor) OnStringObject(m map[string]string) error {
	if done, err := vs.onEmptyObject(len(m)); done {
		return err
	}
	if err := vs.onObject(stringMarker, len(m)); err != nil {
		return err
	}
	for k, v := range m {
		if err := vs.string(str2Bytes(k), false); err != nil {
			return err
		}
		if err := vs.string(str2Bytes(v), false); err != nil {
			return err
		}
	}

	return nil
}

func (vs *Visitor) OnBoolObject(m map[string]bool) error {
	if done, err := vs.onEmptyObject(len(m)); done {
		return err
	}
	if err := vs.optionalCount(len(m)); err != nil {
		return err
	}

	for k, v := range m {
		if err := vs.string(str2Bytes(k), false); err != nil {
			return err
		}
		if err := vs.OnBool(v); err != nil {
			return err
		}
	}

	return nil
}

func (vs *Visitor) OnInt8Object(m map[string]int8) error {
	if done, err := vs.onEmptyObject(len(m)); done {
		return err
	}
	if err := vs.onObject(int8Marker, len(m)); err != nil {
		return err
	}
	for k, v := range m {
		if err := vs.string(str2Bytes(k), false); err != nil {
			return err
		}
		if err := vs.int8(v, false); err != nil {
			return err
		}
	}

	return nil
}

func (vs *Visitor) OnInt16Object(m map[string]int16) error {
	if done, err := vs.onEmptyObject(len(m)); done {
		return err
	}
	if err := vs.onObject(int16Marker, len(m)); err != nil {
		return err
	}
	for k, v := range m {
		if err := vs.string(str2Bytes(k), false); err != nil {
			return err
		}
		if err := vs.int16(v, false); err != nil {
			return err
		}
	}

	return nil
}

func (vs *Visitor) OnInt32Object(m map[string]int32) error {
	if done, err := vs.onEmptyObject(len(m)); done {
		return err
	}
	if err := vs.onObject(int32Marker, len(m)); err != nil {
		return err
	}
	for k, v := range m {
		if err := vs.string(str2Bytes(k), false); err != nil {
			return err
		}
		if err := vs.int32(v, false); err != nil {
			return err
		}
	}

	return nil
}

func (vs *Visitor) OnInt64Object(m map[string]int64) error {
	if done, err := vs.onEmptyObject(len(m)); done {
		return err
	}

	if err := vs.onObject(int64Marker, len(m)); err != nil {
		return err
	}
	for k, v := range m {
		if err := vs.string(str2Bytes(k), false); err != nil {
			return err
		}
		if err := vs.int64(v, false); err != nil {
			return err
		}
	}

	return nil
}

func (vs *Visitor) OnIntObject(m map[string]int) error {
	if done, err := vs.onEmptyObject(len(m)); done {
		return err
	}

	marker := int32Marker
	if isInt64 {
		marker = int64Marker
	}

	if err := vs.onObject(marker, len(m)); err != nil {
		return err
	}
	for k, v := range m {
		var err error
		if err = vs.string(str2Bytes(k), false); err != nil {
			return err
		}
		if isInt64 {
			err = vs.int64(int64(v), false)
		} else {
			err = vs.int32(int32(v), false)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (vs *Visitor) OnUint8Object(m map[string]uint8) error {
	if done, err := vs.onEmptyObject(len(m)); done {
		return err
	}

	if err := vs.onObject(uint8Marker, len(m)); err != nil {
		return err
	}
	for k, v := range m {
		if err := vs.string(str2Bytes(k), false); err != nil {
			return err
		}
		if err := vs.uint8(v, false); err != nil {
			return err
		}
	}

	return nil
}

func (vs *Visitor) OnUint16Object(m map[string]uint16) error {
	if done, err := vs.onEmptyObject(len(m)); done {
		return err
	}

	// find type:
	minT := int8Marker
	for _, v := range m {
		minT = maxNumType(minT, uintType(uint64(v)))
	}

	//serialize object
	if err := vs.onObject(minT, len(m)); err != nil {
		return err
	}
	for k, v := range m {
		if err := vs.string(str2Bytes(k), false); err != nil {
			return err
		}
		if err := vs.uint64(uint64(v), minT, false); err != nil {
			return err
		}
	}

	return nil
}

func (vs *Visitor) OnUint32Object(m map[string]uint32) error {
	if done, err := vs.onEmptyObject(len(m)); done {
		return err
	}

	// find type:
	minT := int8Marker
	for _, v := range m {
		minT = maxNumType(minT, uintType(uint64(v)))
	}

	//serialize object
	if err := vs.onObject(minT, len(m)); err != nil {
		return err
	}
	for k, v := range m {
		if err := vs.string(str2Bytes(k), false); err != nil {
			return err
		}
		if err := vs.uint64(uint64(v), minT, false); err != nil {
			return err
		}
	}

	return nil
}

func (vs *Visitor) OnUint64Object(m map[string]uint64) error {
	if done, err := vs.onEmptyObject(len(m)); done {
		return err
	}

	// find type:
	minT := int8Marker
	for _, v := range m {
		minT = maxNumType(minT, uintType(uint64(v)))
	}

	//serialize object
	if err := vs.onObject(minT, len(m)); err != nil {
		return err
	}
	for k, v := range m {
		if err := vs.string(str2Bytes(k), false); err != nil {
			return err
		}
		if err := vs.uint64(uint64(v), minT, false); err != nil {
			return err
		}
	}

	return nil
}

func (vs *Visitor) OnUintObject(m map[string]uint) error {
	if done, err := vs.onEmptyObject(len(m)); done {
		return err
	}

	// find type:
	minT := int8Marker
	for _, v := range m {
		minT = maxNumType(minT, uintType(uint64(v)))
	}

	//serialize object
	if err := vs.onObject(minT, len(m)); err != nil {
		return err
	}
	for k, v := range m {
		if err := vs.string(str2Bytes(k), false); err != nil {
			return err
		}
		if err := vs.uint64(uint64(v), minT, false); err != nil {
			return err
		}
	}

	return nil
}

func (vs *Visitor) OnFloat32Object(m map[string]float32) error {
	if done, err := vs.onEmptyObject(len(m)); done {
		return err
	}
	if err := vs.onObject(float32Marker, len(m)); err != nil {
		return err
	}
	for k, v := range m {
		if err := vs.string(str2Bytes(k), false); err != nil {
			return err
		}
		if err := vs.float32(v, false); err != nil {
			return err
		}
	}

	return nil
}

func (vs *Visitor) OnFloat64Object(m map[string]float64) error {
	if done, err := vs.onEmptyObject(len(m)); done {
		return err
	}

	if err := vs.onObject(float64Marker, len(m)); err != nil {
		return err
	}
	for k, v := range m {
		if err := vs.string(str2Bytes(k), false); err != nil {
			return err
		}
		if err := vs.float64(v, false); err != nil {
			return err
		}
	}

	return nil
}

func (vs *Visitor) onEmptyArray(l int) (bool, error) {
	if l > 0 {
		return false, nil
	}

	vs.scratch[0], vs.scratch[1] = arrStartMarker, arrEndMarker
	return true, vs.w.write(vs.scratch[:2])
}

func (vs *Visitor) onEmptyObject(l int) (bool, error) {
	if l > 0 {
		return false, nil
	}

	vs.scratch[0], vs.scratch[1] = objStartMarker, objEndMarker
	return true, vs.w.write(vs.scratch[:2])
}

func (vs *Visitor) onObject(marker byte, count int) error {
	return vs.onTypedStruct(objStartMarker, marker, count)
}

func maxNumType(a, b byte) byte {
	switch {
	case a == highPrecMarker || b == highPrecMarker:
		return highPrecMarker
	case a == int64Marker || b == int64Marker:
		return int64Marker
	case a == int32Marker || b == int32Marker:
		return int32Marker
	case a == int16Marker || b == int16Marker:
		return int16Marker
	case a == uint8Marker || b == uint8Marker:
		return uint8Marker
	default:
		return int8Marker
	}
}

func uintType(u uint64) byte {
	switch {
	case u <= math.MaxInt8:
		return int8Marker
	case u <= math.MaxUint8:
		return uint8Marker
	case u <= math.MaxInt16:
		return int16Marker
	case u <= math.MaxInt32:
		return int32Marker
	case u <= math.MaxInt64:
		return int64Marker
	default:
		return highPrecMarker
	}
}
