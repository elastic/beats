// Copyright 2017 Tamás Gulácsi
//
//
// SPDX-License-Identifier: UPL-1.0 OR Apache-2.0

package godror

/*
#include <stdlib.h>
#include "dpiImpl.h"
*/
import "C"
import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"time"
	"unsafe"

	errors "golang.org/x/xerrors"
)

// Data holds the data to/from Oracle.
type Data struct {
	ObjectType    ObjectType
	dpiData       C.dpiData
	implicitObj   bool
	NativeTypeNum C.dpiNativeTypeNum
}

var ErrNotSupported = errors.New("not supported")

// NewData creates a new Data structure for the given type, populated with the given type.
func NewData(v interface{}) (*Data, error) {
	if v == nil {
		return nil, errors.Errorf("%s: %w", "nil type", ErrNotSupported)
	}
	data := Data{dpiData: C.dpiData{isNull: 1}}
	return &data, data.Set(v)
}

// IsNull returns whether the data is null.
func (d *Data) IsNull() bool {
	// Use of C.dpiData_getIsNull(&d.dpiData) would be safer,
	// but ODPI-C 3.1.4 just returns dpiData->isNull, so do the same
	// without calling CGO.
	return d == nil || d.dpiData.isNull == 1
}

// SetNull sets the value of the data to be the null value.
func (d *Data) SetNull() {
	if !d.IsNull() {
		// Maybe C.dpiData_setNull(&d.dpiData) would be safer, but as we don't use C.dpiData_getIsNull,
		// and those functions (at least in ODPI-C 3.1.4) just operate on data->isNull directly,
		// don't use CGO if possible.
		d.dpiData.isNull = 1
	}
}

// GetBool returns the bool data.
func (d *Data) GetBool() bool {
	return !d.IsNull() && C.dpiData_getBool(&d.dpiData) == 1
}

// SetBool sets the data as bool.
func (d *Data) SetBool(b bool) {
	var i C.int
	if b {
		i = 1
	}
	C.dpiData_setBool(&d.dpiData, i)
}

// GetBytes returns the []byte from the data.
func (d *Data) GetBytes() []byte {
	if d.IsNull() {
		return nil
	}
	b := C.dpiData_getBytes(&d.dpiData)
	if b.ptr == nil || b.length == 0 {
		return nil
	}
	return ((*[32767]byte)(unsafe.Pointer(b.ptr)))[:b.length:b.length]
}

// SetBytes set the data as []byte.
func (d *Data) SetBytes(b []byte) {
	if b == nil {
		d.dpiData.isNull = 1
		return
	}
	C.dpiData_setBytes(&d.dpiData, (*C.char)(unsafe.Pointer(&b[0])), C.uint32_t(len(b)))
}

// GetFloat32 gets float32 from the data.
func (d *Data) GetFloat32() float32 {
	if d.IsNull() {
		return 0
	}
	return float32(C.dpiData_getFloat(&d.dpiData))
}

// SetFloat32 sets the data as float32.
func (d *Data) SetFloat32(f float32) {
	C.dpiData_setFloat(&d.dpiData, C.float(f))
}

// GetFloat64 gets float64 from the data.
func (d *Data) GetFloat64() float64 {
	//fmt.Println("GetFloat64", d.IsNull(), d)
	if d.IsNull() {
		return 0
	}
	return float64(C.dpiData_getDouble(&d.dpiData))
}

// SetFloat64 sets the data as float64.
func (d *Data) SetFloat64(f float64) {
	C.dpiData_setDouble(&d.dpiData, C.double(f))
}

// GetInt64 gets int64 from the data.
func (d *Data) GetInt64() int64 {
	if d.IsNull() {
		return 0
	}
	return int64(C.dpiData_getInt64(&d.dpiData))
}

// SetInt64 sets the data as int64.
func (d *Data) SetInt64(i int64) {
	C.dpiData_setInt64(&d.dpiData, C.int64_t(i))
}

// GetIntervalDS gets duration as interval date-seconds from data.
func (d *Data) GetIntervalDS() time.Duration {
	if d.IsNull() {
		return 0
	}
	ds := C.dpiData_getIntervalDS(&d.dpiData)
	return time.Duration(ds.days)*24*time.Hour +
		time.Duration(ds.hours)*time.Hour +
		time.Duration(ds.minutes)*time.Minute +
		time.Duration(ds.seconds)*time.Second +
		time.Duration(ds.fseconds)
}

// SetIntervalDS sets the duration as interval date-seconds to data.
func (d *Data) SetIntervalDS(dur time.Duration) {
	C.dpiData_setIntervalDS(&d.dpiData,
		C.int32_t(int64(dur.Hours())/24),
		C.int32_t(int64(dur.Hours())%24), C.int32_t(dur.Minutes()), C.int32_t(dur.Seconds()),
		C.int32_t(dur.Nanoseconds()),
	)
}

// GetIntervalYM gets IntervalYM from the data.
func (d *Data) GetIntervalYM() IntervalYM {
	if d.IsNull() {
		return IntervalYM{}
	}
	ym := C.dpiData_getIntervalYM(&d.dpiData)
	return IntervalYM{Years: int(ym.years), Months: int(ym.months)}
}

// SetIntervalYM sets IntervalYM to the data.
func (d *Data) SetIntervalYM(ym IntervalYM) {
	C.dpiData_setIntervalYM(&d.dpiData, C.int32_t(ym.Years), C.int32_t(ym.Months))
}

// GetLob gets data as Lob.
func (d *Data) GetLob() *Lob {
	if d.IsNull() {
		return nil
	}
	return &Lob{Reader: &dpiLobReader{dpiLob: C.dpiData_getLOB(&d.dpiData)}}
}

// SetLob sets Lob to the data.
func (d *Data) SetLob(lob *DirectLob) {
	C.dpiData_setLOB(&d.dpiData, lob.dpiLob)
}

// GetObject gets Object from data.
//
// As with all Objects, you MUST call Close on it when not needed anymore!
func (d *Data) GetObject() *Object {
	if d == nil {
		panic("null")
	}
	if d.IsNull() {
		return nil
	}

	o := C.dpiData_getObject(&d.dpiData)
	if o == nil {
		return nil
	}
	if !d.implicitObj {
		if C.dpiObject_addRef(o) == C.DPI_FAILURE {
			panic(d.ObjectType.getError())
		}
	}
	obj := &Object{dpiObject: o, ObjectType: d.ObjectType}
	obj.init()
	return obj
}

// SetObject sets Object to data.
func (d *Data) SetObject(o *Object) {
	C.dpiData_setObject(&d.dpiData, o.dpiObject)
}

// GetStmt gets Stmt from data.
func (d *Data) GetStmt() driver.Stmt {
	if d.IsNull() {
		return nil
	}
	return &statement{dpiStmt: C.dpiData_getStmt(&d.dpiData)}
}

// SetStmt sets Stmt to data.
func (d *Data) SetStmt(s *statement) {
	C.dpiData_setStmt(&d.dpiData, s.dpiStmt)
}

// GetTime gets Time from data.
func (d *Data) GetTime() time.Time {
	if d.IsNull() {
		return time.Time{}
	}
	ts := C.dpiData_getTimestamp(&d.dpiData)
	return time.Date(
		int(ts.year), time.Month(ts.month), int(ts.day),
		int(ts.hour), int(ts.minute), int(ts.second), int(ts.fsecond),
		timeZoneFor(ts.tzHourOffset, ts.tzMinuteOffset),
	)

}

// SetTime sets Time to data.
func (d *Data) SetTime(t time.Time) {
	_, z := t.Zone()
	C.dpiData_setTimestamp(&d.dpiData,
		C.int16_t(t.Year()), C.uint8_t(t.Month()), C.uint8_t(t.Day()),
		C.uint8_t(t.Hour()), C.uint8_t(t.Minute()), C.uint8_t(t.Second()), C.uint32_t(t.Nanosecond()),
		C.int8_t(z/3600), C.int8_t((z%3600)/60),
	)
}

// GetUint64 gets data as uint64.
func (d *Data) GetUint64() uint64 {
	if d.IsNull() {
		return 0
	}
	return uint64(C.dpiData_getUint64(&d.dpiData))
}

// SetUint64 sets data to uint64.
func (d *Data) SetUint64(u uint64) {
	C.dpiData_setUint64(&d.dpiData, C.uint64_t(u))
}

// IntervalYM holds Years and Months as interval.
type IntervalYM struct {
	Years, Months int
}

// Get returns the contents of Data.
func (d *Data) Get() interface{} {
	switch d.NativeTypeNum {
	case C.DPI_NATIVE_TYPE_BOOLEAN:
		return d.GetBool()
	case C.DPI_NATIVE_TYPE_BYTES:
		return d.GetBytes()
	case C.DPI_NATIVE_TYPE_DOUBLE:
		return d.GetFloat64()
	case C.DPI_NATIVE_TYPE_FLOAT:
		return d.GetFloat32()
	case C.DPI_NATIVE_TYPE_INT64:
		return d.GetInt64()
	case C.DPI_NATIVE_TYPE_INTERVAL_DS:
		return d.GetIntervalDS()
	case C.DPI_NATIVE_TYPE_INTERVAL_YM:
		return d.GetIntervalYM()
	case C.DPI_NATIVE_TYPE_LOB:
		return d.GetLob()
	case C.DPI_NATIVE_TYPE_OBJECT:
		return d.GetObject()
	case C.DPI_NATIVE_TYPE_STMT:
		return d.GetStmt()
	case C.DPI_NATIVE_TYPE_TIMESTAMP:
		return d.GetTime()
	case C.DPI_NATIVE_TYPE_UINT64:
		return d.GetUint64()
	default:
		panic(fmt.Sprintf("unknown NativeTypeNum=%d", d.NativeTypeNum))
	}
}

// Set the data.
func (d *Data) Set(v interface{}) error {
	if v == nil {
		return errors.Errorf("%s: %w", "nil type", ErrNotSupported)
	}
	switch x := v.(type) {
	case int32:
		d.NativeTypeNum = C.DPI_NATIVE_TYPE_INT64
		d.SetInt64(int64(x))
	case int64:
		d.NativeTypeNum = C.DPI_NATIVE_TYPE_INT64
		d.SetInt64(x)
	case uint64:
		d.NativeTypeNum = C.DPI_NATIVE_TYPE_UINT64
		d.SetUint64(x)
	case float32:
		d.NativeTypeNum = C.DPI_NATIVE_TYPE_FLOAT
		d.SetFloat32(x)
	case float64:
		d.NativeTypeNum = C.DPI_NATIVE_TYPE_DOUBLE
		d.SetFloat64(x)
	case string:
		d.NativeTypeNum = C.DPI_NATIVE_TYPE_BYTES
		d.SetBytes([]byte(x))
	case []byte:
		d.NativeTypeNum = C.DPI_NATIVE_TYPE_BYTES
		d.SetBytes(x)
	case time.Time:
		d.NativeTypeNum = C.DPI_NATIVE_TYPE_TIMESTAMP
		d.SetTime(x)
	case time.Duration:
		d.NativeTypeNum = C.DPI_NATIVE_TYPE_INTERVAL_DS
		d.SetIntervalDS(x)
	case IntervalYM:
		d.NativeTypeNum = C.DPI_NATIVE_TYPE_INTERVAL_YM
		d.SetIntervalYM(x)
	case *DirectLob:
		d.NativeTypeNum = C.DPI_NATIVE_TYPE_LOB
		d.SetLob(x)
	case *Object:
		d.NativeTypeNum = C.DPI_NATIVE_TYPE_OBJECT
		d.ObjectType = x.ObjectType
		d.SetObject(x)
	//case *stmt:
	//d.NativeTypeNum = C.DPI_NATIVE_TYPE_STMT
	//d.SetStmt(x)
	case bool:
		d.NativeTypeNum = C.DPI_NATIVE_TYPE_BOOLEAN
		d.SetBool(x)
	//case rowid:
	//d.NativeTypeNum = C.DPI_NATIVE_TYPE_ROWID
	//d.SetRowid(x)
	default:
		return errors.Errorf("%T: %w", ErrNotSupported, v)
	}
	return nil
}

// IsObject returns whether the data contains an Object or not.
func (d *Data) IsObject() bool {
	return d.NativeTypeNum == C.DPI_NATIVE_TYPE_OBJECT
}

// NewData returns Data for input parameters on Object/ObjectCollection.
func (c *conn) NewData(baseType interface{}, sliceLen, bufSize int) ([]*Data, error) {
	if c == nil || c.dpiConn == nil {
		return nil, errors.New("connection is nil")
	}

	vi, err := newVarInfo(baseType, sliceLen, bufSize)
	if err != nil {
		return nil, err
	}

	v, dpiData, err := c.newVar(vi)
	if err != nil {
		return nil, err
	}
	defer C.dpiVar_release(v)

	data := make([]*Data, sliceLen)
	for i := 0; i < sliceLen; i++ {
		data[i] = &Data{dpiData: dpiData[i], NativeTypeNum: vi.NatTyp}
	}

	return data, nil
}

func newVarInfo(baseType interface{}, sliceLen, bufSize int) (varInfo, error) {
	var vi varInfo

	switch v := baseType.(type) {
	case Lob, []Lob:
		vi.NatTyp = C.DPI_NATIVE_TYPE_LOB
		var isClob bool
		switch v := v.(type) {
		case Lob:
			isClob = v.IsClob
		case []Lob:
			isClob = len(v) > 0 && v[0].IsClob
		}
		if isClob {
			vi.Typ = C.DPI_ORACLE_TYPE_CLOB
		} else {
			vi.Typ = C.DPI_ORACLE_TYPE_BLOB
		}
	case Number, []Number:
		vi.Typ, vi.NatTyp = C.DPI_ORACLE_TYPE_NUMBER, C.DPI_NATIVE_TYPE_BYTES
	case int, []int, int64, []int64, sql.NullInt64, []sql.NullInt64:
		vi.Typ, vi.NatTyp = C.DPI_ORACLE_TYPE_NUMBER, C.DPI_NATIVE_TYPE_INT64
	case int32, []int32:
		vi.Typ, vi.NatTyp = C.DPI_ORACLE_TYPE_NATIVE_INT, C.DPI_NATIVE_TYPE_INT64
	case uint, []uint, uint64, []uint64:
		vi.Typ, vi.NatTyp = C.DPI_ORACLE_TYPE_NUMBER, C.DPI_NATIVE_TYPE_UINT64
	case uint32, []uint32:
		vi.Typ, vi.NatTyp = C.DPI_ORACLE_TYPE_NATIVE_UINT, C.DPI_NATIVE_TYPE_UINT64
	case float32, []float32:
		vi.Typ, vi.NatTyp = C.DPI_ORACLE_TYPE_NATIVE_FLOAT, C.DPI_NATIVE_TYPE_FLOAT
	case float64, []float64, sql.NullFloat64, []sql.NullFloat64:
		vi.Typ, vi.NatTyp = C.DPI_ORACLE_TYPE_NATIVE_DOUBLE, C.DPI_NATIVE_TYPE_DOUBLE
	case bool, []bool:
		vi.Typ, vi.NatTyp = C.DPI_ORACLE_TYPE_BOOLEAN, C.DPI_NATIVE_TYPE_BOOLEAN
	case []byte, [][]byte:
		vi.Typ, vi.NatTyp = C.DPI_ORACLE_TYPE_RAW, C.DPI_NATIVE_TYPE_BYTES
		switch v := v.(type) {
		case []byte:
			bufSize = len(v)
		case [][]byte:
			for _, b := range v {
				if n := len(b); n > bufSize {
					bufSize = n
				}
			}
		}
	case string, []string, nil:
		vi.Typ, vi.NatTyp = C.DPI_ORACLE_TYPE_VARCHAR, C.DPI_NATIVE_TYPE_BYTES
		bufSize = 32767
	case time.Time, []time.Time:
		vi.Typ, vi.NatTyp = C.DPI_ORACLE_TYPE_DATE, C.DPI_NATIVE_TYPE_TIMESTAMP
	case userType, []userType:
		vi.Typ, vi.NatTyp = C.DPI_ORACLE_TYPE_OBJECT, C.DPI_NATIVE_TYPE_OBJECT
		switch v := v.(type) {
		case userType:
			vi.ObjectType = v.ObjectRef().ObjectType.dpiObjectType
		case []userType:
			if len(v) > 0 {
				vi.ObjectType = v[0].ObjectRef().ObjectType.dpiObjectType
			}
		}
	default:
		return vi, errors.Errorf("unknown type %T", v)
	}

	vi.IsPLSArray = reflect.TypeOf(baseType).Kind() == reflect.Slice
	vi.SliceLen = sliceLen
	vi.BufSize = bufSize

	return vi, nil
}

func (d *Data) reset() {
	d.NativeTypeNum = 0
	d.ObjectType = ObjectType{}
	d.implicitObj = false
	d.SetBytes(nil)
	d.dpiData.isNull = 1
}
