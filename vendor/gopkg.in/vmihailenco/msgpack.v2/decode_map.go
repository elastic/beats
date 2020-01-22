package msgpack

import (
	"fmt"
	"reflect"

	"gopkg.in/vmihailenco/msgpack.v2/codes"
)

const mapElemsAllocLimit = 1e4

var mapStringStringPtrType = reflect.TypeOf((*map[string]string)(nil))
var mapStringStringType = mapStringStringPtrType.Elem()

var mapStringInterfacePtrType = reflect.TypeOf((*map[string]interface{})(nil))
var mapStringInterfaceType = mapStringInterfacePtrType.Elem()

func decodeMapValue(d *Decoder, v reflect.Value) error {
	n, err := d.DecodeMapLen()
	if err != nil {
		return err
	}

	typ := v.Type()
	if n == -1 {
		v.Set(reflect.Zero(typ))
		return nil
	}

	if v.IsNil() {
		v.Set(reflect.MakeMap(typ))
	}
	keyType := typ.Key()
	valueType := typ.Elem()

	for i := 0; i < n; i++ {
		mk := reflect.New(keyType).Elem()
		if err := d.DecodeValue(mk); err != nil {
			return err
		}

		mv := reflect.New(valueType).Elem()
		if err := d.DecodeValue(mv); err != nil {
			return err
		}

		v.SetMapIndex(mk, mv)
	}

	return nil
}
func decodeMap(d *Decoder) (interface{}, error) {
	n, err := d.DecodeMapLen()
	if err != nil {
		return nil, err
	}
	if n == -1 {
		return nil, nil
	}

	m := make(map[interface{}]interface{}, min(n, mapElemsAllocLimit))
	for i := 0; i < n; i++ {
		mk, err := d.DecodeInterface()
		if err != nil {
			return nil, err
		}
		mv, err := d.DecodeInterface()
		if err != nil {
			return nil, err
		}
		m[mk] = mv
	}
	return m, nil
}

func (d *Decoder) DecodeMapLen() (int, error) {
	c, err := d.readByte()
	if err != nil {
		return 0, err
	}

	if codes.IsExt(c) {
		if err = d.skipExtHeader(c); err != nil {
			return 0, err
		}

		c, err = d.readByte()
		if err != nil {
			return 0, err
		}
	}

	return d.mapLen(c)
}

func (d *Decoder) mapLen(c byte) (int, error) {
	if c == codes.Nil {
		return -1, nil
	}
	if c >= codes.FixedMapLow && c <= codes.FixedMapHigh {
		return int(c & codes.FixedMapMask), nil
	}
	if c == codes.Map16 {
		n, err := d.uint16()
		return int(n), err
	}
	if c == codes.Map32 {
		n, err := d.uint32()
		return int(n), err
	}
	return 0, fmt.Errorf("msgpack: invalid code %x decoding map length", c)
}

func decodeMapStringStringValue(d *Decoder, v reflect.Value) error {
	mptr := v.Addr().Convert(mapStringStringPtrType).Interface().(*map[string]string)
	return d.decodeMapStringStringPtr(mptr)
}

func (d *Decoder) decodeMapStringStringPtr(ptr *map[string]string) error {
	n, err := d.DecodeMapLen()
	if err != nil {
		return err
	}
	if n == -1 {
		*ptr = nil
		return nil
	}

	m := *ptr
	if m == nil {
		*ptr = make(map[string]string, min(n, mapElemsAllocLimit))
		m = *ptr
	}

	for i := 0; i < n; i++ {
		mk, err := d.DecodeString()
		if err != nil {
			return err
		}
		mv, err := d.DecodeString()
		if err != nil {
			return err
		}
		m[mk] = mv
	}

	return nil
}

func decodeMapStringInterfaceValue(d *Decoder, v reflect.Value) error {
	ptr := v.Addr().Convert(mapStringInterfacePtrType).Interface().(*map[string]interface{})
	return d.decodeMapStringInterfacePtr(ptr)
}

func (d *Decoder) decodeMapStringInterfacePtr(ptr *map[string]interface{}) error {
	n, err := d.DecodeMapLen()
	if err != nil {
		return err
	}
	if n == -1 {
		*ptr = nil
		return nil
	}

	m := *ptr
	if m == nil {
		*ptr = make(map[string]interface{}, min(n, mapElemsAllocLimit))
		m = *ptr
	}

	for i := 0; i < n; i++ {
		mk, err := d.DecodeString()
		if err != nil {
			return err
		}
		mv, err := d.DecodeInterface()
		if err != nil {
			return err
		}
		m[mk] = mv
	}

	return nil
}

func (d *Decoder) DecodeMap() (interface{}, error) {
	return d.DecodeMapFunc(d)
}

func (d *Decoder) skipMap(c byte) error {
	n, err := d.mapLen(c)
	if err != nil {
		return err
	}
	for i := 0; i < n; i++ {
		if err := d.Skip(); err != nil {
			return err
		}
		if err := d.Skip(); err != nil {
			return err
		}
	}
	return nil
}

func decodeStructValue(d *Decoder, strct reflect.Value) error {
	c, err := d.readByte()
	if err != nil {
		return err
	}

	var isArray bool

	n, err := d.mapLen(c)
	if err != nil {
		var err2 error
		n, err2 = d.arrayLen(c)
		if err2 != nil {
			return err
		}
		isArray = true
	}
	if n == -1 {
		strct.Set(reflect.Zero(strct.Type()))
		return nil
	}

	fields := structs.Fields(strct.Type())

	if isArray {
		for i, f := range fields.List {
			if i >= n {
				break
			}
			if err := f.DecodeValue(d, strct); err != nil {
				return err
			}
		}
		// Skip extra values.
		for i := len(fields.List); i < n; i++ {
			if err := d.Skip(); err != nil {
				return err
			}
		}
		return nil
	}

	for i := 0; i < n; i++ {
		name, err := d.DecodeString()
		if err != nil {
			return err
		}
		if f := fields.Table[name]; f != nil {
			if err := f.DecodeValue(d, strct); err != nil {
				return err
			}
		} else {
			if err := d.Skip(); err != nil {
				return err
			}
		}
	}

	return nil
}
