// Copyright 2017 Tamás Gulácsi
//
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.

package goracle

/*
#include <stdlib.h>
#include "dpiImpl.h"
*/
import "C"
import (
	"fmt"
	"reflect"
	"unsafe"

	"github.com/pkg/errors"
)

var _ = fmt.Printf

// Object represents a dpiObject.
type Object struct {
	scratch Data
	ObjectType
	dpiObject *C.dpiObject
}

func (O *Object) getError() error { return O.drv.getError() }

// ErrNoSuchKey is the error for missing key in lookup.
var ErrNoSuchKey = errors.New("no such key")

// GetAttribute gets the i-th attribute into data.
func (O *Object) GetAttribute(data *Data, name string) error {
	if O == nil || O.dpiObject == nil {
		panic("nil dpiObject")
	}
	attr, ok := O.Attributes[name]
	if !ok {
		return errors.Wrap(ErrNoSuchKey, name)
	}

	data.reset()
	data.NativeTypeNum = attr.NativeTypeNum
	data.ObjectType = attr.ObjectType
	wasNull := data.dpiData == nil
	// the maximum length of that buffer must be supplied
	// in the value.asBytes.length attribute before calling this function.
	if attr.NativeTypeNum == C.DPI_NATIVE_TYPE_BYTES && attr.OracleTypeNum == C.DPI_ORACLE_TYPE_NUMBER {
		var a [22]byte
		C.dpiData_setBytes(data.dpiData, (*C.char)(unsafe.Pointer(&a[0])), 22)
	}
	//fmt.Printf("getAttributeValue(%p, %p, %d, %+v)\n", O.dpiObject, attr.dpiObjectAttr, data.NativeTypeNum, data.dpiData)
	if C.dpiObject_getAttributeValue(O.dpiObject, attr.dpiObjectAttr, data.NativeTypeNum, data.dpiData) == C.DPI_FAILURE {
		if wasNull {
			C.free(unsafe.Pointer(data.dpiData))
			data.dpiData = nil
		}
		return errors.Wrapf(O.getError(), "getAttributeValue(obj=%+v, attr=%+v, typ=%d)", O, attr.dpiObjectAttr, data.NativeTypeNum)
	}
	//fmt.Printf("getAttributeValue(%p, %q=%p, %d, %+v)\n", O.dpiObject, attr.Name, attr.dpiObjectAttr, data.NativeTypeNum, data.dpiData)
	return nil
}

// SetAttribute sets the i-th attribute with data.
func (O *Object) SetAttribute(name string, data *Data) error {
	attr := O.Attributes[name]
	if data.NativeTypeNum == 0 {
		data.NativeTypeNum = attr.NativeTypeNum
		data.ObjectType = attr.ObjectType
	}
	if C.dpiObject_setAttributeValue(O.dpiObject, attr.dpiObjectAttr, data.NativeTypeNum, data.dpiData) == C.DPI_FAILURE {
		return O.getError()
	}
	return nil
}

// Get scans the named attribute into dest, and returns it.
func (O *Object) Get(name string) (interface{}, error) {
	if err := O.GetAttribute(&O.scratch, name); err != nil {
		return nil, err
	}
	isObject := O.scratch.IsObject()
	if isObject {
		O.scratch.ObjectType = O.Attributes[name].ObjectType
	}
	v := O.scratch.Get()
	if !isObject {
		return v, nil
	}
	sub := v.(*Object)
	if sub != nil && sub.CollectionOf != nil {
		return &ObjectCollection{Object: sub}, nil
	}
	return sub, nil
}

// ObjectCollection represents a Collection of Objects - itself an Object, too.
type ObjectCollection struct {
	*Object
}

// ErrNotCollection is returned when the Object is not a collection.
var ErrNotCollection = errors.New("not collection")

// ErrNotExist is returned when the collection's requested element does not exist.
var ErrNotExist = errors.New("not exist")

// AsSlice retrieves the collection into a slice.
func (O *ObjectCollection) AsSlice(dest interface{}) (interface{}, error) {
	var data Data
	var dr reflect.Value
	needsInit := dest == nil
	if !needsInit {
		dr = reflect.ValueOf(dest)
	}
	for i, err := O.First(); err == nil; i, err = O.Next(i) {
		if O.CollectionOf.NativeTypeNum == C.DPI_NATIVE_TYPE_OBJECT {
			data.ObjectType = *O.CollectionOf
		}
		if err = O.Get(&data, i); err != nil {
			return dest, err
		}
		vr := reflect.ValueOf(data.Get())
		if needsInit {
			needsInit = false
			length, lengthErr := O.Len()
			if lengthErr != nil {
				return dr.Interface(), lengthErr
			}
			dr = reflect.MakeSlice(reflect.SliceOf(vr.Type()), 0, length)
		}
		if dr = reflect.Append(dr, vr); err != nil {
			return dr.Interface(), err
		}
	}
	return dr.Interface(), nil
}

// Append data to the collection.
func (O *ObjectCollection) Append(data *Data) error {
	if C.dpiObject_appendElement(O.dpiObject, data.NativeTypeNum, data.dpiData) == C.DPI_FAILURE {
		return errors.Wrapf(O.getError(), "append(%d)", data.NativeTypeNum)
	}
	return nil
}

// Delete i-th element of the collection.
func (O *ObjectCollection) Delete(i int) error {
	if C.dpiObject_deleteElementByIndex(O.dpiObject, C.int32_t(i)) == C.DPI_FAILURE {
		return errors.Wrapf(O.getError(), "delete(%d)", i)
	}
	return nil
}

// Get the i-th element of the collection into data.
func (O *ObjectCollection) Get(data *Data, i int) error {
	if data == nil {
		panic("data cannot be nil")
	}
	idx := C.int32_t(i)
	var exists C.int
	if C.dpiObject_getElementExistsByIndex(O.dpiObject, idx, &exists) == C.DPI_FAILURE {
		return errors.Wrapf(O.getError(), "exists(%d)", idx)
	}
	if exists == 0 {
		return ErrNotExist
	}
	data.reset()
	data.NativeTypeNum = O.CollectionOf.NativeTypeNum
	data.ObjectType = *O.CollectionOf
	if C.dpiObject_getElementValueByIndex(O.dpiObject, idx, data.NativeTypeNum, data.dpiData) == C.DPI_FAILURE {
		return errors.Wrapf(O.getError(), "get(%d[%d])", idx, data.NativeTypeNum)
	}
	return nil
}

// Set the i-th element of the collection with data.
func (O *ObjectCollection) Set(i int, data *Data) error {
	if C.dpiObject_setElementValueByIndex(O.dpiObject, C.int32_t(i), data.NativeTypeNum, data.dpiData) == C.DPI_FAILURE {
		return errors.Wrapf(O.getError(), "set(%d[%d])", i, data.NativeTypeNum)
	}
	return nil
}

// First returns the first element's index of the collection.
func (O *ObjectCollection) First() (int, error) {
	var exists C.int
	var idx C.int32_t
	if C.dpiObject_getFirstIndex(O.dpiObject, &idx, &exists) == C.DPI_FAILURE {
		return 0, errors.Wrap(O.getError(), "first")
	}
	if exists == 1 {
		return int(idx), nil
	}
	return 0, ErrNotExist
}

// Last returns the index of the last element.
func (O *ObjectCollection) Last() (int, error) {
	var exists C.int
	var idx C.int32_t
	if C.dpiObject_getLastIndex(O.dpiObject, &idx, &exists) == C.DPI_FAILURE {
		return 0, errors.Wrap(O.getError(), "last")
	}
	if exists == 1 {
		return int(idx), nil
	}
	return 0, ErrNotExist
}

// Next returns the succeeding index of i.
func (O *ObjectCollection) Next(i int) (int, error) {
	var exists C.int
	var idx C.int32_t
	if C.dpiObject_getNextIndex(O.dpiObject, C.int32_t(i), &idx, &exists) == C.DPI_FAILURE {
		return 0, errors.Wrapf(O.getError(), "next(%d)", i)
	}
	if exists == 1 {
		return int(idx), nil
	}
	return 0, ErrNotExist
}

// Len returns the length of the collection.
func (O *ObjectCollection) Len() (int, error) {
	var size C.int32_t
	if C.dpiObject_getSize(O.dpiObject, &size) == C.DPI_FAILURE {
		return 0, errors.Wrap(O.getError(), "len")
	}
	return int(size), nil
}

// Trim the collection to n.
func (O *ObjectCollection) Trim(n int) error {
	if C.dpiObject_trim(O.dpiObject, C.uint32_t(n)) == C.DPI_FAILURE {
		return O.getError()
	}
	return nil
}

// ObjectType holds type info of an Object.
type ObjectType struct {
	Schema, Name                        string
	DBSize, ClientSizeInBytes, CharSize int
	CollectionOf                        *ObjectType
	Attributes                          map[string]ObjectAttribute
	dpiObjectType                       *C.dpiObjectType
	drv                                 *drv
	OracleTypeNum                       C.dpiOracleTypeNum
	NativeTypeNum                       C.dpiNativeTypeNum
	Precision                           int16
	Scale                               int8
	FsPrecision                         uint8
}

func (t ObjectType) getError() error { return t.drv.getError() }

// FullName returns the object's name with the schame prepended.
func (t ObjectType) FullName() string {
	if t.Schema == "" {
		return t.Name
	}
	return t.Schema + "." + t.Name
}

// GetObjectType returns the ObjectType of a name.
func (c *conn) GetObjectType(name string) (ObjectType, error) {
	cName := C.CString(name)
	defer func() { C.free(unsafe.Pointer(cName)) }()
	objType := (*C.dpiObjectType)(C.malloc(C.sizeof_void))
	if C.dpiConn_getObjectType(c.dpiConn, cName, C.uint32_t(len(name)), &objType) == C.DPI_FAILURE {
		C.free(unsafe.Pointer(objType))
		return ObjectType{}, errors.Wrapf(c.getError(), "getObjectType(%q) conn=%p", name, c.dpiConn)
	}
	t := ObjectType{drv: c.drv, dpiObjectType: objType}
	return t, t.init()
}

// NewObject returns a new Object with ObjectType type.
func (t ObjectType) NewObject() (*Object, error) {
	obj := (*C.dpiObject)(C.malloc(C.sizeof_void))
	if C.dpiObjectType_createObject(t.dpiObjectType, &obj) == C.DPI_FAILURE {
		C.free(unsafe.Pointer(obj))
		return nil, t.getError()
	}
	return &Object{ObjectType: t, dpiObject: obj}, nil
}

func wrapObject(d *drv, objectType *C.dpiObjectType, object *C.dpiObject) (*Object, error) {
	if objectType == nil {
		return nil, errors.New("objectType is nil")
	}
	o := &Object{
		ObjectType: ObjectType{dpiObjectType: objectType, drv: d},
		dpiObject:  object,
	}
	return o, o.init()
}

func (t *ObjectType) init() error {
	if t.drv == nil {
		panic("drv is nil")
	}
	if t.Name != "" && t.Attributes != nil {
		return nil
	}
	if t.dpiObjectType == nil {
		return nil
	}
	var info C.dpiObjectTypeInfo
	if C.dpiObjectType_getInfo(t.dpiObjectType, &info) == C.DPI_FAILURE {
		return errors.Wrapf(t.getError(), "%v.getInfo", t)
	}
	t.Schema = C.GoStringN(info.schema, C.int(info.schemaLength))
	t.Name = C.GoStringN(info.name, C.int(info.nameLength))
	t.CollectionOf = nil
	numAttributes := int(info.numAttributes)

	if info.isCollection == 1 {
		t.CollectionOf = &ObjectType{drv: t.drv}
		if err := t.CollectionOf.fromDataTypeInfo(info.elementTypeInfo); err != nil {
			return err
		}
		if t.CollectionOf.Name == "" {
			t.CollectionOf.Schema = t.Schema
			t.CollectionOf.Name = t.Name
		}
	}
	if numAttributes == 0 {
		t.Attributes = map[string]ObjectAttribute{}
		return nil
	}
	t.Attributes = make(map[string]ObjectAttribute, numAttributes)
	attrs := make([]*C.dpiObjectAttr, numAttributes)
	if C.dpiObjectType_getAttributes(t.dpiObjectType,
		C.uint16_t(len(attrs)),
		(**C.dpiObjectAttr)(unsafe.Pointer(&attrs[0])),
	) == C.DPI_FAILURE {
		return errors.Wrapf(t.getError(), "%v.getAttributes", t)
	}
	for i, attr := range attrs {
		var attrInfo C.dpiObjectAttrInfo
		if C.dpiObjectAttr_getInfo(attr, &attrInfo) == C.DPI_FAILURE {
			return errors.Wrapf(t.getError(), "%v.attr_getInfo", attr)
		}
		if Log != nil {
			Log("i", i, "attrInfo", attrInfo)
		}
		typ := attrInfo.typeInfo
		sub, err := objectTypeFromDataTypeInfo(t.drv, typ)
		if err != nil {
			return err
		}
		objAttr := ObjectAttribute{
			dpiObjectAttr: attr,
			Name:          C.GoStringN(attrInfo.name, C.int(attrInfo.nameLength)),
			ObjectType:    sub,
		}
		//fmt.Printf("%d=%q. typ=%+v sub=%+v\n", i, objAttr.Name, typ, sub)
		t.Attributes[objAttr.Name] = objAttr
	}
	return nil
}

func (t *ObjectType) fromDataTypeInfo(typ C.dpiDataTypeInfo) error {
	t.dpiObjectType = typ.objectType

	t.OracleTypeNum = typ.oracleTypeNum
	t.NativeTypeNum = typ.defaultNativeTypeNum
	t.DBSize = int(typ.dbSizeInBytes)
	t.ClientSizeInBytes = int(typ.clientSizeInBytes)
	t.CharSize = int(typ.sizeInChars)
	t.Precision = int16(typ.precision)
	t.Scale = int8(typ.scale)
	t.FsPrecision = uint8(typ.fsPrecision)
	return t.init()
}
func objectTypeFromDataTypeInfo(drv *drv, typ C.dpiDataTypeInfo) (ObjectType, error) {
	if drv == nil {
		panic("drv nil")
	}
	if typ.oracleTypeNum == 0 {
		panic("typ is nil")
	}
	t := ObjectType{drv: drv}
	err := t.fromDataTypeInfo(typ)
	return t, err
}

// ObjectAttribute is an attribute of an Object.
type ObjectAttribute struct {
	Name          string
	dpiObjectAttr *C.dpiObjectAttr
	ObjectType
}

// Close the ObjectAttribute.
func (A ObjectAttribute) Close() error {
	attr := A.dpiObjectAttr
	if attr == nil {
		return nil
	}

	A.dpiObjectAttr = nil
	if C.dpiObjectAttr_release(attr) == C.DPI_FAILURE {
		return A.getError()
	}
	return nil
}

// GetObjectType returns the ObjectType for the name.
func GetObjectType(ex Execer, typeName string) (ObjectType, error) {
	c, err := getConn(ex)
	if err != nil {
		return ObjectType{}, errors.WithMessage(err, "getConn for "+typeName)
	}
	return c.GetObjectType(typeName)
}
