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
	"database/sql/driver"
	"fmt"
	"time"
	"unsafe"
)

// Data holds the data to/from Oracle.
type Data struct {
	ObjectType    ObjectType
	dpiData       *C.dpiData
	NativeTypeNum C.dpiNativeTypeNum
}

// IsNull returns whether the data is null.
func (d *Data) IsNull() bool {
	return d == nil || d.dpiData == nil || d.dpiData.isNull == 1
}

// GetBool returns the bool data.
func (d *Data) GetBool() bool {
	return !d.IsNull() && C.dpiData_getBool(d.dpiData) == 1
}

// SetBool sets the data as bool.
func (d *Data) SetBool(b bool) {
	var i C.int
	if b {
		i = 1
	}
	C.dpiData_setBool(d.dpiData, i)
}

// GetBytes returns the []byte from the data.
func (d *Data) GetBytes() []byte {
	if d.IsNull() {
		return nil
	}
	b := C.dpiData_getBytes(d.dpiData)
	return ((*[32767]byte)(unsafe.Pointer(b.ptr)))[:b.length:b.length]
}

// SetBytes set the data as []byte.
func (d *Data) SetBytes(b []byte) {
	if b == nil {
		d.dpiData.isNull = 1
		return
	}
	C.dpiData_setBytes(d.dpiData, (*C.char)(unsafe.Pointer(&b[0])), C.uint32_t(len(b)))
}

// GetFloat32 gets float32 from the data.
func (d *Data) GetFloat32() float32 {
	if d.IsNull() {
		return 0
	}
	return float32(C.dpiData_getFloat(d.dpiData))
}

// SetFloat32 sets the data as float32.
func (d *Data) SetFloat32(f float32) {
	C.dpiData_setFloat(d.dpiData, C.float(f))
}

// GetFloat64 gets float64 from the data.
func (d *Data) GetFloat64() float64 {
	//fmt.Println("GetFloat64", d.IsNull(), d)
	if d.IsNull() {
		return 0
	}
	return float64(C.dpiData_getDouble(d.dpiData))
}

// SetFloat64 sets the data as float64.
func (d *Data) SetFloat64(f float64) {
	C.dpiData_setDouble(d.dpiData, C.double(f))
}

// GetInt64 gets int64 from the data.
func (d *Data) GetInt64() int64 {
	if d.IsNull() {
		return 0
	}
	return int64(C.dpiData_getInt64(d.dpiData))
}

// SetInt64 sets the data as int64.
func (d *Data) SetInt64(i int64) {
	C.dpiData_setInt64(d.dpiData, C.int64_t(i))
}

// GetIntervalDS gets duration as interval date-seconds from data.
func (d *Data) GetIntervalDS() time.Duration {
	if d.IsNull() {
		return 0
	}
	ds := C.dpiData_getIntervalDS(d.dpiData)
	return time.Duration(ds.days)*24*time.Hour +
		time.Duration(ds.hours)*time.Hour +
		time.Duration(ds.minutes)*time.Minute +
		time.Duration(ds.seconds)*time.Second +
		time.Duration(ds.fseconds)
}

// SetIntervalDS sets the duration as interval date-seconds to data.
func (d *Data) SetIntervalDS(dur time.Duration) {
	C.dpiData_setIntervalDS(d.dpiData,
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
	ym := C.dpiData_getIntervalYM(d.dpiData)
	return IntervalYM{Years: int(ym.years), Months: int(ym.months)}
}

// SetIntervalYM sets IntervalYM to the data.
func (d *Data) SetIntervalYM(ym IntervalYM) {
	C.dpiData_setIntervalYM(d.dpiData, C.int32_t(ym.Years), C.int32_t(ym.Months))
}

// GetLob gets data as Lob.
func (d *Data) GetLob() *Lob {
	if d.IsNull() {
		return nil
	}
	return &Lob{Reader: &dpiLobReader{dpiLob: C.dpiData_getLOB(d.dpiData)}}
}

// GetObject gets Object from data.
func (d *Data) GetObject() *Object {
	if d == nil || d.dpiData == nil {
		panic("null")
	}
	if d.IsNull() {
		return nil
	}

	o := C.dpiData_getObject(d.dpiData)
	if o == nil {
		return nil
	}
	obj := &Object{dpiObject: o, ObjectType: d.ObjectType}
	obj.init()
	return obj
}

// SetObject sets Object to data.
func (d *Data) SetObject(o *Object) {
	C.dpiData_setObject(d.dpiData, o.dpiObject)
}

// GetStmt gets Stmt from data.
func (d *Data) GetStmt() driver.Stmt {
	if d.IsNull() {
		return nil
	}
	return &statement{dpiStmt: C.dpiData_getStmt(d.dpiData)}
}

// SetStmt sets Stmt to data.
func (d *Data) SetStmt(s *statement) {
	C.dpiData_setStmt(d.dpiData, s.dpiStmt)
}

// GetTime gets Time from data.
func (d *Data) GetTime() time.Time {
	if d.IsNull() {
		return time.Time{}
	}
	ts := C.dpiData_getTimestamp(d.dpiData)
	return time.Date(
		int(ts.year), time.Month(ts.month), int(ts.day),
		int(ts.hour), int(ts.minute), int(ts.second), int(ts.fsecond),
		timeZoneFor(ts.tzHourOffset, ts.tzMinuteOffset),
	)

}

// SetTime sets Time to data.
func (d *Data) SetTime(t time.Time) {
	_, z := t.Zone()
	C.dpiData_setTimestamp(d.dpiData,
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
	return uint64(C.dpiData_getUint64(d.dpiData))
}

// SetUint64 sets data to uint64.
func (d *Data) SetUint64(u uint64) {
	C.dpiData_setUint64(d.dpiData, C.uint64_t(u))
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

// IsObject returns whether the data contains an Object or not.
func (d *Data) IsObject() bool {
	return d.NativeTypeNum == C.DPI_NATIVE_TYPE_OBJECT
}

func (d *Data) reset() {
	d.NativeTypeNum = 0
	d.ObjectType = ObjectType{}
	if d.dpiData == nil {
		d.dpiData = &C.dpiData{}
	} else {
		d.SetBytes(nil)
	}
}
