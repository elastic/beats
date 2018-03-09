package visitors

import structform "github.com/elastic/go-structform"

type emptyVisitor struct {
}

func NilVisitor() structform.Visitor {
	return (*emptyVisitor)(nil)
}

func (e *emptyVisitor) OnObjectStart(len int, baseType structform.BaseType) error {
	return nil
}

func (e *emptyVisitor) OnObjectFinished() error {
	return nil
}

func (e *emptyVisitor) OnKey(s string) error {
	return nil
}

func (e *emptyVisitor) OnArrayStart(len int, baseType structform.BaseType) error {
	return nil
}

func (e *emptyVisitor) OnArrayFinished() error {
	return nil
}

func (e *emptyVisitor) OnNil() error {
	return nil
}

func (e *emptyVisitor) OnBool(b bool) error {
	return nil
}

func (e *emptyVisitor) OnString(s string) error {
	return nil
}

func (e *emptyVisitor) OnInt8(i int8) error {
	return nil
}

func (e *emptyVisitor) OnInt16(i int16) error {
	return nil
}

func (e *emptyVisitor) OnInt32(i int32) error {
	return nil
}

func (e *emptyVisitor) OnInt64(i int64) error {
	return nil
}

func (e *emptyVisitor) OnInt(i int) error {
	return nil
}

func (e *emptyVisitor) OnByte(b byte) error {
	return nil
}

func (e *emptyVisitor) OnUint8(u uint8) error {
	return nil
}

func (e *emptyVisitor) OnUint16(u uint16) error {
	return nil
}

func (e *emptyVisitor) OnUint32(u uint32) error {
	return nil
}

func (e *emptyVisitor) OnUint64(u uint64) error {
	return nil
}

func (e *emptyVisitor) OnUint(u uint) error {
	return nil
}

func (e *emptyVisitor) OnFloat32(f float32) error {
	return nil
}

func (e *emptyVisitor) OnFloat64(f float64) error {
	return nil
}

func (e *emptyVisitor) OnBoolArray([]bool) error {
	return nil
}

func (e *emptyVisitor) OnStringArray([]string) error {
	return nil
}

func (e *emptyVisitor) OnInt8Array([]int8) error {
	return nil
}

func (e *emptyVisitor) OnInt16Array([]int16) error {
	return nil
}

func (e *emptyVisitor) OnInt32Array([]int32) error {
	return nil
}

func (e *emptyVisitor) OnInt64Array([]int64) error {
	return nil
}

func (e *emptyVisitor) OnIntArray([]int) error {
	return nil
}

func (e *emptyVisitor) OnBytes([]byte) error {
	return nil
}

func (e *emptyVisitor) OnUint8Array([]uint8) error {
	return nil
}

func (e *emptyVisitor) OnUint16Array([]uint16) error {
	return nil
}

func (e *emptyVisitor) OnUint32Array([]uint32) error {
	return nil
}

func (e *emptyVisitor) OnUint64Array([]uint64) error {
	return nil
}

func (e *emptyVisitor) OnUintArray([]uint) error {
	return nil
}

func (e *emptyVisitor) OnFloat32Array([]float32) error {
	return nil
}

func (e *emptyVisitor) OnFloat64Array([]float64) error {
	return nil
}

func (e *emptyVisitor) OnBoolObject(map[string]bool) error {
	return nil
}

func (e *emptyVisitor) OnStringObject(map[string]string) error {
	return nil
}

func (e *emptyVisitor) OnInt8Object(map[string]int8) error {
	return nil
}

func (e *emptyVisitor) OnInt16Object(map[string]int16) error {
	return nil
}

func (e *emptyVisitor) OnInt32Object(map[string]int32) error {
	return nil
}

func (e *emptyVisitor) OnInt64Object(map[string]int64) error {
	return nil
}

func (e *emptyVisitor) OnIntObject(map[string]int) error {
	return nil
}

func (e *emptyVisitor) OnUint8Object(map[string]uint8) error {
	return nil
}

func (e *emptyVisitor) OnUint16Object(map[string]uint16) error {
	return nil
}

func (e *emptyVisitor) OnUint32Object(map[string]uint32) error {
	return nil
}

func (e *emptyVisitor) OnUint64Object(map[string]uint64) error {
	return nil
}

func (e *emptyVisitor) OnUintObject(map[string]uint) error {
	return nil
}

func (e *emptyVisitor) OnFloat32Object(map[string]float32) error {
	return nil
}

func (e *emptyVisitor) OnFloat64Object(map[string]float64) error {
	return nil
}

func (e *emptyVisitor) OnStringRef(s []byte) error {
	return nil
}

func (e *emptyVisitor) OnKeyRef(s []byte) error {
	return nil
}
