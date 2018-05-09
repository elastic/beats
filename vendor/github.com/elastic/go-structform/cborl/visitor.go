package cborl

import (
	"encoding/binary"
	"io"
	"math"

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

func (vs *Visitor) OnObjectStart(len int, baseType structform.BaseType) error {
	if err := vs.optLen(majorMap, len); err != nil {
		return err
	}

	vs.length.push(int64(len))
	return nil
}

func (vs *Visitor) OnObjectFinished() error {
	if vs.length.pop() < 0 {
		return vs.writeByte(codeBreak)
	}
	return nil
}

func (vs *Visitor) OnKey(s string) error {
	return vs.string(str2Bytes(s))
}

func (vs *Visitor) OnKeyRef(s []byte) error {
	return vs.string(s)
}

func (vs *Visitor) OnArrayStart(len int, baseType structform.BaseType) error {
	if err := vs.optLen(majorArr, len); err != nil {
		return err
	}

	vs.length.push(int64(len))
	return nil
}

func (vs *Visitor) OnArrayFinished() error {
	if vs.length.pop() < 0 {
		return vs.writeByte(codeBreak)
	}
	return nil
}

func (vs *Visitor) OnNil() error {
	return vs.writeByte(codeNull)
}

func (vs *Visitor) OnBool(b bool) error {
	if b {
		return vs.writeByte(codeTrue)
	}
	return vs.writeByte(codeFalse)
}

func (vs *Visitor) OnString(s string) error {
	return vs.string(str2Bytes(s))
}

func (vs *Visitor) OnStringRef(s []byte) error {
	return vs.string(s)
}

func (vs *Visitor) OnInt8(i int8) error {
	return vs.int8(i)
}

func (vs *Visitor) OnInt16(i int16) error {
	return vs.int16(i)
}

func (vs *Visitor) OnInt32(i int32) error {
	return vs.int32(i)
}

func (vs *Visitor) OnInt64(i int64) error {
	return vs.int64(i)
}

func (vs *Visitor) OnInt(i int) error {
	return vs.int64(int64(i))
}

func (vs *Visitor) OnByte(b byte) error {
	return vs.uint8(majorUint, b)
}

func (vs *Visitor) OnUint8(u uint8) error {
	return vs.uint8(majorUint, u)
}

func (vs *Visitor) OnUint16(u uint16) error {
	return vs.uint16(majorUint, u)
}

func (vs *Visitor) OnUint32(u uint32) error {
	return vs.uint32(majorUint, u)
}

func (vs *Visitor) OnUint64(u uint64) error {
	return vs.uint64(majorUint, u)
}

func (vs *Visitor) OnUint(u uint) error {
	return vs.uint64(majorUint, uint64(u))
}

func (vs *Visitor) OnFloat32(f float32) error {
	b := math.Float32bits(f)
	vs.scratch[0] = codeSingleFloat
	binary.BigEndian.PutUint32(vs.scratch[1:5], b)
	return vs.w.write(vs.scratch[:5])
}

func (vs *Visitor) OnFloat64(f float64) error {
	b := math.Float64bits(f)
	vs.scratch[0] = codeDoubleFloat
	binary.BigEndian.PutUint64(vs.scratch[1:9], b)
	return vs.w.write(vs.scratch[:9])
}

func (vs *Visitor) OnBoolArray(a []bool) error {
	if err := vs.arrLen(len(a)); err != nil {
		return nil
	}
	for _, v := range a {
		if err := vs.OnBool(v); err != nil {
			return err
		}
	}

	return nil
}

func (vs *Visitor) OnStringArray(a []string) error {
	if err := vs.arrLen(len(a)); err != nil {
		return nil
	}
	for _, v := range a {
		if err := vs.OnString(v); err != nil {
			return err
		}
	}
	return nil
}

func (vs *Visitor) OnInt8Array(a []int8) error {
	if err := vs.arrLen(len(a)); err != nil {
		return nil
	}
	for _, v := range a {
		if err := vs.OnInt8(v); err != nil {
			return err
		}
	}
	return nil
}

func (vs *Visitor) OnInt16Array(a []int16) error {
	if err := vs.arrLen(len(a)); err != nil {
		return nil
	}
	for _, v := range a {
		if err := vs.OnInt16(v); err != nil {
			return err
		}
	}
	return nil
}

func (vs *Visitor) OnInt32Array(a []int32) error {
	if err := vs.arrLen(len(a)); err != nil {
		return nil
	}
	for _, v := range a {
		if err := vs.OnInt32(v); err != nil {
			return err
		}
	}
	return nil
}

func (vs *Visitor) OnInt64Array(a []int64) error {
	if err := vs.arrLen(len(a)); err != nil {
		return nil
	}
	for _, v := range a {
		if err := vs.OnInt64(v); err != nil {
			return err
		}
	}
	return nil
}

func (vs *Visitor) OnIntArray(a []int) error {
	if err := vs.arrLen(len(a)); err != nil {
		return nil
	}
	for _, v := range a {
		if err := vs.OnInt(v); err != nil {
			return err
		}
	}
	return nil
}

func (vs *Visitor) OnBytes(a []byte) error {
	return vs.bytes(majorBytes, a)
}

func (vs *Visitor) OnUint8Array(a []uint8) error {
	return vs.bytes(majorBytes, a)
}

func (vs *Visitor) OnUint16Array(a []uint16) error {
	if err := vs.arrLen(len(a)); err != nil {
		return nil
	}
	for _, v := range a {
		if err := vs.OnUint16(v); err != nil {
			return err
		}
	}
	return nil
}

func (vs *Visitor) OnUint32Array(a []uint32) error {
	if err := vs.arrLen(len(a)); err != nil {
		return nil
	}
	for _, v := range a {
		if err := vs.OnUint32(v); err != nil {
			return err
		}
	}
	return nil
}

func (vs *Visitor) OnUint64Array(a []uint64) error {
	if err := vs.arrLen(len(a)); err != nil {
		return nil
	}
	for _, v := range a {
		if err := vs.OnUint64(v); err != nil {
			return err
		}
	}
	return nil
}

func (vs *Visitor) OnUintArray(a []uint) error {
	if err := vs.arrLen(len(a)); err != nil {
		return nil
	}
	for _, v := range a {
		if err := vs.OnUint(v); err != nil {
			return err
		}
	}
	return nil
}

func (vs *Visitor) OnFloat32Array(a []float32) error {
	if err := vs.arrLen(len(a)); err != nil {
		return nil
	}
	for _, v := range a {
		if err := vs.OnFloat32(v); err != nil {
			return err
		}
	}
	return nil
}

func (vs *Visitor) OnFloat64Array(a []float64) error {
	if err := vs.arrLen(len(a)); err != nil {
		return nil
	}
	for _, v := range a {
		if err := vs.OnFloat64(v); err != nil {
			return err
		}
	}
	return nil
}

func (vs *Visitor) string(s []byte) error {
	return vs.bytes(majorText, s)
}

func (vs *Visitor) bytes(major uint8, buf []byte) error {
	if err := vs.uint64(major, uint64(len(buf))); err != nil {
		return err
	}
	return vs.w.write(buf)
}

func (vs *Visitor) arrLen(len int) error {
	return vs.uint64(majorArr, uint64(len))
}

func (vs *Visitor) optLen(major uint8, len int) error {
	if len < 0 {
		return vs.writeByte(major | lenIndef)
	}
	return vs.uint64(major, uint64(len))
}

func (vs *Visitor) int8(v int8) error {
	if v < 0 {
		return vs.uint8(majorNeg, ^uint8(v))
	}
	return vs.uint8(majorUint, uint8(v))
}

func (vs *Visitor) int16(v int16) error {
	if v < 0 {
		return vs.uint16(majorNeg, ^uint16(v))
	}
	return vs.uint16(majorUint, uint16(v))
}

func (vs *Visitor) int32(v int32) error {
	if v < 0 {
		return vs.uint32(majorNeg, ^uint32(v))
	}
	return vs.uint32(majorUint, uint32(v))
}

func (vs *Visitor) int64(v int64) error {
	if v < 0 {
		return vs.uint64(majorNeg, ^uint64(v))
	}
	return vs.uint64(majorUint, uint64(v))
}

func (vs *Visitor) uint8(major uint8, v uint8) error {
	if v < len8b {
		return vs.writeByte(major | v)
	}

	vs.scratch[0], vs.scratch[1] = major|len8b, v
	return vs.w.write(vs.scratch[:2])
}

func (vs *Visitor) uint16(major uint8, v uint16) error {
	switch {
	case v < uint16(len8b):
		return vs.writeByte(major | uint8(v))
	case v <= math.MaxUint8:
		vs.scratch[0], vs.scratch[1] = major|len8b, uint8(v)
		return vs.w.write(vs.scratch[:2])
	default:
		vs.scratch[0] = major | len16b
		binary.BigEndian.PutUint16(vs.scratch[1:3], v)
		return vs.w.write(vs.scratch[:3])
	}
}

func (vs *Visitor) uint32(major uint8, v uint32) error {
	switch {
	case v < uint32(len8b):
		return vs.writeByte(major | uint8(v))
	case v <= math.MaxUint8:
		vs.scratch[0], vs.scratch[1] = major|len8b, uint8(v)
		return vs.w.write(vs.scratch[:2])
	case v <= math.MaxUint16:
		vs.scratch[0] = major | len16b
		binary.BigEndian.PutUint16(vs.scratch[1:3], uint16(v))
		return vs.w.write(vs.scratch[:3])
	default:
		vs.scratch[0] = major | len32b
		binary.BigEndian.PutUint32(vs.scratch[1:5], v)
		return vs.w.write(vs.scratch[:5])
	}
}

func (vs *Visitor) uint64(major uint8, v uint64) error {
	switch {
	case v < uint64(len8b):
		return vs.writeByte(major | uint8(v))
	case v <= math.MaxUint8:
		vs.scratch[0], vs.scratch[1] = major|len8b, uint8(v)
		return vs.w.write(vs.scratch[:2])
	case v <= math.MaxUint16:
		vs.scratch[0] = major | len16b
		binary.BigEndian.PutUint16(vs.scratch[1:3], uint16(v))
		return vs.w.write(vs.scratch[:3])
	case v <= math.MaxUint32:
		vs.scratch[0] = major | len32b
		binary.BigEndian.PutUint32(vs.scratch[1:5], uint32(v))
		return vs.w.write(vs.scratch[:5])
	default:
		vs.scratch[0] = major | len64b
		binary.BigEndian.PutUint64(vs.scratch[1:9], uint64(v))
		return vs.w.write(vs.scratch[:9])
	}
}
