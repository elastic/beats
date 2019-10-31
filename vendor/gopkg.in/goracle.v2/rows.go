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
#include "dpiImpl.h"
*/
import "C"
import (
	"bytes"
	"database/sql/driver"
	"fmt"
	"io"
	"math"
	"reflect"
	"strconv"
	"time"
	"unsafe"

	"github.com/pkg/errors"
)

var _ = driver.Rows((*rows)(nil))
var _ = driver.RowsColumnTypeDatabaseTypeName((*rows)(nil))
var _ = driver.RowsColumnTypeLength((*rows)(nil))
var _ = driver.RowsColumnTypeNullable((*rows)(nil))
var _ = driver.RowsColumnTypePrecisionScale((*rows)(nil))
var _ = driver.RowsColumnTypeScanType((*rows)(nil))
var _ = driver.RowsNextResultSet((*rows)(nil))

type rows struct {
	columns   []Column
	vars      []*C.dpiVar
	data      [][]C.dpiData
	err       error
	nextRsErr error
	*statement
	origSt         *statement
	nextRs         *C.dpiStmt
	bufferRowIndex C.uint32_t
	fetched        C.uint32_t
	finished       bool
}

// Columns returns the names of the columns. The number of
// columns of the result is inferred from the length of the
// slice. If a particular column name isn't known, an empty
// string should be returned for that entry.
func (r *rows) Columns() []string {
	names := make([]string, len(r.columns))
	for i, col := range r.columns {
		names[i] = col.Name
	}
	return names
}

// Close closes the rows iterator.
func (r *rows) Close() error {
	if r == nil {
		return nil
	}
	r.columns = nil
	r.data = nil
	for _, v := range r.vars {
		C.dpiVar_release(v)
	}
	r.vars = nil
	if r.statement == nil {
		return nil
	}
	st := r.statement
	r.statement = nil

	st.Lock()
	defer st.Unlock()
	if st.dpiStmt == nil {
		return nil
	}
	var err error
	if C.dpiStmt_release(st.dpiStmt) == C.DPI_FAILURE {
		err = errors.Wrap(r.getError(), "rows/dpiStmt_release")
	}
	return err
}

// ColumnTypeLength return the length of the column type if the column is a variable length type.
// If the column is not a variable length type ok should return false.
// If length is not limited other than system limits, it should return math.MaxInt64.
// The following are examples of returned values for various types:
//
// TEXT          (math.MaxInt64, true)
// varchar(10)   (10, true)
// nvarchar(10)  (10, true)
// decimal       (0, false)
// int           (0, false)
// bytea(30)     (30, true)
func (r *rows) ColumnTypeLength(index int) (length int64, ok bool) {
	switch col := r.columns[index]; col.OracleType {
	case C.DPI_ORACLE_TYPE_VARCHAR, C.DPI_ORACLE_TYPE_NVARCHAR,
		C.DPI_ORACLE_TYPE_CHAR, C.DPI_ORACLE_TYPE_NCHAR,
		C.DPI_ORACLE_TYPE_LONG_VARCHAR,
		C.DPI_NATIVE_TYPE_BYTES:
		return int64(col.Size), true
	case C.DPI_ORACLE_TYPE_CLOB, C.DPI_ORACLE_TYPE_NCLOB,
		C.DPI_ORACLE_TYPE_BLOB,
		C.DPI_ORACLE_TYPE_BFILE,
		C.DPI_NATIVE_TYPE_LOB:
		return math.MaxInt64, true
	default:
		return 0, false
	}
}

// ColumnTypeDatabaseTypeName returns the database system type name without the length.
// Type names should be uppercase.
// Examples of returned types: "VARCHAR", "NVARCHAR", "VARCHAR2", "CHAR", "TEXT", "DECIMAL", "SMALLINT", "INT", "BIGINT", "BOOL", "[]BIGINT", "JSONB", "XML", "TIMESTAMP".
func (r *rows) ColumnTypeDatabaseTypeName(index int) string {
	switch r.columns[index].OracleType {
	case C.DPI_ORACLE_TYPE_VARCHAR:
		return "VARCHAR2"
	case C.DPI_ORACLE_TYPE_NVARCHAR:
		return "NVARCHAR2"
	case C.DPI_ORACLE_TYPE_CHAR:
		return "CHAR"
	case C.DPI_ORACLE_TYPE_NCHAR:
		return "NCHAR"
	case C.DPI_ORACLE_TYPE_LONG_VARCHAR:
		return "LONG"
	case C.DPI_NATIVE_TYPE_BYTES, C.DPI_ORACLE_TYPE_RAW:
		return "RAW"
	case C.DPI_ORACLE_TYPE_ROWID, C.DPI_NATIVE_TYPE_ROWID:
		return "ROWID"
	case C.DPI_ORACLE_TYPE_LONG_RAW:
		return "LONG RAW"
	case C.DPI_ORACLE_TYPE_NUMBER:
		return "NUMBER"
	case C.DPI_ORACLE_TYPE_NATIVE_FLOAT, C.DPI_NATIVE_TYPE_FLOAT:
		return "FLOAT"
	case C.DPI_ORACLE_TYPE_NATIVE_DOUBLE, C.DPI_NATIVE_TYPE_DOUBLE:
		return "DOUBLE"
	case C.DPI_ORACLE_TYPE_NATIVE_INT, C.DPI_NATIVE_TYPE_INT64:
		return "BINARY_INTEGER"
	case C.DPI_ORACLE_TYPE_NATIVE_UINT, C.DPI_NATIVE_TYPE_UINT64:
		return "BINARY_INTEGER"
	case C.DPI_ORACLE_TYPE_TIMESTAMP, C.DPI_NATIVE_TYPE_TIMESTAMP:
		return "TIMESTAMP"
	case C.DPI_ORACLE_TYPE_TIMESTAMP_TZ:
		return "TIMESTAMP WITH TIMEZONE"
	case C.DPI_ORACLE_TYPE_TIMESTAMP_LTZ:
		return "TIMESTAMP WITH LOCAL TIMEZONE"
	case C.DPI_ORACLE_TYPE_DATE:
		return "DATE"
	case C.DPI_ORACLE_TYPE_INTERVAL_DS, C.DPI_NATIVE_TYPE_INTERVAL_DS:
		return "INTERVAL DAY TO SECOND"
	case C.DPI_ORACLE_TYPE_INTERVAL_YM, C.DPI_NATIVE_TYPE_INTERVAL_YM:
		return "INTERVAL YEAR TO MONTH"
	case C.DPI_ORACLE_TYPE_CLOB:
		return "CLOB"
	case C.DPI_ORACLE_TYPE_NCLOB:
		return "NCLOB"
	case C.DPI_ORACLE_TYPE_BLOB:
		return "BLOB"
	case C.DPI_ORACLE_TYPE_BFILE:
		return "BFILE"
	case C.DPI_ORACLE_TYPE_STMT, C.DPI_NATIVE_TYPE_STMT:
		return "SYS_REFCURSOR"
	case C.DPI_ORACLE_TYPE_BOOLEAN, C.DPI_NATIVE_TYPE_BOOLEAN:
		return "BOOLEAN"
	case C.DPI_ORACLE_TYPE_OBJECT:
		return "OBJECT"
	default:
		return fmt.Sprintf("OTHER[%d]", r.columns[index].OracleType)
	}
}

// ColumnTypeNullable. The nullable value should be true if it is known the column may be null, or false if the column is known to be not nullable. If the column nullability is unknown, ok should be false.

func (r *rows) ColumnTypeNullable(index int) (nullable, ok bool) {
	return r.columns[index].Nullable, true
}

// ColumnTypePrecisionScale returns the precision and scale for decimal types.
// If not applicable, ok should be false.
// The following are examples of returned values for various types:
//
// decimal(38, 4)    (38, 4, true)
// int               (0, 0, false)
// decimal           (math.MaxInt64, math.MaxInt64, true)
func (r *rows) ColumnTypePrecisionScale(index int) (precision, scale int64, ok bool) {
	switch col := r.columns[index]; col.OracleType {
	case
		//C.DPI_ORACLE_TYPE_NATIVE_FLOAT, C.DPI_NATIVE_TYPE_FLOAT,
		//C.DPI_ORACLE_TYPE_NATIVE_DOUBLE, C.DPI_NATIVE_TYPE_DOUBLE,
		//C.DPI_ORACLE_TYPE_NATIVE_INT, C.DPI_NATIVE_TYPE_INT64,
		//C.DPI_ORACLE_TYPE_NATIVE_UINT, C.DPI_NATIVE_TYPE_UINT64,
		C.DPI_ORACLE_TYPE_NUMBER:
		return int64(col.Precision), int64(col.Scale), true
	default:
		return 0, 0, false
	}
}

// ColumnTypeScanType returns the value type that can be used to scan types into.
// For example, the database column type "bigint" this should return "reflect.TypeOf(int64(0))".
func (r *rows) ColumnTypeScanType(index int) reflect.Type {
	switch col := r.columns[index]; col.OracleType {
	case C.DPI_NATIVE_TYPE_BYTES, C.DPI_ORACLE_TYPE_RAW,
		C.DPI_ORACLE_TYPE_ROWID, C.DPI_NATIVE_TYPE_ROWID,
		C.DPI_ORACLE_TYPE_LONG_RAW:
		return reflect.TypeOf([]byte(nil))
	case C.DPI_ORACLE_TYPE_NUMBER:
		switch col.NativeType {
		case C.DPI_NATIVE_TYPE_INT64:
			return reflect.TypeOf(int64(0))
		case C.DPI_NATIVE_TYPE_UINT64:
			return reflect.TypeOf(uint64(0))
		//case C.DPI_NATIVE_TYPE_FLOAT:
		//	return reflect.TypeOf(float32(0))
		//case C.DPI_NATIVE_TYPE_DOUBLE:
		//		return reflect.TypeOf(float64(0))
		default:
			return reflect.TypeOf(Number(""))
		}
	case C.DPI_ORACLE_TYPE_NATIVE_FLOAT, C.DPI_NATIVE_TYPE_FLOAT:
		return reflect.TypeOf(float32(0))
	case C.DPI_ORACLE_TYPE_NATIVE_DOUBLE, C.DPI_NATIVE_TYPE_DOUBLE:
		return reflect.TypeOf(float64(0))
	case C.DPI_ORACLE_TYPE_NATIVE_INT, C.DPI_NATIVE_TYPE_INT64:
		return reflect.TypeOf(int64(0))
	case C.DPI_ORACLE_TYPE_NATIVE_UINT, C.DPI_NATIVE_TYPE_UINT64:
		return reflect.TypeOf(uint64(0))
	case C.DPI_ORACLE_TYPE_TIMESTAMP, C.DPI_NATIVE_TYPE_TIMESTAMP,
		C.DPI_ORACLE_TYPE_TIMESTAMP_TZ, C.DPI_ORACLE_TYPE_TIMESTAMP_LTZ,
		C.DPI_ORACLE_TYPE_DATE:
		return reflect.TypeOf(time.Time{})
	case C.DPI_ORACLE_TYPE_INTERVAL_DS, C.DPI_NATIVE_TYPE_INTERVAL_DS:
		return reflect.TypeOf(time.Duration(0))
	case C.DPI_ORACLE_TYPE_CLOB, C.DPI_ORACLE_TYPE_NCLOB:
		return reflect.TypeOf("")
	case C.DPI_ORACLE_TYPE_BLOB, C.DPI_ORACLE_TYPE_BFILE:
		return reflect.TypeOf([]byte(nil))
	case C.DPI_ORACLE_TYPE_STMT, C.DPI_NATIVE_TYPE_STMT:
		return reflect.TypeOf(&statement{})
	case C.DPI_ORACLE_TYPE_BOOLEAN, C.DPI_NATIVE_TYPE_BOOLEAN:
		return reflect.TypeOf(false)
	default:
		return reflect.TypeOf("")
	}
}

// Next is called to populate the next row of data into
// the provided slice. The provided slice will be the same
// size as the Columns() are wide.
//
// Next should return io.EOF when there are no more rows.
func (r *rows) Next(dest []driver.Value) error {
	if r.err != nil {
		return r.err
	}
	if r.finished {
		_ = r.Close()
		return io.EOF
	}
	if len(dest) != len(r.columns) {
		return errors.Errorf("column count mismatch: we have %d columns, but given %d destination", len(r.columns), len(dest))
	}
	if r.fetched == 0 {
		var moreRows C.int
		if C.dpiStmt_fetchRows(r.dpiStmt, C.uint32_t(r.statement.FetchRowCount()), &r.bufferRowIndex, &r.fetched, &moreRows) == C.DPI_FAILURE {
			return errors.Wrap(r.getError(), "Next")
		}
		if Log != nil {
			Log("msg", "fetched", "bri", r.bufferRowIndex, "fetched", r.fetched, "moreRows", moreRows, "len(data)", len(r.data), "cols", len(r.columns))
		}
		if r.fetched == 0 {
			r.finished = moreRows == 0
			_ = r.Close()
			return io.EOF
		}
		if r.data == nil {
			r.data = make([][]C.dpiData, len(r.columns))
			for i := range r.columns {
				var n C.uint32_t
				var data *C.dpiData
				if C.dpiVar_getReturnedData(r.vars[i], 0, &n, &data) == C.DPI_FAILURE {
					return errors.Wrapf(r.getError(), "getReturnedData[%d]", i)
				}
				r.data[i] = (*[maxArraySize]C.dpiData)(unsafe.Pointer(data))[:n:n]
				//fmt.Printf("data %d=%+v\n%+v\n", n, data, r.data[i][0])
			}
		}

	}
	//fmt.Printf("data=%#v\n", r.data)

	//fmt.Printf("bri=%d fetched=%d\n", r.bufferRowIndex, r.fetched)
	//fmt.Printf("data=%#v\n", r.data[0][r.bufferRowIndex])
	//fmt.Printf("VC=%d\n", C.DPI_ORACLE_TYPE_VARCHAR)
	for i, col := range r.columns {
		typ := col.OracleType
		d := &r.data[i][r.bufferRowIndex]
		isNull := d.isNull == 1
		if Log != nil {
			Log("msg", "Next", "i", i, "row", r.bufferRowIndex, "typ", typ, "null", isNull) //, "data", fmt.Sprintf("%+v", d), "typ", typ)
		}

		switch typ {
		case C.DPI_ORACLE_TYPE_VARCHAR, C.DPI_ORACLE_TYPE_NVARCHAR,
			C.DPI_ORACLE_TYPE_CHAR, C.DPI_ORACLE_TYPE_NCHAR,
			C.DPI_ORACLE_TYPE_LONG_VARCHAR:
			//fmt.Printf("CHAR\n")
			if isNull {
				dest[i] = ""
				continue
			}
			b := C.dpiData_getBytes(d)
			if b.length == 0 {
				dest[i] = ""
				continue
			}
			dest[i] = C.GoStringN(b.ptr, C.int(b.length))

		case C.DPI_ORACLE_TYPE_NUMBER:
			if isNull {
				//if Log != nil { Log("msg", "null", "i", i, "T", fmt.Sprintf("%T", dest[i]), "type", reflect.TypeOf(dest[i])) }
				dest[i] = nil
				continue
			}
			switch col.NativeType {
			case C.DPI_NATIVE_TYPE_INT64:
				dest[i] = int64(C.dpiData_getInt64(d))
			case C.DPI_NATIVE_TYPE_UINT64:
				dest[i] = uint64(C.dpiData_getUint64(d))
			case C.DPI_NATIVE_TYPE_FLOAT:
				//dest[i] = float32(C.dpiData_getFloat(d))
				dest[i] = printFloat(float64(C.dpiData_getFloat(d)))
			case C.DPI_NATIVE_TYPE_DOUBLE:
				//dest[i] = float64(C.dpiData_getDouble(d))
				dest[i] = printFloat(float64(C.dpiData_getDouble(d)))
			default:
				b := C.dpiData_getBytes(d)
				dest[i] = Number(C.GoStringN(b.ptr, C.int(b.length)))
				if Log != nil {
					Log("msg", "b", "i", i, "ptr", b.ptr, "length", b.length, "typ", col.NativeType, "int64", C.dpiData_getInt64(d), "dest", dest[i])
				}
			}
			if Log != nil {
				Log("msg", "num", "t", col.NativeType, "i", i, "dest", fmt.Sprintf("%T %+v", dest[i], dest[i]))
			}

		case C.DPI_ORACLE_TYPE_ROWID, C.DPI_NATIVE_TYPE_ROWID,
			C.DPI_ORACLE_TYPE_RAW, C.DPI_ORACLE_TYPE_LONG_RAW:
			if isNull {
				dest[i] = nil
				continue
			}
			b := C.dpiData_getBytes(d)
			if b.length == 0 {
				dest[i] = []byte{}
				continue
			}
			dest[i] = C.GoBytes(unsafe.Pointer(b.ptr), C.int(b.length))
		case C.DPI_ORACLE_TYPE_NATIVE_FLOAT, C.DPI_NATIVE_TYPE_FLOAT:
			if isNull {
				dest[i] = nil
				continue
			}
			dest[i] = float32(C.dpiData_getFloat(d))
		case C.DPI_ORACLE_TYPE_NATIVE_DOUBLE, C.DPI_NATIVE_TYPE_DOUBLE:
			if isNull {
				dest[i] = nil
				continue
			}
			dest[i] = float64(C.dpiData_getDouble(d))
		case C.DPI_ORACLE_TYPE_NATIVE_INT, C.DPI_NATIVE_TYPE_INT64:
			if isNull {
				dest[i] = nil
				continue
			}
			dest[i] = int64(C.dpiData_getInt64(d))
		case C.DPI_ORACLE_TYPE_NATIVE_UINT, C.DPI_NATIVE_TYPE_UINT64:
			if isNull {
				dest[i] = nil
				continue
			}
			dest[i] = uint64(C.dpiData_getUint64(d))
		case C.DPI_ORACLE_TYPE_TIMESTAMP,
			C.DPI_ORACLE_TYPE_TIMESTAMP_TZ, C.DPI_ORACLE_TYPE_TIMESTAMP_LTZ,
			C.DPI_NATIVE_TYPE_TIMESTAMP,
			C.DPI_ORACLE_TYPE_DATE:
			if isNull {
				dest[i] = time.Time{}
				continue
			}
			ts := C.dpiData_getTimestamp(d)
			tz := r.conn.timeZone
			if col.OracleType != C.DPI_ORACLE_TYPE_TIMESTAMP && col.OracleType != C.DPI_ORACLE_TYPE_DATE {
				tz = timeZoneFor(ts.tzHourOffset, ts.tzMinuteOffset)
			}
			dest[i] = time.Date(int(ts.year), time.Month(ts.month), int(ts.day), int(ts.hour), int(ts.minute), int(ts.second), int(ts.fsecond), tz)
		case C.DPI_ORACLE_TYPE_INTERVAL_DS, C.DPI_NATIVE_TYPE_INTERVAL_DS:
			if isNull {
				dest[i] = nil
				continue
			}
			ds := C.dpiData_getIntervalDS(d)
			dest[i] = time.Duration(ds.days)*24*time.Hour +
				time.Duration(ds.hours)*time.Hour +
				time.Duration(ds.minutes)*time.Minute +
				time.Duration(ds.seconds)*time.Second +
				time.Duration(ds.fseconds)
		case C.DPI_ORACLE_TYPE_INTERVAL_YM, C.DPI_NATIVE_TYPE_INTERVAL_YM:
			if isNull {
				dest[i] = nil
				continue
			}
			ym := C.dpiData_getIntervalYM(d)
			dest[i] = fmt.Sprintf("%dy%dm", ym.years, ym.months)

		case C.DPI_ORACLE_TYPE_CLOB, C.DPI_ORACLE_TYPE_NCLOB,
			C.DPI_ORACLE_TYPE_BLOB,
			C.DPI_ORACLE_TYPE_BFILE,
			C.DPI_NATIVE_TYPE_LOB:
			isClob := typ == C.DPI_ORACLE_TYPE_CLOB || typ == C.DPI_ORACLE_TYPE_NCLOB
			if isNull {
				if isClob && r.ClobAsString() {
					dest[i] = ""
				} else {
					dest[i] = nil
				}
				continue
			}
			rdr := &dpiLobReader{dpiLob: C.dpiData_getLOB(d), conn: r.conn, IsClob: isClob}
			if isClob && r.ClobAsString() {
				sb := stringBuilders.Get()
				if _, err := io.Copy(sb, rdr); err != nil {
					stringBuilders.Put(sb)
					return err
				}
				dest[i] = sb.String()
				stringBuilders.Put(sb)
				continue
			}
			dest[i] = &Lob{Reader: rdr, IsClob: rdr.IsClob}

		case C.DPI_ORACLE_TYPE_STMT, C.DPI_NATIVE_TYPE_STMT:
			if isNull {
				dest[i] = nil
				continue
			}
			st := &statement{conn: r.conn, dpiStmt: C.dpiData_getStmt(d),
				stmtOptions: r.statement.stmtOptions, // inherit parent statement's options
			}
			var colCount C.uint32_t
			if C.dpiStmt_getNumQueryColumns(st.dpiStmt, &colCount) == C.DPI_FAILURE {
				return errors.Wrap(r.getError(), "getNumQueryColumns")
			}
			st.Lock()
			r2, err := st.openRows(int(colCount))
			st.Unlock()
			if err != nil {
				return err
			}
			dest[i] = r2

		case C.DPI_ORACLE_TYPE_BOOLEAN, C.DPI_NATIVE_TYPE_BOOLEAN:
			if isNull {
				dest[i] = nil
				continue
			}
			dest[i] = C.dpiData_getBool(d) == 1

		case C.DPI_ORACLE_TYPE_OBJECT: //Default type used for named type columns in the database. Data is transferred to/from Oracle in Oracle's internal format.
			if isNull {
				dest[i] = nil
				continue
			}
			o, err := wrapObject(r.drv, col.ObjectType, C.dpiData_getObject(d))
			if err != nil {
				return err
			}
			dest[i] = o

		default:
			return errors.Errorf("unsupported column type %d", typ)
		}

		//fmt.Printf("dest[%d]=%#v\n", i, dest[i])
	}
	r.bufferRowIndex++
	r.fetched--

	return nil
}

var _ = driver.Rows((*directRow)(nil))

type directRow struct {
	args   []string
	result []interface{}
	query  string
	conn   *conn
}

func (dr *directRow) Columns() []string {
	if Log != nil {
		Log("directRow", "Columns")
	}
	switch dr.query {
	case getConnection:
		return []string{"CONNECTION"}
	}
	return nil
}

// Close closes the rows iterator.
func (dr *directRow) Close() error {
	dr.conn = nil
	dr.query = ""
	dr.args = nil
	dr.result = nil
	return nil
}

// Next is called to populate the next row of data into
// the provided slice. The provided slice will be the same
// size as the Columns() are wide.
//
// Next should return io.EOF when there are no more rows.
func (dr *directRow) Next(dest []driver.Value) error {
	if Log != nil {
		Log("directRow", "Next", "query", dr.query, "dest", dest)
	}
	switch dr.query {
	case getConnection:
		*(dest[0].(*interface{})) = dr.result[0]
	}
	return nil
}

func (r *rows) getImplicitResult() {
	if r == nil || r.nextRsErr != nil {
		return
	}
	// use the original statement for the NextResultSet call.
	st := r.origSt
	if st == nil {
		st = r.statement
		r.origSt = st
	}
	if C.dpiStmt_getImplicitResult(st.dpiStmt, &r.nextRs) == C.DPI_FAILURE {
		r.nextRsErr = errors.Wrap(r.getError(), "getImplicitResult")
	}
}
func (r *rows) HasNextResultSet() bool {
	if r == nil || r.statement == nil || r.conn == nil {
		return false
	}
	if r.nextRs != nil {
		return true
	}
	if !((r.conn.Client.Version > 12 || r.conn.Client.Version == 12 && r.conn.Client.Release >= 1) &&
		(r.conn.Server.Version > 12 || r.conn.Server.Version == 12 && r.conn.Server.Release >= 1)) {
		return false
	}
	r.getImplicitResult()
	return r.nextRs != nil
}
func (r *rows) NextResultSet() error {
	if r.nextRs == nil {
		r.getImplicitResult()
		if r.nextRsErr != nil {
			return r.nextRsErr
		}
		if r.nextRs == nil {
			return errors.Wrap(io.EOF, "getImplicitResult")
		}
	}
	st := &statement{conn: r.conn, dpiStmt: r.nextRs}

	var n C.uint32_t
	if C.dpiStmt_getNumQueryColumns(st.dpiStmt, &n) == C.DPI_FAILURE {
		return errors.Wrapf(io.EOF, "getNumQueryColumns: %v", r.getError())
	}
	// keep the originam statement for the succeeding NextResultSet calls.
	nr, err := st.openRows(int(n))
	if err != nil {
		return err
	}
	nr.origSt = r.origSt
	if nr.origSt == nil {
		nr.origSt = r.statement
	}
	*r = *nr
	return nil
}

func printFloat(f float64) string {
	var a [40]byte
	b := strconv.AppendFloat(a[:0], f, 'f', -1, 64)
	i := bytes.IndexByte(b, '.')
	if i < 0 {
		return string(b)
	}
	for j := i + 1; j < len(b); j++ {
		if b[j] != '0' {
			return string(b)
		}
	}
	return string(b[:i])
}
