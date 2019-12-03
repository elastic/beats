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

const int sizeof_dpiData = sizeof(void);
*/
import "C"
import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"sync"
	"time"
	"unsafe"

	"github.com/pkg/errors"
)

type stmtOptions struct {
	fetchRowCount       int
	arraySize           int
	callTimeout         time.Duration
	execMode            C.dpiExecMode
	plSQLArrays         bool
	lobAsReader         bool
	magicTypeConversion bool
}

func (o stmtOptions) ExecMode() C.dpiExecMode {
	if o.execMode == 0 {
		return C.DPI_MODE_EXEC_DEFAULT
	}
	return o.execMode
}

func (o stmtOptions) ArraySize() int {
	if o.arraySize <= 0 || o.arraySize > 32<<10 {
		return DefaultArraySize
	}
	return o.arraySize
}
func (o stmtOptions) FetchRowCount() int {
	if o.fetchRowCount <= 0 {
		return DefaultFetchRowCount
	}
	return o.fetchRowCount
}
func (o stmtOptions) PlSQLArrays() bool { return o.plSQLArrays }

func (o stmtOptions) ClobAsString() bool { return !o.lobAsReader }
func (o stmtOptions) LobAsReader() bool  { return o.lobAsReader }

func (o stmtOptions) MagicTypeConversion() bool { return o.magicTypeConversion }

// Option holds statement options.
type Option func(*stmtOptions)

// PlSQLArrays is to signal that the slices given in arguments of Exec to
// be left as is - the default is to treat them as arguments for ExecMany.
var PlSQLArrays Option = func(o *stmtOptions) { o.plSQLArrays = true }

// FetchRowCount returns an option to set the rows to be fetched, overriding DefaultFetchRowCount.
func FetchRowCount(rowCount int) Option {
	if rowCount <= 0 {
		return nil
	}
	return func(o *stmtOptions) { o.fetchRowCount = rowCount }
}

// ArraySize returns an option to set the array size to be used, overriding DefaultArraySize.
func ArraySize(arraySize int) Option {
	if arraySize <= 0 {
		return nil
	}
	return func(o *stmtOptions) { o.arraySize = arraySize }
}
func parseOnly(o *stmtOptions) { o.execMode = C.DPI_MODE_EXEC_PARSE_ONLY }

// ParseOnly returns an option to set the ExecMode to only Parse.
func ParseOnly() Option {
	return parseOnly
}

func describeOnly(o *stmtOptions) { o.execMode = C.DPI_MODE_EXEC_DESCRIBE_ONLY }

// ClobAsString returns an option to force fetching CLOB columns as strings.
//
// DEPRECATED.
func ClobAsString() Option { return func(o *stmtOptions) { o.lobAsReader = false } }

// LobAsReader is an option to set query columns of CLOB/BLOB to be returned as a Lob.
//
// LOB as a reader and writer is not the most performant at all. Yes, OCI
// and ODPI-C provide a way to retrieve this data directly. Effectively,
// all you need to do is tell ODPI-C that you want a "long string" or "long
// raw" returned. You can do that by telling ODPI-C you want a variable
// with oracleTypeNum=DPI_ORACLE_TYPE_LONG_VARCHAR or
// DPI_ORACLE_TYPE_LONG_RAW and nativeTypeNum=DPI_NATIVE_TYPE_BYTES. ODPI-C
// will handle all of the dynamic fetching and allocation that is required.
// :-) You can also use DPI_ORACLE_TYPE_VARCHAR and DPI_ORACLE_TYPE_RAW as
// long as you set the size > 32767 -- whichever way you wish to use.
//
// With the use of LOBs, there is one round-trip to get the LOB locators,
// then a round-trip for each read() that is performed. If you request the
// length there is another round-trip required. So if you fetch 100 rows
// with 2 CLOB columns, that means you get 401 round-trips. Using
// string/[]bytes directly means only one round trip. So you can see that
// if your database is remote with high latency you can have a significant
// performance penalty!
func LobAsReader() Option { return func(o *stmtOptions) { o.lobAsReader = true } }

// MagicTypeConversion returns an option to force converting named scalar types (e.g. "type underlying int64") to their scalar underlying type.
func MagicTypeConversion() Option {
	return func(o *stmtOptions) { o.magicTypeConversion = true }
}

// CallTimeout sets the round-trip timeout (OCI_ATTR_CALL_TIMEOUT).
//
// See https://docs.oracle.com/en/database/oracle/oracle-database/18/lnoci/handle-and-descriptor-attributes.html#GUID-D8EE68EB-7E38-4068-B06E-DF5686379E5E
func CallTimeout(d time.Duration) Option {
	return func(o *stmtOptions) { o.callTimeout = d }
}

const minChunkSize = 1 << 16

var _ = driver.Stmt((*statement)(nil))
var _ = driver.StmtQueryContext((*statement)(nil))
var _ = driver.StmtExecContext((*statement)(nil))
var _ = driver.NamedValueChecker((*statement)(nil))

type statement struct {
	stmtOptions
	columns  []Column
	isSlice  []bool
	gets     []dataGetter
	dests    []interface{}
	data     [][]C.dpiData
	vars     []*C.dpiVar
	varInfos []varInfo
	query    string
	sync.Mutex
	arrLen int
	*conn
	dpiStmt     *C.dpiStmt
	isReturning bool
}
type dataGetter func(v interface{}, data []C.dpiData) error

// Close closes the statement.
//
// As of Go 1.1, a Stmt will not be closed if it's in use
// by any queries.
func (st *statement) Close() error {
	if st == nil {
		return nil
	}
	st.Lock()
	defer st.Unlock()

	return st.close()
}
func (st *statement) close() error {
	if st == nil {
		return nil
	}
	dpiStmt := st.dpiStmt
	c := st.conn
	st.cleanup()

	var si C.dpiStmtInfo
	if dpiStmt != nil &&
		C.dpiStmt_getInfo(dpiStmt, &si) != C.DPI_FAILURE && // this is just to check the validity of dpiStmt, to avoid SIGSEGV
		C.dpiStmt_release(dpiStmt) != C.DPI_FAILURE {
		return nil
	}
	if c == nil {
		return driver.ErrBadConn
	}
	return errors.Wrap(c.getError(), "statement/dpiStmt_release")
}

func (st *statement) cleanup() error {
	if st == nil {
		return nil
	}

	for _, v := range st.vars {
		C.dpiVar_release(v)
	}
	st.data = nil
	st.vars = nil
	st.varInfos = nil
	st.gets = nil
	st.dests = nil
	st.columns = nil
	st.dpiStmt = nil
	c := st.conn
	st.conn = nil

	if c == nil {
		return driver.ErrBadConn
	}
	return errors.Wrap(c.getError(), "statement/dpiStmt_release")
}

// Exec executes a query that doesn't return rows, such
// as an INSERT or UPDATE.
//
// Deprecated: Drivers should implement StmtExecContext instead (or additionally).
func (st *statement) Exec(args []driver.Value) (driver.Result, error) {
	nargs := make([]driver.NamedValue, len(args))
	for i, arg := range args {
		nargs[i].Ordinal = i + 1
		nargs[i].Value = arg
	}
	return st.ExecContext(context.Background(), nargs)
}

// Query executes a query that may return rows, such as a
// SELECT.
//
// Deprecated: Drivers should implement StmtQueryContext instead (or additionally).
func (st *statement) Query(args []driver.Value) (driver.Rows, error) {
	nargs := make([]driver.NamedValue, len(args))
	for i, arg := range args {
		nargs[i].Ordinal = i + 1
		nargs[i].Value = arg
	}
	return st.QueryContext(context.Background(), nargs)
}

// ExecContext executes a query that doesn't return rows, such as an INSERT or UPDATE.
//
// ExecContext must honor the context timeout and return when it is canceled.
func (st *statement) ExecContext(ctx context.Context, args []driver.NamedValue) (res driver.Result, err error) {
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	Log := ctxGetLog(ctx)

	closeIfBadConn := func(err error) error {
		if err != nil && err == driver.ErrBadConn {
			if Log != nil {
				Log("error", driver.ErrBadConn)
			}
			st.close()
		}
		return err
	}

	st.Lock()
	defer st.Unlock()
	if st.dpiStmt == nil && st.query == getConnection {
		*(args[0].Value.(sql.Out).Dest.(*interface{})) = st.conn
		return driver.ResultNoRows, nil
	}
	st.isReturning = false

	st.conn.RLock()
	defer st.conn.RUnlock()

	// bind variables
	if err = st.bindVars(args, Log); err != nil {
		return nil, closeIfBadConn(err)
	}

	mode := st.ExecMode()
	//fmt.Printf("%p.%p: inTran? %t\n%s\n", st.conn, st, st.inTransaction, st.query)
	if !st.inTransaction {
		mode |= C.DPI_MODE_EXEC_COMMIT_ON_SUCCESS
	}
	st.setCallTimeout(ctx)

	done := make(chan error, 1)
	// execute
	go func() {
		defer close(done)
	Loop:
		for i := 0; i < 3; i++ {
			if err = ctx.Err(); err != nil {
				done <- err
				return
			}
			if !st.PlSQLArrays() && st.arrLen > 0 {
				if Log != nil {
					Log("C", "dpiStmt_executeMany", "mode", mode, "len", st.arrLen)
				}
				if C.dpiStmt_executeMany(st.dpiStmt, mode, C.uint32_t(st.arrLen)) == C.DPI_FAILURE {
					if err = ctx.Err(); err == nil {
						err = st.getError()
					}
				}
			} else {
				var colCount C.uint32_t
				if Log != nil {
					Log("C", "dpiStmt_execute", "mode", mode, "colCount", colCount)
				}
				if C.dpiStmt_execute(st.dpiStmt, mode, &colCount) == C.DPI_FAILURE {
					if err = ctx.Err(); err == nil {
						err = st.getError()
					}
				}
			}
			if Log != nil {
				Log("msg", "st.Execute", "error", err)
			}
			if err == nil {
				var info C.dpiStmtInfo
				if C.dpiStmt_getInfo(st.dpiStmt, &info) == C.DPI_FAILURE {
					err = errors.Wrap(st.getError(), "getInfo")
				}
				st.isReturning = info.isReturning != 0
				return
			}
			cdr, ok := errors.Cause(err).(interface {
				Code() int
			})
			if !ok {
				break
			}
			switch code := cdr.Code(); code {
			// ORA-04068: "existing state of packages has been discarded"
			case 4061, 4065, 4068:
				if Log != nil {
					Log("msg", "retry", "ora", code)
				}
				continue Loop
			}
			break
		}
		done <- maybeBadConn(errors.Wrapf(err, "dpiStmt_execute(mode=%d arrLen=%d)", mode, st.arrLen))
	}()

	select {
	case err = <-done:
		if err != nil {
			return nil, closeIfBadConn(err)
		}
	case <-ctx.Done():
		// select again to avoid race condition if both are done
		select {
		case err = <-done:
			if err != nil {
				return nil, closeIfBadConn(err)
			}
		case <-ctx.Done():
			if Log != nil {
				Log("msg", "BREAK statement")
			}
			_ = st.Break()
			st.cleanup()
			return nil, driver.ErrBadConn
		}
	}

	if Log != nil {
		Log("gets", st.gets, "dests", st.dests)
	}
	for i, get := range st.gets {
		if get == nil {
			continue
		}
		if st.isReturning {
			var n C.uint32_t
			data := &st.data[i][0]
			if C.dpiVar_getReturnedData(st.vars[i], 0, &n, &data) == C.DPI_FAILURE {
				err = st.getError()
				return nil, errors.Wrapf(closeIfBadConn(err), "%d.getReturnedData", i)
			}
			if n == 0 {
				st.data[i] = st.data[i][:0]
			} else {
				st.data[i] = (*(*[maxArraySize]C.dpiData)(unsafe.Pointer(data)))[:int(n):int(n)]
			}
		}
		dest := st.dests[i]
		if !st.isSlice[i] {
			if err = get(dest, st.data[i]); err != nil {
				if Log != nil {
					Log("get", i, "error", err)
				}
				return nil, errors.Wrapf(closeIfBadConn(err), "%d. get[%d]", i, 0)
			}
			continue
		}
		var n C.uint32_t = 1
		if C.dpiVar_getNumElementsInArray(st.vars[i], &n) == C.DPI_FAILURE {
			err = st.getError()
			if Log != nil {
				Log("msg", "getNumElementsInArray", "i", i, "error", err)
			}
			return nil, errors.Wrapf(closeIfBadConn(err), "%d.getNumElementsInArray", i)
		}
		//fmt.Printf("i=%d dest=%T %#v\n", i, dest, dest)
		if err = get(dest, st.data[i][:n]); err != nil {
			if Log != nil {
				Log("msg", "get", "i", i, "n", n, "error", err)
			}
			return nil, errors.Wrapf(closeIfBadConn(err), "%d. get", i)
		}
	}
	var count C.uint64_t
	if C.dpiStmt_getRowCount(st.dpiStmt, &count) == C.DPI_FAILURE {
		return nil, nil
	}
	return driver.RowsAffected(count), nil
}

// QueryContext executes a query that may return rows, such as a SELECT.
//
// QueryContext must honor the context timeout and return when it is canceled.
func (st *statement) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	Log := ctxGetLog(ctx)

	closeIfBadConn := func(err error) error {
		if err != nil && err == driver.ErrBadConn {
			st.close()
		}
		return err
	}

	st.Lock()
	defer st.Unlock()
	st.isReturning = false
	st.conn.RLock()
	defer st.conn.RUnlock()

	switch st.query {
	case getConnection:
		if Log != nil {
			Log("msg", "QueryContext", "args", args)
		}
		return &directRow{conn: st.conn, query: st.query, result: []interface{}{st.conn}}, nil

	case wrapResultset:
		if Log != nil {
			Log("msg", "QueryContext", "args", args)
		}
		return args[0].Value.(driver.Rows), nil
	}

	//fmt.Printf("QueryContext(%+v)\n", args)
	// bind variables
	if err := st.bindVars(args, Log); err != nil {
		return nil, closeIfBadConn(err)
	}

	// execute
	var colCount C.uint32_t
	done := make(chan error, 1)
	go func() {
		var err error
		defer close(done)
		for i := 0; i < 3; i++ {
			if err = ctx.Err(); err != nil {
				done <- err
				return
			}
			st.setCallTimeout(ctx)
			if C.dpiStmt_execute(st.dpiStmt, st.ExecMode(), &colCount) != C.DPI_FAILURE {
				break
			}
			if err = ctx.Err(); err == nil {
				err = st.getError()
				if c, ok := err.(interface{ Code() int }); ok && c.Code() != 4068 {
					break
				}
			}
		}
		done <- maybeBadConn(errors.Wrap(err, "dpiStmt_execute"))
	}()

	select {
	case err := <-done:
		if err != nil {
			return nil, closeIfBadConn(err)
		}
	case <-ctx.Done():
		select {
		case err := <-done:
			if err != nil {
				return nil, closeIfBadConn(err)
			}
		case <-ctx.Done():
			if Log != nil {
				Log("msg", "BREAK query")
			}
			_ = st.Break()
			st.cleanup()
			return nil, driver.ErrBadConn
		}
	}
	rows, err := st.openRows(int(colCount))
	return rows, closeIfBadConn(err)
}

// NumInput returns the number of placeholder parameters.
//
// If NumInput returns >= 0, the sql package will sanity check
// argument counts from callers and return errors to the caller
// before the statement's Exec or Query methods are called.
//
// NumInput may also return -1, if the driver doesn't know
// its number of placeholders. In that case, the sql package
// will not sanity check Exec or Query argument counts.
func (st *statement) NumInput() int {
	if st.query == wrapResultset {
		return 1
	}
	if st.dpiStmt == nil {
		switch st.query {
		case getConnection, wrapResultset:
			return 1
		}
		return 0
	}

	if !go10 {
		return -1
	}

	st.Lock()
	defer st.Unlock()
	var cnt C.uint32_t
	//defer func() { fmt.Printf("%p.NumInput=%d (%q)\n", st, cnt, st.query) }()
	if C.dpiStmt_getBindCount(st.dpiStmt, &cnt) == C.DPI_FAILURE {
		return -1
	}
	if cnt < 2 { // 1 can't decrease...
		return int(cnt)
	}
	names := make([]*C.char, int(cnt))
	lengths := make([]C.uint32_t, int(cnt))
	if C.dpiStmt_getBindNames(st.dpiStmt, &cnt, &names[0], &lengths[0]) == C.DPI_FAILURE {
		return -1
	}
	//fmt.Printf("%p.NumInput=%d\n", st, cnt)

	// return the number of *unique* arguments
	return int(cnt)
}

func (st *statement) setCallTimeout(ctx context.Context) {
	if st.callTimeout != 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, st.callTimeout)
		_ = cancel
	}
	st.conn.setCallTimeout(ctx)
}

type argInfo struct {
	objType     *C.dpiObjectType
	set         dataSetter
	bufSize     int
	typ         C.dpiOracleTypeNum
	natTyp      C.dpiNativeTypeNum
	isIn, isOut bool
}

// bindVars binds the given args into new variables.
func (st *statement) bindVars(args []driver.NamedValue, Log logFunc) error {
	if Log != nil {
		Log("enter", "bindVars", "args", args)
	}
	if cap(st.vars) < len(args) || cap(st.varInfos) < len(args) {
		for i, v := range st.vars {
			if v != nil {
				C.dpiVar_release(v)
				st.vars[i], st.varInfos[i] = nil, varInfo{}
			}
		}
	}
	var named bool
	if cap(st.vars) < len(args) {
		st.vars = make([]*C.dpiVar, len(args))
	} else {
		st.vars = st.vars[:len(args)]
	}
	if cap(st.varInfos) < len(args) {
		st.varInfos = make([]varInfo, len(args))
	} else {
		st.varInfos = st.varInfos[:len(args)]
	}
	if cap(st.data) < len(args) {
		st.data = make([][]C.dpiData, len(args))
	} else {
		st.data = st.data[:len(args)]
	}
	if cap(st.gets) < len(args) {
		st.gets = make([]dataGetter, len(args))
	} else {
		st.gets = st.gets[:len(args)]
	}
	if cap(st.dests) < len(args) {
		st.dests = make([]interface{}, len(args))
	} else {
		st.dests = st.dests[:len(args)]
	}
	if cap(st.isSlice) < len(args) {
		st.isSlice = make([]bool, len(args))
	} else {
		st.isSlice = st.isSlice[:len(args)]
	}

	rArgs := make([]reflect.Value, len(args))
	minArrLen, maxArrLen := -1, -1

	st.arrLen = minArrLen
	maxArraySize := st.ArraySize()

	infos := make([]argInfo, len(args))
	//fmt.Printf("bindVars %d\n", len(args))
	for i, a := range args {
		st.gets[i] = nil
		st.dests[i] = nil
		if !named {
			named = a.Name != ""
		}
		info := &(infos[i])
		info.isIn = true
		value := a.Value
		if out, ok := value.(sql.Out); ok {
			if !st.PlSQLArrays() && st.arrLen > 1 {
				st.arrLen = maxArraySize
			}
			info.isIn, info.isOut = out.In, true
			value = out.Dest
		}
		st.dests[i] = value
		rv := reflect.ValueOf(value)
		if info.isOut {
			if rv.IsNil() {
				fmt.Printf("%d. v=%T %#v kind=%s\n", i, value, value, reflect.ValueOf(value).Kind())
			}
			if rv.Kind() == reflect.Ptr {
				rv = rv.Elem()
				value = rv.Interface()
			}
		}
		st.isSlice[i] = false
		rArgs[i] = rv
		if rv.Kind() == reflect.Ptr {
			// deref in rArgs, but NOT value!
			rArgs[i] = rv.Elem()
		}
		if _, isByteSlice := value.([]byte); !isByteSlice {
			st.isSlice[i] = rArgs[i].Kind() == reflect.Slice
			if !st.PlSQLArrays() && st.isSlice[i] {
				n := rArgs[i].Len()
				if minArrLen == -1 || n < minArrLen {
					minArrLen = n
				}
				if maxArrLen == -1 || n > maxArrLen {
					maxArrLen = n
				}
			}
		}

		if Log != nil {
			Log("msg", "bindVars", "i", i, "in", info.isIn, "out", info.isOut, "value", fmt.Sprintf("%T %#v", st.dests[i], st.dests[i]))
		}
	}

	if maxArrLen > maxArraySize {
		if st.arrLen == maxArraySize {
			st.arrLen = maxArrLen
		}
		maxArraySize = maxArrLen
	}
	doManyCount := 1
	doExecMany := !st.PlSQLArrays()
	if doExecMany {
		if minArrLen != -1 && minArrLen != maxArrLen {
			return errors.Errorf("PlSQLArrays is not set, but has different lengthed slices (min=%d < %d=max)", minArrLen, maxArrLen)
		}
		st.arrLen = minArrLen
		if doExecMany = st.arrLen > 1; doExecMany {
			doManyCount = st.arrLen
		}
	}
	if Log != nil {
		Log("doManyCount", doManyCount, "arrLen", st.arrLen, "doExecMany", doExecMany, "minArrLen", "maxArrLen")
	}

	for i := range args {
		info := &(infos[i])
		value := st.dests[i]

		var err error
		if value, err = st.bindVarTypeSwitch(info, &(st.gets[i]), value); err != nil {
			return errors.Wrapf(err, "%d. arg", i+1)
		}

		var rv reflect.Value
		if st.isSlice[i] {
			rv = reflect.ValueOf(value)
		}

		n := doManyCount
		if st.PlSQLArrays() && st.isSlice[i] {
			n = rv.Len()
			if info.isOut {
				n = rv.Cap()
			}
		}
		if Log != nil {
			Log("msg", "newVar", "i", i, "plSQLArrays", st.PlSQLArrays(), "typ", int(info.typ), "natTyp", int(info.natTyp), "sliceLen", n, "bufSize", info.bufSize, "isSlice", st.isSlice[i])
		}
		//i, st.PlSQLArrays(), info.typ, info.natTyp dataSliceLen, info.bufSize)
		vi := varInfo{
			IsPLSArray: st.PlSQLArrays() && st.isSlice[i],
			Typ:        info.typ, NatTyp: info.natTyp,
			SliceLen: n, BufSize: info.bufSize,
			ObjectType: info.objType,
		}
		if vi.IsPLSArray && vi.SliceLen > maxArraySize {
			return errors.Errorf("maximum array size allowed is %d", maxArraySize)
		}
		if st.vars[i] == nil || st.data[i] == nil || st.varInfos[i] != vi {
			if st.vars[i] != nil {
				C.dpiVar_release(st.vars[i])
				st.vars[i] = nil
			}
			if st.vars[i], st.data[i], err = st.newVar(vi); err != nil {
				return errors.WithMessage(err, fmt.Sprintf("%d", i))
			}
			st.varInfos[i] = vi
		}

		// Have to setNumElementsInArray for the actual lengths for PL/SQL arrays
		dv, data := st.vars[i], st.data[i]
		if !info.isIn {
			if st.PlSQLArrays() {
				if Log != nil {
					Log("C", "dpiVar_setNumElementsInArray", "i", i, "n", 0)
				}
				if C.dpiVar_setNumElementsInArray(dv, C.uint32_t(0)) == C.DPI_FAILURE {
					return errors.Wrapf(st.getError(), "setNumElementsInArray[%d](%d)", i, 0)
				}
			}
			continue
		}

		if !st.isSlice[i] {
			if Log != nil {
				Log("msg", "set", "i", i, "value", fmt.Sprintf("%T=%#v", value, value))
			}
			if err := info.set(dv, data[:1], value); err != nil {
				return errors.Wrapf(err, "set(data[%d][%d], %#v (%T))", i, 0, value, value)
			}
			continue
		}

		if st.PlSQLArrays() {
			n = rv.Len()

			if Log != nil {
				Log("C", "dpiVar_setNumElementsInArray", "i", i, "n", n)
			}
			if C.dpiVar_setNumElementsInArray(dv, C.uint32_t(n)) == C.DPI_FAILURE {
				return errors.Wrapf(st.getError(), "%+v.setNumElementsInArray[%d](%d)", dv, i, n)
			}
		}
		//fmt.Println("n:", len(st.data[i]))
		if err := info.set(dv, data, value); err != nil {
			return err
		}
	}

	if !named {
		for i, v := range st.vars {
			//if Log != nil {Log("C", "dpiStmt_bindByPos", "dpiStmt", st.dpiStmt, "i", i, "v", v) }
			if C.dpiStmt_bindByPos(st.dpiStmt, C.uint32_t(i+1), v) == C.DPI_FAILURE {
				return errors.Wrapf(st.getError(), "bindByPos[%d]", i)
			}
		}
		return nil
	}
	for i, a := range args {
		name := a.Name
		if name == "" {
			name = strconv.Itoa(a.Ordinal)
		}
		//fmt.Printf("bindByName(%q)\n", name)
		cName := C.CString(name)
		res := C.dpiStmt_bindByName(st.dpiStmt, cName, C.uint32_t(len(name)), st.vars[i])
		C.free(unsafe.Pointer(cName))
		if res == C.DPI_FAILURE {
			return errors.Wrapf(st.getError(), "bindByName[%q]", name)
		}
	}
	return nil
}

func (st *statement) bindVarTypeSwitch(info *argInfo, get *dataGetter, value interface{}) (interface{}, error) {
	nilPtr := false
	if Log != nil {
		Log("msg", "bindVarTypeSwitch", "info", info, "value", fmt.Sprintf("[%T]%v", value, value))
	}
	vlr, isValuer := value.(driver.Valuer)

	switch value.(type) {
	case *driver.Rows:
	default:
		var magic bool
		rv := reflect.ValueOf(value)
		kind := rv.Kind()
		if kind != reflect.Ptr {
			if magic = st.MagicTypeConversion(); magic {
				if isValuer {
					var err error
					if value, err = vlr.Value(); err != nil {
						return value, errors.Wrap(err, "arg.Value()")
					}
					return st.bindVarTypeSwitch(info, get, value)
				}
			}
		} else {
			if nilPtr = rv.IsNil(); nilPtr {
				info.set = dataSetNull
				value = reflect.Zero(rv.Type().Elem()).Interface()
			} else {
				value = rv.Elem().Interface()
			}
		}
		if magic {
			switch kind {
			case reflect.Bool,
				reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
				reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64,
				reflect.Float32, reflect.Float64,
				reflect.String:
			default:
				magic = false
			}
		}
		if magic {
			switch rv.Type().Name() {
			case "bool",
				"int", "int8", "int16", "int32", "int64",
				"uint", "uint8", "uint16", "uint32", "uint64",
				"float32", "float64",
				"string":
				// nothing to do
				magic = false
			default:
			}
		}
		if magic {
			switch kind {
			case reflect.Bool:
				value = rv.Bool()
			case reflect.Int:
				value = int(rv.Int())
			case reflect.Int8:
				value = int8(rv.Int())
			case reflect.Int16:
				value = int16(rv.Int())
			case reflect.Int32:
				value = int32(rv.Int())
			case reflect.Int64:
				value = rv.Int()
			case reflect.Uint:
				value = uint(rv.Uint())
			case reflect.Uint16:
				value = uint16(rv.Uint())
			case reflect.Uint32:
				value = uint32(rv.Uint())
			case reflect.Uint64:
				value = rv.Uint()
			case reflect.Float32:
				value = float32(rv.Float())
			case reflect.Float64:
				value = rv.Float()
			case reflect.String:
				value = rv.String()
			}
		}
	}

	switch v := value.(type) {
	case Lob, []Lob:
		info.typ, info.natTyp = C.DPI_ORACLE_TYPE_BLOB, C.DPI_NATIVE_TYPE_LOB
		var isClob bool
		switch v := v.(type) {
		case Lob:
			isClob = v.IsClob
		case []Lob:
			isClob = len(v) > 0 && v[0].IsClob
		}
		if isClob {
			info.typ = C.DPI_ORACLE_TYPE_CLOB
		}
		info.set = st.dataSetLOB
		if info.isOut {
			*get = st.dataGetLOB
		}
	case *driver.Rows:
		info.typ, info.natTyp = C.DPI_ORACLE_TYPE_STMT, C.DPI_NATIVE_TYPE_STMT
		info.set = dataSetNull
		if info.isOut {
			*get = st.dataGetStmt
		}
	case int, []int:
		info.typ, info.natTyp = C.DPI_ORACLE_TYPE_NUMBER, C.DPI_NATIVE_TYPE_INT64
		if !nilPtr {
			info.set = dataSetNumber
			if info.isOut {
				*get = dataGetNumber
			}
		}
	case int32, []int32:
		info.typ, info.natTyp = C.DPI_ORACLE_TYPE_NATIVE_INT, C.DPI_NATIVE_TYPE_INT64
		if !nilPtr {
			info.set = dataSetNumber
			if info.isOut {
				*get = dataGetNumber
			}
		}
	case int64, []int64, sql.NullInt64, []sql.NullInt64:
		info.typ, info.natTyp = C.DPI_ORACLE_TYPE_NUMBER, C.DPI_NATIVE_TYPE_INT64
		if !nilPtr {
			info.set = dataSetNumber
			if info.isOut {
				*get = dataGetNumber
			}
		}
	case uint, []uint:
		info.typ, info.natTyp = C.DPI_ORACLE_TYPE_NUMBER, C.DPI_NATIVE_TYPE_UINT64
		if !nilPtr {
			info.set = dataSetNumber
			if info.isOut {
				*get = dataGetNumber
			}
		}
	case uint32, []uint32:
		info.typ, info.natTyp = C.DPI_ORACLE_TYPE_NATIVE_UINT, C.DPI_NATIVE_TYPE_UINT64
		if !nilPtr {
			info.set = dataSetNumber
			if info.isOut {
				*get = dataGetNumber
			}
		}
	case uint64, []uint64:
		info.typ, info.natTyp = C.DPI_ORACLE_TYPE_NUMBER, C.DPI_NATIVE_TYPE_UINT64
		if !nilPtr {
			info.set = dataSetNumber
			if info.isOut {
				*get = dataGetNumber
			}
		}
	case float32, []float32:
		info.typ, info.natTyp = C.DPI_ORACLE_TYPE_NATIVE_FLOAT, C.DPI_NATIVE_TYPE_FLOAT
		if !nilPtr {
			info.set = dataSetNumber
			if info.isOut {
				*get = dataGetNumber
			}
		}
	case float64, []float64:
		info.typ, info.natTyp = C.DPI_ORACLE_TYPE_NATIVE_DOUBLE, C.DPI_NATIVE_TYPE_DOUBLE
		if !nilPtr {
			info.set = dataSetNumber
			if info.isOut {
				*get = dataGetNumber
			}
		}
	case sql.NullFloat64, []sql.NullFloat64:
		info.typ, info.natTyp = C.DPI_ORACLE_TYPE_NATIVE_DOUBLE, C.DPI_NATIVE_TYPE_DOUBLE
		info.set = dataSetNumber
		if info.isOut {
			*get = dataGetNumber
		}
	case bool, []bool:
		info.typ, info.natTyp = C.DPI_ORACLE_TYPE_BOOLEAN, C.DPI_NATIVE_TYPE_BOOLEAN
		info.set = dataSetBool
		if info.isOut {
			*get = dataGetBool
		}

	case []byte, [][]byte:
		info.typ, info.natTyp = C.DPI_ORACLE_TYPE_RAW, C.DPI_NATIVE_TYPE_BYTES
		switch v := v.(type) {
		case []byte:
			info.bufSize = len(v)
		case [][]byte:
			for _, b := range v {
				if n := len(b); n > info.bufSize {
					info.bufSize = n
				}
			}
		}
		info.set = dataSetBytes
		if info.isOut {
			info.bufSize = 4000
			*get = dataGetBytes
		}

	case Number, []Number:
		info.typ, info.natTyp = C.DPI_ORACLE_TYPE_NUMBER, C.DPI_NATIVE_TYPE_BYTES
		switch v := v.(type) {
		case Number:
			info.bufSize = len(v)
		case []Number:
			for _, s := range v {
				if n := len(s); n > info.bufSize {
					info.bufSize = n
				}
			}
		}
		info.set = dataSetBytes
		if info.isOut {
			info.bufSize = 32767
			*get = dataGetBytes
		}

	case string, []string, nil:
		info.typ, info.natTyp = C.DPI_ORACLE_TYPE_VARCHAR, C.DPI_NATIVE_TYPE_BYTES
		switch v := v.(type) {
		case string:
			info.bufSize = 4 * len(v)
		case []string:
			for _, s := range v {
				if n := 4 * len(s); n > info.bufSize {
					info.bufSize = n
				}
			}
		}
		info.set = dataSetBytes
		if info.isOut {
			info.bufSize = 32767
			*get = dataGetBytes
		}

	case time.Time, []time.Time:
		info.typ, info.natTyp = C.DPI_ORACLE_TYPE_DATE, C.DPI_NATIVE_TYPE_TIMESTAMP
		info.set = st.conn.dataSetTime
		if info.isOut {
			*get = st.conn.dataGetTime
		}

	case Object:
		info.objType = v.ObjectType.dpiObjectType
		info.typ, info.natTyp = C.DPI_ORACLE_TYPE_OBJECT, C.DPI_NATIVE_TYPE_OBJECT
		info.set = st.dataSetObject
		if info.isOut {
			*get = st.dataGetObject
		}

	case *Object:
		info.objType = v.ObjectType.dpiObjectType
		info.typ, info.natTyp = C.DPI_ORACLE_TYPE_OBJECT, C.DPI_NATIVE_TYPE_OBJECT
		info.set = st.dataSetObject
		if info.isOut {
			*get = st.dataGetObject
		}

	case userType:
		info.objType = v.ObjectRef().ObjectType.dpiObjectType
		info.typ, info.natTyp = C.DPI_ORACLE_TYPE_OBJECT, C.DPI_NATIVE_TYPE_OBJECT
		info.set = st.dataSetObject
		if info.isOut {
			*get = st.dataGetObject
		}

	default:
		if !isValuer {
			return value, errors.Errorf("unknown type %T", value)
		}
		var err error
		if value, err = vlr.Value(); err != nil {
			return value, errors.Wrap(err, "arg.Value()")
		}
		return st.bindVarTypeSwitch(info, get, value)
	}

	return value, nil
}

type dataSetter func(dv *C.dpiVar, data []C.dpiData, vv interface{}) error

func dataSetNull(dv *C.dpiVar, data []C.dpiData, vv interface{}) error {
	for i := range data {
		data[i].isNull = 1
	}
	return nil
}
func dataGetBool(v interface{}, data []C.dpiData) error {
	if b, ok := v.(*bool); ok {
		if len(data) == 0 || data[0].isNull == 1 {
			*b = false
			return nil
		}
		*b = C.dpiData_getBool(&data[0]) == 1
		return nil
	}
	slice := v.(*[]bool)
	if cap(*slice) >= len(data) {
		*slice = (*slice)[:len(data)]
	} else {
		*slice = make([]bool, len(data))
	}
	for i := range data {
		if data[i].isNull == 1 {
			(*slice)[i] = false
			continue
		}
		(*slice)[i] = C.dpiData_getBool(&data[i]) == 1
	}
	return nil
}
func dataSetBool(dv *C.dpiVar, data []C.dpiData, vv interface{}) error {
	if vv == nil {
		return dataSetNull(dv, data, nil)
	}
	b := C.int(0)
	if bb, ok := vv.([]bool); ok {
		for i, v := range bb {
			if v {
				b = 1
			}
			C.dpiData_setBool(&data[i], b)
		}
	} else {
		for i := range data {
			data[i].isNull = 1
		}
	}
	return nil
}
func (c *conn) dataGetTime(v interface{}, data []C.dpiData) error {
	if x, ok := v.(*time.Time); ok {
		if len(data) == 0 || data[0].isNull == 1 {
			*x = time.Time{}
			return nil
		}
		c.dataGetTimeC(x, &data[0])
		return nil
	}
	slice := v.(*[]time.Time)
	n := len(data)
	if cap(*slice) >= n {
		*slice = (*slice)[:n]
	} else {
		*slice = make([]time.Time, n)
	}
	for i := range data {
		c.dataGetTimeC(&((*slice)[i]), &data[i])
	}
	return nil
}

func (c *conn) dataGetTimeC(t *time.Time, data *C.dpiData) {
	if data.isNull == 1 {
		*t = time.Time{}
		return
	}
	ts := C.dpiData_getTimestamp(data)
	tz := c.timeZone
	if ts.tzHourOffset != 0 || ts.tzMinuteOffset != 0 {
		tz = timeZoneFor(ts.tzHourOffset, ts.tzMinuteOffset)
	}
	*t = time.Date(
		int(ts.year), time.Month(ts.month), int(ts.day),
		int(ts.hour), int(ts.minute), int(ts.second), int(ts.fsecond),
		tz,
	)
}

func (c *conn) dataSetTime(dv *C.dpiVar, data []C.dpiData, vv interface{}) error {
	if vv == nil {
		return dataSetNull(dv, data, nil)
	}
	times := []time.Time{{}}
	if t, ok := vv.(time.Time); ok {
		times[0] = t
	} else {
		var ok bool
		if times, ok = vv.([]time.Time); !ok {
			for i := range data {
				data[i].isNull = 1
			}
			return nil
		}
	}
	tzHour, tzMin := C.int8_t(c.tzOffSecs/3600), C.int8_t((c.tzOffSecs%3600)/60)
	for i, t := range times {
		if t.IsZero() {
			data[i].isNull = 1
			continue
		}
		data[i].isNull = 0
		t = t.In(c.timeZone)
		Y, M, D := t.Date()
		h, m, s := t.Clock()
		C.dpiData_setTimestamp(&data[i],
			C.int16_t(Y), C.uint8_t(M), C.uint8_t(D),
			C.uint8_t(h), C.uint8_t(m), C.uint8_t(s), C.uint32_t(t.Nanosecond()),
			tzHour, tzMin,
		)
	}
	return nil
}

func dataGetNumber(v interface{}, data []C.dpiData) error {
	switch x := v.(type) {
	case *int:
		if len(data) == 0 || data[0].isNull == 1 {
			*x = 0
		} else {
			*x = int(C.dpiData_getInt64(&data[0]))
		}
	case *[]int:
		*x = (*x)[:0]
		for i := range data {
			if data[i].isNull == 1 {
				*x = append(*x, 0)
			} else {
				*x = append(*x, int(C.dpiData_getInt64(&data[i])))
			}
		}
	case *int32:
		if len(data) == 0 || data[0].isNull == 1 {
			*x = 0
		} else {
			*x = int32(C.dpiData_getInt64(&data[0]))
		}
	case *[]int32:
		*x = (*x)[:0]
		for i := range data {
			if data[i].isNull == 1 {
				*x = append(*x, 0)
			} else {
				*x = append(*x, int32(C.dpiData_getInt64(&data[i])))
			}
		}
	case *int64:
		if len(data) == 0 || data[0].isNull == 1 {
			*x = 0
		} else {
			*x = int64(C.dpiData_getInt64(&data[0]))
		}
	case *[]int64:
		*x = (*x)[:0]
		for i := range data {
			if data[i].isNull == 1 {
				*x = append(*x, 0)
			} else {
				*x = append(*x, int64(C.dpiData_getInt64(&data[i])))
			}
		}
	case *sql.NullInt64:
		if len(data) == 0 || data[0].isNull == 1 {
			x.Valid = false
		} else {
			x.Valid, x.Int64 = true, int64(C.dpiData_getInt64(&data[0]))
		}
	case *[]sql.NullInt64:
		*x = (*x)[:0]
		for i := range data {
			if data[i].isNull == 1 {
				*x = append(*x, sql.NullInt64{Valid: false})
			} else {
				*x = append(*x, sql.NullInt64{Valid: true,
					Int64: int64(C.dpiData_getInt64(&data[i]))})
			}
		}
	case *sql.NullFloat64:
		if len(data) == 0 || data[0].isNull == 1 {
			x.Valid = false
		} else {
			x.Valid, x.Float64 = true, float64(C.dpiData_getDouble(&data[0]))
		}
	case *[]sql.NullFloat64:
		*x = (*x)[:0]
		for i := range data {
			if data[i].isNull == 1 {
				*x = append(*x, sql.NullFloat64{Valid: false})
			} else {
				*x = append(*x, sql.NullFloat64{Valid: true, Float64: float64(C.dpiData_getDouble(&data[i]))})
			}
		}

	case *uint:
		if len(data) == 0 || data[0].isNull == 1 {
			*x = 0
		} else {
			*x = uint(C.dpiData_getUint64(&data[0]))
		}
	case *[]uint:
		*x = (*x)[:0]
		for i := range data {
			if data[i].isNull == 1 {
				*x = append(*x, 0)
			} else {
				*x = append(*x, uint(C.dpiData_getUint64(&data[i])))
			}
		}
	case *uint32:
		if len(data) == 0 || data[0].isNull == 1 {
			*x = 0
		} else {
			*x = uint32(C.dpiData_getUint64(&data[0]))
		}
	case *[]uint32:
		*x = (*x)[:0]
		for i := range data {
			if data[i].isNull == 1 {
				*x = append(*x, 0)
			} else {
				*x = append(*x, uint32(C.dpiData_getUint64(&data[i])))
			}
		}
	case *uint64:
		if len(data) == 0 || data[0].isNull == 1 {
			*x = 0
		} else {
			*x = uint64(C.dpiData_getUint64(&data[0]))
		}
	case *[]uint64:
		*x = (*x)[:0]
		for i := range data {
			if data[i].isNull == 1 {
				*x = append(*x, 0)
			} else {
				*x = append(*x, uint64(C.dpiData_getUint64(&data[i])))
			}
		}

	case *float32:
		if len(data) == 0 || data[0].isNull == 1 {
			*x = 0
		} else {
			*x = float32(C.dpiData_getFloat(&data[0]))
		}
	case *[]float32:
		*x = (*x)[:0]
		for i := range data {
			if data[i].isNull == 1 {
				*x = append(*x, 0)
			} else {
				*x = append(*x, float32(C.dpiData_getFloat(&data[i])))
			}
		}
	case *float64:
		if len(data) == 0 || data[0].isNull == 1 {
			*x = 0
		} else {
			*x = float64(C.dpiData_getDouble(&data[0]))
		}
	case *[]float64:
		*x = (*x)[:0]
		for i := range data {
			if data[i].isNull == 1 {
				*x = append(*x, 0)
			} else {
				*x = append(*x, float64(C.dpiData_getDouble(&data[i])))
			}
		}

	default:
		return errors.Errorf("unknown number [%T] %#v", v, v)
	}

	//fmt.Printf("setInt64(%#v, %#v)\n", data, C.int64_t(int64(v.(int))))
	return nil
}

func dataSetNumber(dv *C.dpiVar, data []C.dpiData, vv interface{}) error {
	if len(data) == 0 {
		return nil
	}
	if vv == nil {
		return dataSetNull(dv, data, nil)
	}
	switch slice := vv.(type) {
	case int:
		i, x := 0, slice
		C.dpiData_setInt64(&data[i], C.int64_t(x))
	case []int:
		for i, x := range slice {
			C.dpiData_setInt64(&data[i], C.int64_t(x))
		}
	case int32:
		i, x := 0, slice
		C.dpiData_setInt64(&data[i], C.int64_t(x))
	case []int32:
		for i, x := range slice {
			C.dpiData_setInt64(&data[i], C.int64_t(x))
		}
	case int64:
		i, x := 0, slice
		C.dpiData_setInt64(&data[i], C.int64_t(x))
	case []int64:
		for i, x := range slice {
			C.dpiData_setInt64(&data[i], C.int64_t(x))
		}
	case sql.NullInt64:
		i, x := 0, slice
		if x.Valid {
			data[i].isNull = 0
			C.dpiData_setInt64(&data[i], C.int64_t(x.Int64))
		} else {
			data[i].isNull = 1
		}
	case []sql.NullInt64:
		for i, x := range slice {
			if x.Valid {
				data[i].isNull = 0
				C.dpiData_setInt64(&data[i], C.int64_t(x.Int64))
			} else {
				data[i].isNull = 1
			}
		}
	case sql.NullFloat64:
		i, x := 0, slice
		if x.Valid {
			data[i].isNull = 0
			C.dpiData_setDouble(&data[i], C.double(x.Float64))
		} else {
			data[i].isNull = 1
		}
	case []sql.NullFloat64:
		for i, x := range slice {
			if x.Valid {
				data[i].isNull = 0
				C.dpiData_setDouble(&data[i], C.double(x.Float64))
			} else {
				data[i].isNull = 1
			}
		}

	case uint:
		i, x := 0, slice
		C.dpiData_setUint64(&data[i], C.uint64_t(x))
	case []uint:
		for i, x := range slice {
			C.dpiData_setUint64(&data[i], C.uint64_t(x))
		}
	case uint32:
		i, x := 0, slice
		C.dpiData_setUint64(&data[i], C.uint64_t(x))
	case []uint32:
		for i, x := range slice {
			C.dpiData_setUint64(&data[i], C.uint64_t(x))
		}
	case uint64:
		i, x := 0, slice
		C.dpiData_setUint64(&data[i], C.uint64_t(x))
	case []uint64:
		for i, x := range slice {
			C.dpiData_setUint64(&data[i], C.uint64_t(x))
		}

	case float32:
		i, x := 0, slice
		C.dpiData_setFloat(&data[i], C.float(x))
	case []float32:
		for i, x := range slice {
			C.dpiData_setFloat(&data[i], C.float(x))
		}
	case float64:
		i, x := 0, slice
		C.dpiData_setDouble(&data[i], C.double(x))
	case []float64:
		for i, x := range slice {
			C.dpiData_setDouble(&data[i], C.double(x))
		}

	default:
		return errors.Errorf("unknown number slice [%T] %#v", vv, vv)
	}

	//fmt.Printf("setInt64(%#v, %#v)\n", data, C.int64_t(int64(v.(int))))
	return nil
}

func dataGetBytes(v interface{}, data []C.dpiData) error {
	switch x := v.(type) {
	case *[]byte:
		if len(data) == 0 || data[0].isNull == 1 {
			*x = nil
			return nil
		}
		b := C.dpiData_getBytes(&data[0])

		*x = ((*[32767]byte)(unsafe.Pointer(b.ptr)))[:b.length:b.length]
	case *[][]byte:
		*x = (*x)[:0]
		for i := range data {
			if data[i].isNull == 1 {
				*x = append(*x, nil)
				continue
			}
			b := C.dpiData_getBytes(&data[i])
			*x = append(*x, ((*[32767]byte)(unsafe.Pointer(b.ptr)))[:b.length:b.length])
		}

	case *Number:
		if len(data) == 0 || data[0].isNull == 1 {
			*x = ""
			return nil
		}
		b := C.dpiData_getBytes(&data[0])
		*x = Number(((*[32767]byte)(unsafe.Pointer(b.ptr)))[:b.length:b.length])
	case *[]Number:
		*x = (*x)[:0]
		for i := range data {
			if data[i].isNull == 1 {
				*x = append(*x, "")
				continue
			}
			b := C.dpiData_getBytes(&data[i])
			*x = append(*x, Number(((*[32767]byte)(unsafe.Pointer(b.ptr)))[:b.length:b.length]))
		}

	case *string:
		if len(data) == 0 || data[0].isNull == 1 {
			*x = ""
			return nil
		}
		b := C.dpiData_getBytes(&data[0])
		*x = string(((*[32767]byte)(unsafe.Pointer(b.ptr)))[:b.length:b.length])
	case *[]string:
		*x = (*x)[:0]
		for i := range data {
			if data[i].isNull == 1 {
				*x = append(*x, "")
				continue
			}
			b := C.dpiData_getBytes(&data[i])
			*x = append(*x, string(((*[32767]byte)(unsafe.Pointer(b.ptr)))[:b.length:b.length]))
		}

	case *interface{}:
		switch y := (*x).(type) {
		case []byte:
			err := dataGetBytes(&y, data[:1])
			*x = y
			return err
		case [][]byte:
			err := dataGetBytes(&y, data)
			*x = y
			return err

		case Number:
			err := dataGetBytes(&y, data[:1])
			*x = y
			return err
		case []Number:
			err := dataGetBytes(&y, data)
			*x = y
			return err

		case string:
			err := dataGetBytes(&y, data[:1])
			*x = y
			return err
		case []string:
			err := dataGetBytes(&y, data)
			*x = y
			return err

		default:
			return errors.Errorf("awaited []byte/string/Number, got %T (%#v)", x, x)
		}

	default:
		return errors.Errorf("awaited []byte/string/Number, got %T (%#v)", v, v)
	}
	return nil
}

func dataSetBytes(dv *C.dpiVar, data []C.dpiData, vv interface{}) error {
	if len(data) == 0 {
		return nil
	}
	if vv == nil {
		return dataSetNull(dv, data, nil)
	}
	var p *C.char
	switch slice := vv.(type) {
	case []byte:
		i, x := 0, slice
		if len(x) == 0 {
			data[i].isNull = 1
			return nil
		}
		data[i].isNull = 0
		p = (*C.char)(unsafe.Pointer(&x[0]))
		//if Log != nil {Log("C", "dpiVar_setFromBytes", "dv", dv, "pos", pos, "p", p, "len", len(x)) }
		C.dpiVar_setFromBytes(dv, C.uint32_t(i), p, C.uint32_t(len(x)))
	case [][]byte:
		for i, x := range slice {
			if len(x) == 0 {
				data[i].isNull = 1
				continue
			}
			data[i].isNull = 0
			p = (*C.char)(unsafe.Pointer(&x[0]))
			//if Log != nil {Log("C", "dpiVar_setFromBytes", "dv", dv, "pos", pos, "p", p, "len", len(x)) }
			C.dpiVar_setFromBytes(dv, C.uint32_t(i), p, C.uint32_t(len(x)))
		}

	case Number:
		i, x := 0, slice
		if len(x) == 0 {
			data[i].isNull = 1
			return nil
		}
		data[i].isNull = 0
		dpiSetFromString(dv, C.uint32_t(i), string(x))
	case []Number:
		for i, x := range slice {
			if len(x) == 0 {
				data[i].isNull = 1
				continue
			}
			data[i].isNull = 0
			dpiSetFromString(dv, C.uint32_t(i), string(x))
		}

	case string:
		i, x := 0, slice
		if len(x) == 0 {
			data[i].isNull = 1
			return nil
		}
		data[i].isNull = 0
		dpiSetFromString(dv, C.uint32_t(i), x)
	case []string:
		for i, x := range slice {
			if len(x) == 0 {
				data[i].isNull = 1
				continue
			}
			data[i].isNull = 0
			dpiSetFromString(dv, C.uint32_t(i), x)
		}

	default:
		return errors.Errorf("awaited [][]byte/[]string/[]Number, got %T (%#v)", vv, vv)
	}
	return nil
}

func (c *conn) dataGetStmt(v interface{}, data []C.dpiData) error {
	if row, ok := v.(*driver.Rows); ok {
		if len(data) == 0 || data[0].isNull == 1 {
			*row = nil
			return nil
		}
		return c.dataGetStmtC(row, &data[0])
	}
	rows := v.(*[]driver.Rows)
	if cap(*rows) >= len(data) {
		*rows = (*rows)[:len(data)]
	} else {
		*rows = make([]driver.Rows, len(data))
	}
	var firstErr error
	for i := range data {
		if err := c.dataGetStmtC(&((*rows)[i]), &data[i]); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (c *conn) dataGetStmtC(row *driver.Rows, data *C.dpiData) error {
	if data.isNull == 1 {
		*row = nil
		return nil
	}
	st := &statement{conn: c, dpiStmt: C.dpiData_getStmt(data)}

	var n C.uint32_t
	if C.dpiStmt_getNumQueryColumns(st.dpiStmt, &n) == C.DPI_FAILURE {
		*row = &rows{
			err: errors.Wrapf(io.EOF, "getNumQueryColumns: %v", c.getError()),
		}
		return nil
	}
	var err error
	*row, err = st.openRows(int(n))
	return err
}

func (c *conn) dataGetLOB(v interface{}, data []C.dpiData) error {
	if L, ok := v.(*Lob); ok {
		if len(data) == 0 || data[0].isNull == 1 {
			*L = Lob{}
			return nil
		}
		c.dataGetLOBC(L, &data[0])
		return nil
	}
	slice := v.(*[]Lob)
	n := len(data)
	if cap(*slice) >= n {
		*slice = (*slice)[:n]
	} else {
		*slice = make([]Lob, n)
	}
	for i := range data {
		c.dataGetLOBC(&((*slice)[i]), &data[i])
	}
	return nil
}
func (c *conn) dataGetLOBC(L *Lob, data *C.dpiData) {
	L.Reader = nil
	if data.isNull == 1 {
		return
	}
	lob := C.dpiData_getLOB(data)
	if lob == nil {
		return
	}
	L.Reader = &dpiLobReader{conn: c, dpiLob: lob, IsClob: L.IsClob}
}

func (c *conn) dataSetLOB(dv *C.dpiVar, data []C.dpiData, vv interface{}) error {
	if len(data) == 0 {
		return nil
	}
	if vv == nil {
		return dataSetNull(dv, data, nil)
	}

	lobs := []Lob{{}}
	if L, ok := vv.(Lob); ok {
		lobs[0] = L
	} else {
		lobs = vv.([]Lob)
	}
	var firstErr error
	for i, L := range lobs {
		//fmt.Printf("dataSetLob[%d]=(%T) %#v\n", i, L, L)
		if L.Reader == nil {
			data[i].isNull = 1
			continue
		}
		data[i].isNull = 0
		if r, ok := L.Reader.(*dpiLobReader); ok {
			C.dpiVar_setFromLob(dv, C.uint32_t(i), r.dpiLob)
			continue
		}

		typ := C.dpiOracleTypeNum(C.DPI_ORACLE_TYPE_BLOB)
		if L.IsClob {
			typ = C.DPI_ORACLE_TYPE_CLOB
		}
		var lob *C.dpiLob
		if C.dpiConn_newTempLob(c.dpiConn, typ, &lob) == C.DPI_FAILURE {
			return errors.Wrapf(c.getError(), "newTempLob(typ=%d)", typ)
		}
		var chunkSize C.uint32_t
		_ = C.dpiLob_getChunkSize(lob, &chunkSize)
		if chunkSize == 0 {
			chunkSize = 8192
		}
		for chunkSize < minChunkSize {
			chunkSize <<= 1
		}
		lw := &dpiLobWriter{dpiLob: lob, conn: c, isClob: L.IsClob}
		_, err := io.CopyBuffer(lw, L, make([]byte, int(chunkSize)))
		//fmt.Printf("%p written %d with chunkSize=%d\n", lob, n, chunkSize)
		if closeErr := lw.Close(); closeErr != nil {
			if err == nil {
				err = closeErr
			}
			//fmt.Printf("close %p: %+v\n", lob, closeErr)
		}
		C.dpiVar_setFromLob(dv, C.uint32_t(i), lob)
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

type userType interface {
	ObjectRef() *Object
}

// ObjectScanner assigns a value from a database object
type ObjectScanner interface {
	sql.Scanner
	userType
}

// ObjectWriter update database object before binding
type ObjectWriter interface {
	WriteObject() error
	userType
}

func (c *conn) dataSetObject(dv *C.dpiVar, data []C.dpiData, vv interface{}) error {
	//fmt.Printf("\ndataSetObject(dv=%+v, data=%+v, vv=%+v)\n", dv, data, vv)
	if len(data) == 0 {
		return nil
	}
	if vv == nil {
		return dataSetNull(dv, data, nil)
	}
	objs := []Object{{}}
	switch o := vv.(type) {
	case Object:
		objs[0] = o
	case []Object:
		objs = o
	case ObjectWriter:
		err := o.WriteObject()
		if err != nil {
			return err
		}
		objs[0] = *o.ObjectRef()
	case []ObjectWriter:
		for _, ut := range o {
			err := ut.WriteObject()
			if err != nil {
				return err
			}
			objs = append(objs, *ut.ObjectRef())
		}
	case userType:
		objs[0] = *o.ObjectRef()
	case []userType:
		for _, ut := range o {
			objs = append(objs, *ut.ObjectRef())
		}
	}
	for i, obj := range objs {
		if obj.dpiObject == nil {
			data[i].isNull = 1
			return nil
		}
		data[i].isNull = 0
		C.dpiVar_setFromObject(dv, C.uint32_t(i), obj.dpiObject)
	}
	return nil
}

func (c *conn) dataGetObject(v interface{}, data []C.dpiData) error {
	switch out := v.(type) {
	case *Object:
		d := Data{
			ObjectType: out.ObjectType,
			dpiData:    &data[0],
		}
		*out = *d.GetObject()
	case ObjectScanner:
		d := Data{
			ObjectType: out.ObjectRef().ObjectType,
			dpiData:    &data[0],
		}
		return out.Scan(d.GetObject())
	default:

		return fmt.Errorf("dataGetObject not implemented for type %T (maybe you need to implement the Scan method)", v)
	}

	return nil
}

// CheckNamedValue is called before passing arguments to the driver
// and is called in place of any ColumnConverter. CheckNamedValue must do type
// validation and conversion as appropriate for the driver.
//
// If CheckNamedValue returns ErrRemoveArgument, the NamedValue will not be included
// in the final query arguments.
// This may be used to pass special options to the query itself.
//
// If ErrSkip is returned the column converter error checking path is used
// for the argument.
// Drivers may wish to return ErrSkip after they have exhausted their own special cases.
func (st *statement) CheckNamedValue(nv *driver.NamedValue) error {
	if nv == nil {
		return nil
	}
	if apply, ok := nv.Value.(Option); ok {
		if apply != nil {
			apply(&st.stmtOptions)
		}
		return driver.ErrRemoveArgument
	}
	return nil
}

// ColumnConverter may be optionally implemented by Stmt
// if the statement is aware of its own columns' types and
// can convert from any type to a driver Value.
func (st *statement) ColumnConverter(idx int) driver.ValueConverter {
	c := driver.ValueConverter(driver.DefaultParameterConverter)
	switch col := st.columns[idx]; col.OracleType {
	case C.DPI_ORACLE_TYPE_NUMBER:
		switch col.NativeType {
		case C.DPI_NATIVE_TYPE_INT64, C.DPI_NATIVE_TYPE_UINT64:
			c = Int64
		//case C.DPI_NATIVE_TYPE_FLOAT, C.DPI_NATIVE_TYPE_DOUBLE:
		//	c = Float64
		default:
			c = Num
		}
	}
	if Log != nil {
		Log("msg", "ColumnConverter", "c", c)
	}
	return driver.Null{Converter: c}
}

func (st *statement) openRows(colCount int) (*rows, error) {
	C.dpiStmt_setFetchArraySize(st.dpiStmt, C.uint32_t(st.FetchRowCount()))

	r := rows{
		statement: st,
		columns:   make([]Column, colCount),
		vars:      make([]*C.dpiVar, colCount),
		data:      make([][]C.dpiData, colCount),
	}
	sliceLen := st.FetchRowCount()

	var info C.dpiQueryInfo
	var ti C.dpiDataTypeInfo
	for i := 0; i < colCount; i++ {
		if C.dpiStmt_getQueryInfo(st.dpiStmt, C.uint32_t(i+1), &info) == C.DPI_FAILURE {
			return nil, errors.Wrapf(st.getError(), "getQueryInfo[%d]", i)
		}
		ti = info.typeInfo
		bufSize := int(ti.clientSizeInBytes)
		//if Log != nil {Log("msg", "openRows", "col", i, "info", ti) }
		//if Log != nil {Log("dNTN", int(ti.defaultNativeTypeNum), "number", C.DPI_ORACLE_TYPE_NUMBER) }
		switch ti.oracleTypeNum {
		case C.DPI_ORACLE_TYPE_NUMBER:
			switch ti.defaultNativeTypeNum {
			case C.DPI_NATIVE_TYPE_FLOAT, C.DPI_NATIVE_TYPE_DOUBLE:
				ti.defaultNativeTypeNum = C.DPI_NATIVE_TYPE_BYTES
				bufSize = 40
			}
		case C.DPI_ORACLE_TYPE_DATE:
			ti.defaultNativeTypeNum = C.DPI_NATIVE_TYPE_TIMESTAMP

		case C.DPI_ORACLE_TYPE_BLOB:
			if !st.LobAsReader() {
				ti.oracleTypeNum = C.DPI_ORACLE_TYPE_LONG_RAW
				ti.defaultNativeTypeNum = C.DPI_NATIVE_TYPE_BYTES
			}
		case C.DPI_ORACLE_TYPE_CLOB:
			if !st.LobAsReader() {
				ti.oracleTypeNum = C.DPI_ORACLE_TYPE_LONG_VARCHAR
				ti.defaultNativeTypeNum = C.DPI_NATIVE_TYPE_BYTES
			}
		}
		r.columns[i] = Column{
			Name:        C.GoStringN(info.name, C.int(info.nameLength)),
			OracleType:  ti.oracleTypeNum,
			NativeType:  ti.defaultNativeTypeNum,
			Size:        ti.clientSizeInBytes,
			Precision:   ti.precision,
			Scale:       ti.scale,
			Nullable:    info.nullOk == 1,
			ObjectType:  ti.objectType,
			SizeInChars: ti.sizeInChars,
			DBSize:      ti.dbSizeInBytes,
		}
		var err error
		//fmt.Printf("%d. %+v\n", i, r.columns[i])
		vi := varInfo{
			Typ:        ti.oracleTypeNum,
			NatTyp:     ti.defaultNativeTypeNum,
			ObjectType: ti.objectType,
			BufSize:    bufSize,
			SliceLen:   sliceLen,
		}
		if r.vars[i], r.data[i], err = st.newVar(vi); err != nil {
			return nil, err
		}

		if C.dpiStmt_define(st.dpiStmt, C.uint32_t(i+1), r.vars[i]) == C.DPI_FAILURE {
			return nil, errors.Wrapf(st.getError(), "define[%d]", i)
		}
	}
	if C.dpiStmt_addRef(st.dpiStmt) == C.DPI_FAILURE {
		return &r, errors.Wrap(st.getError(), "dpiStmt_addRef")
	}
	st.columns = r.columns
	return &r, nil
}

// Column holds the info from a column.
type Column struct {
	Name                      string
	ObjectType                *C.dpiObjectType
	OracleType                C.dpiOracleTypeNum
	NativeType                C.dpiNativeTypeNum
	Size, SizeInChars, DBSize C.uint32_t
	Precision                 C.int16_t
	Scale                     C.int8_t
	Nullable                  bool
}
