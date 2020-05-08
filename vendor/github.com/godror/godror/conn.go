// Copyright 2019 Tamás Gulácsi
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
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"

	errors "golang.org/x/xerrors"
)

const getConnection = "--GET_CONNECTION--"
const wrapResultset = "--WRAP_RESULTSET--"

// The maximum capacity is limited to (2^32 / sizeof(dpiData))-1 to remain compatible
// with 32-bit platforms. The size of a `C.dpiData` is 32 Byte on a 64-bit system, `C.dpiSubscrMessageTable` is 40 bytes.
// So this is 2^25.
// See https://github.com/go-godror/godror/issues/73#issuecomment-401281714
const maxArraySize = (1<<30)/C.sizeof_dpiSubscrMessageTable - 1

var _ = driver.Conn((*conn)(nil))
var _ = driver.ConnBeginTx((*conn)(nil))
var _ = driver.ConnPrepareContext((*conn)(nil))
var _ = driver.Pinger((*conn)(nil))

//var _ = driver.ExecerContext((*conn)(nil))

type conn struct {
	currentTT      TraceTag
	connParams     ConnectionParams
	Client, Server VersionInfo
	tranParams     tranParams
	mu             sync.RWMutex
	currentUser    string
	drv            *drv
	dpiConn        *C.dpiConn
	timeZone       *time.Location
	objTypes       map[string]ObjectType
	tzOffSecs      int
	inTransaction  bool
	newSession     bool
}

func (c *conn) getError() error {
	if c == nil || c.drv == nil {
		return driver.ErrBadConn
	}
	return c.drv.getError()
}

func (c *conn) Break() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if Log != nil {
		Log("msg", "Break", "dpiConn", c.dpiConn)
	}
	if C.dpiConn_breakExecution(c.dpiConn) == C.DPI_FAILURE {
		return maybeBadConn(errors.Errorf("Break: %w", c.getError()), c)
	}
	return nil
}

func (c *conn) ClientVersion() (VersionInfo, error) { return c.drv.ClientVersion() }

// Ping checks the connection's state.
//
// WARNING: as database/sql calls database/sql/driver.Open when it needs
// a new connection, but does not provide this Context,
// if the Open stalls (unreachable / firewalled host), the
// database/sql.Ping may return way after the Context.Deadline!
func (c *conn) Ping(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := c.ensureContextUser(ctx); err != nil {
		return err
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	done := make(chan error, 1)
	go func() {
		defer close(done)
		failure := C.dpiConn_ping(c.dpiConn) == C.DPI_FAILURE
		if failure {
			done <- maybeBadConn(errors.Errorf("Ping: %w", c.getError()), c)
			return
		}
		done <- nil
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		// select again to avoid race condition if both are done
		select {
		case err := <-done:
			return err
		default:
			_ = c.Break()
			c.close(true)
			return driver.ErrBadConn
		}
	}
}

// Prepare returns a prepared statement, bound to this connection.
func (c *conn) Prepare(query string) (driver.Stmt, error) {
	return c.PrepareContext(context.Background(), query)
}

// Close invalidates and potentially stops any current
// prepared statements and transactions, marking this
// connection as no longer in use.
//
// Because the sql package maintains a free pool of
// connections and only calls Close when there's a surplus of
// idle connections, it shouldn't be necessary for drivers to
// do their own connection caching.
func (c *conn) Close() error {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.close(true)
}

func (c *conn) close(doNotReuse bool) error {
	if c == nil {
		return nil
	}
	c.setTraceTag(TraceTag{})
	dpiConn, objTypes := c.dpiConn, c.objTypes
	c.dpiConn, c.objTypes = nil, nil
	if dpiConn == nil {
		return nil
	}
	defer C.dpiConn_release(dpiConn)

	seen := make(map[string]struct{}, len(objTypes))
	for _, o := range objTypes {
		nm := o.FullName()
		if _, seen := seen[nm]; seen {
			continue
		}
		seen[nm] = struct{}{}
		o.close(doNotReuse)
	}
	if !doNotReuse {
		return nil
	}

	// Just to be sure, break anything in progress.
	done := make(chan struct{})
	go func() {
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			if Log != nil {
				Log("msg", "TIMEOUT releasing connection")
			}
			C.dpiConn_breakExecution(dpiConn)
		}
	}()
	C.dpiConn_close(dpiConn, C.DPI_MODE_CONN_CLOSE_DROP, nil, 0)
	close(done)
	return nil
}

// Begin starts and returns a new transaction.
//
// Deprecated: Drivers should implement ConnBeginTx instead (or additionally).
func (c *conn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}

// BeginTx starts and returns a new transaction.
// If the context is canceled by the user the sql package will
// call Tx.Rollback before discarding and closing the connection.
//
// This must check opts.Isolation to determine if there is a set
// isolation level. If the driver does not support a non-default
// level and one is set or if there is a non-default isolation level
// that is not supported, an error must be returned.
//
// This must also check opts.ReadOnly to determine if the read-only
// value is true to either set the read-only transaction property if supported
// or return an error if it is not supported.
func (c *conn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	const (
		trRO = "READ ONLY"
		trRW = "READ WRITE"
		trLC = "ISOLATION LEVEL READ COMMIT" + "TED" // against misspell check
		trLS = "ISOLATION LEVEL SERIALIZABLE"
	)

	var todo tranParams
	if opts.ReadOnly {
		todo.RW = trRO
	} else {
		todo.RW = trRW
	}
	switch level := sql.IsolationLevel(opts.Isolation); level {
	case sql.LevelDefault:
	case sql.LevelReadCommitted:
		todo.Level = trLC
	case sql.LevelSerializable:
		todo.Level = trLS
	default:
		return nil, errors.Errorf("isolation level is not supported: %s", sql.IsolationLevel(opts.Isolation))
	}

	if todo != c.tranParams {
		for _, qry := range []string{todo.RW, todo.Level} {
			if qry == "" {
				continue
			}
			qry = "SET TRANSACTION " + qry
			stmt, err := c.PrepareContext(ctx, qry)
			if err == nil {
				if stc, ok := stmt.(driver.StmtExecContext); ok {
					_, err = stc.ExecContext(ctx, nil)
				} else {
					_, err = stmt.Exec(nil) //lint:ignore SA1019 as that comment is not relevant here
				}
				stmt.Close()
			}
			if err != nil {
				return nil, maybeBadConn(errors.Errorf("%s: %w", qry, err), c)
			}
		}
		c.tranParams = todo
	}

	c.mu.RLock()
	inTran := c.inTransaction
	c.mu.RUnlock()
	if inTran {
		return nil, errors.New("already in transaction")
	}
	c.mu.Lock()
	c.inTransaction = true
	c.mu.Unlock()
	if tt, ok := ctx.Value(traceTagCtxKey).(TraceTag); ok {
		c.mu.Lock()
		c.setTraceTag(tt)
		c.mu.Unlock()
	}
	return c, nil
}

type tranParams struct {
	RW, Level string
}

// PrepareContext returns a prepared statement, bound to this connection.
// context is for the preparation of the statement,
// it must not store the context within the statement itself.
func (c *conn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := c.ensureContextUser(ctx); err != nil {
		return nil, err
	}
	if tt, ok := ctx.Value(traceTagCtxKey).(TraceTag); ok {
		c.mu.Lock()
		c.setTraceTag(tt)
		c.mu.Unlock()
	}
	if query == getConnection {
		if Log != nil {
			Log("msg", "PrepareContext", "shortcut", query)
		}
		return &statement{conn: c, query: query}, nil
	}

	cSQL := C.CString(query)
	defer func() {
		C.free(unsafe.Pointer(cSQL))
	}()
	c.mu.RLock()
	defer c.mu.RUnlock()
	var dpiStmt *C.dpiStmt
	if C.dpiConn_prepareStmt(c.dpiConn, 0, cSQL, C.uint32_t(len(query)), nil, 0,
		(**C.dpiStmt)(unsafe.Pointer(&dpiStmt)),
	) == C.DPI_FAILURE {
		return nil, maybeBadConn(errors.Errorf("Prepare: %s: %w", query, c.getError()), c)
	}
	return &statement{conn: c, dpiStmt: dpiStmt, query: query}, nil
}
func (c *conn) Commit() error {
	return c.endTran(true)
}
func (c *conn) Rollback() error {
	return c.endTran(false)
}
func (c *conn) endTran(isCommit bool) error {
	c.mu.Lock()
	c.inTransaction = false
	c.tranParams = tranParams{}

	var err error
	//msg := "Commit"
	if isCommit {
		if C.dpiConn_commit(c.dpiConn) == C.DPI_FAILURE {
			err = maybeBadConn(errors.Errorf("Commit: %w", c.getError()), c)
		}
	} else {
		//msg = "Rollback"
		if C.dpiConn_rollback(c.dpiConn) == C.DPI_FAILURE {
			err = maybeBadConn(errors.Errorf("Rollback: %w", c.getError()), c)
		}
	}
	c.mu.Unlock()
	//fmt.Printf("%p.%s\n", c, msg)
	return err
}

type varInfo struct {
	SliceLen, BufSize int
	ObjectType        *C.dpiObjectType
	NatTyp            C.dpiNativeTypeNum
	Typ               C.dpiOracleTypeNum
	IsPLSArray        bool
}

func (c *conn) newVar(vi varInfo) (*C.dpiVar, []C.dpiData, error) {
	if c == nil || c.dpiConn == nil {
		return nil, nil, errors.New("connection is nil")
	}
	isArray := C.int(0)
	if vi.IsPLSArray {
		isArray = 1
	}
	if vi.SliceLen < 1 {
		vi.SliceLen = 1
	}
	var dataArr *C.dpiData
	var v *C.dpiVar
	if Log != nil {
		Log("C", "dpiConn_newVar", "conn", c.dpiConn, "typ", int(vi.Typ), "natTyp", int(vi.NatTyp), "sliceLen", vi.SliceLen, "bufSize", vi.BufSize, "isArray", isArray, "objType", vi.ObjectType, "v", v)
	}
	if C.dpiConn_newVar(
		c.dpiConn, vi.Typ, vi.NatTyp, C.uint32_t(vi.SliceLen),
		C.uint32_t(vi.BufSize), 1,
		isArray, vi.ObjectType,
		&v, &dataArr,
	) == C.DPI_FAILURE {
		return nil, nil, errors.Errorf("newVar(typ=%d, natTyp=%d, sliceLen=%d, bufSize=%d): %w", vi.Typ, vi.NatTyp, vi.SliceLen, vi.BufSize, c.getError())
	}
	// https://github.com/golang/go/wiki/cgo#Turning_C_arrays_into_Go_slices
	/*
		var theCArray *C.YourType = C.getTheArray()
		length := C.getTheArrayLength()
		slice := (*[maxArraySize]C.YourType)(unsafe.Pointer(theCArray))[:length:length]
	*/
	data := ((*[maxArraySize]C.dpiData)(unsafe.Pointer(dataArr)))[:vi.SliceLen:vi.SliceLen]
	return v, data, nil
}

var _ = driver.Tx((*conn)(nil))

func (c *conn) ServerVersion() (VersionInfo, error) {
	return c.Server, nil
}

func (c *conn) init(onInit []string) error {
	if c.Client.Version == 0 {
		var err error
		if c.Client, err = c.drv.ClientVersion(); err != nil {
			return err
		}
	}

	if err := c.initVersionTZ(); err != nil || len(onInit) == 0 || !c.newSession {
		return err
	}
	if Log != nil {
		Log("newSession", c.newSession, "onInit", onInit)
	}
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Duration(len(onInit))*time.Second)
			defer cancel()
			if Log != nil {
				Log("doOnInit", len(onInit))
			}
			for _, qry := range onInit {
				if Log != nil {
					Log("onInit", qry)
				}
				st, err := c.PrepareContext(ctx, qry)
				if err != nil {
					return errors.Errorf("%s: %w", qry, err)
				}
				_, err = st.Exec(nil) //lint:ignore SA1019 - it's hard to use ExecContext here
				st.Close()
				if err != nil {
					return errors.Errorf("%s: %w", qry, err)
				}
			}
			return nil
	}

func (c *conn) initVersionTZ() error {
	if c.Server.Version == 0 {
		var v C.dpiVersionInfo
		var release *C.char
		var releaseLen C.uint32_t
		if C.dpiConn_getServerVersion(c.dpiConn, &release, &releaseLen, &v) == C.DPI_FAILURE {
			if c.connParams.IsPrelim {
				return nil
			}
			return errors.Errorf("getServerVersion: %w", c.getError())
		}
		c.Server.set(&v)
		c.Server.ServerRelease = string(bytes.Replace(
			((*[maxArraySize]byte)(unsafe.Pointer(release)))[:releaseLen:releaseLen],
			[]byte{'\n'}, []byte{';', ' '}, -1))
	}

	if c.timeZone != nil && (c.timeZone != time.Local || c.tzOffSecs != 0) {
		return nil
	}
	c.timeZone = time.Local
	_, c.tzOffSecs = time.Now().In(c.timeZone).Zone()
	if Log != nil {
		Log("tz", c.timeZone, "offSecs", c.tzOffSecs)
	}

	// DBTIMEZONE is useless, false, and misdirecting!
	// https://stackoverflow.com/questions/52531137/sysdate-and-dbtimezone-different-in-oracle-database
	const qry = "SELECT DBTIMEZONE, LTRIM(REGEXP_SUBSTR(TO_CHAR(SYSTIMESTAMP), ' [^ ]+$')) FROM DUAL"
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	st, err := c.PrepareContext(ctx, qry)
	if err != nil {
		return errors.Errorf("%s: %w", qry, err)
	}
	defer st.Close()
	rows, err := st.Query(nil) //lint:ignore SA1019 - it's hard to use QueryContext here
	if err != nil {
		if Log != nil {
			Log("qry", qry, "error", err)
		}
		return nil
	}
	defer rows.Close()
	var dbTZ, timezone string
	vals := []driver.Value{dbTZ, timezone}
	if err = rows.Next(vals); err != nil && err != io.EOF {
		return errors.Errorf("%s: %w", qry, err)
	}
	dbTZ = vals[0].(string)
	timezone = vals[1].(string)

	tz, off, err := calculateTZ(dbTZ, timezone)
	if Log != nil {
		Log("timezone", timezone, "tz", tz, "offSecs", off)
	}
	if err != nil || tz == nil {
		return err
	}
	c.timeZone, c.tzOffSecs = tz, off

	return nil
}

func calculateTZ(dbTZ, timezone string) (*time.Location, int, error) {
	if Log != nil {
		Log("dbTZ", dbTZ, "timezone", timezone)
	}
	var tz *time.Location
	now := time.Now()
	_, localOff := time.Now().Local().Zone()
	off := localOff
	var ok bool
	var err error
	if dbTZ != "" && strings.Contains(dbTZ, "/") {
		tz, err = time.LoadLocation(dbTZ)
		if ok = err == nil; ok {
			if tz == time.Local {
				return tz, off, nil
			}
			_, off = now.In(tz).Zone()
		} else if Log != nil {
			Log("LoadLocation", dbTZ, "error", err)
		}
	}
	if !ok {
		if timezone != "" {
			if off, err = parseTZ(timezone); err != nil {
				return tz, off, errors.Errorf("%s: %w", timezone, err)
			}
		} else if off, err = parseTZ(dbTZ); err != nil {
			return tz, off, errors.Errorf("%s: %w", dbTZ, err)
		}
	}
	// This is dangerous, but I just cannot get whether the DB time zone
	// setting has DST or not - DBTIMEZONE returns just a fixed offset.
	if off != localOff && tz == nil {
		tz = time.FixedZone(timezone, off)
	}
	return tz, off, nil
}
func parseTZ(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, io.EOF
	}
	if s == "Z" || s == "UTC" {
		return 0, nil
	}
	var tz int
	var ok bool
	if i := strings.IndexByte(s, ':'); i >= 0 {
		if i64, err := strconv.ParseInt(s[i+1:], 10, 6); err != nil {
			return tz, errors.Errorf("%s: %w", s, err)
		} else {
			tz = int(i64 * 60)
		}
		s = s[:i]
		ok = true
	}
	if !ok {
		if i := strings.IndexByte(s, '/'); i >= 0 {
			targetLoc, err := time.LoadLocation(s)
			if err != nil {
				return tz, errors.Errorf("%s: %w", s, err)
			}

			_, localOffset := time.Now().In(targetLoc).Zone()

			tz = localOffset
			return tz, nil
		}
	}
	if i64, err := strconv.ParseInt(s, 10, 5); err != nil {
		return tz, errors.Errorf("%s: %w", s, err)
	} else {
		if i64 < 0 {
			tz = -tz
		}
		tz += int(i64 * 3600)
	}
	return tz, nil
}

func (c *conn) setCallTimeout(ctx context.Context) {
	if c.Client.Version < 18 {
		return
	}
	var ms C.uint32_t
	if dl, ok := ctx.Deadline(); ok {
		ms = C.uint32_t(time.Until(dl) / time.Millisecond)
	}
	// force it to be 0 (disabled)
	C.dpiConn_setCallTimeout(c.dpiConn, ms)
}

// maybeBadConn checks whether the error is because of a bad connection, and returns driver.ErrBadConn,
// as database/sql requires.
//
// Also in this case, iff c != nil, closes it.
func maybeBadConn(err error, c *conn) error {
	if err == nil {
		return nil
	}
	cl := func() {}
	if c != nil {
		cl = func() {
			if Log != nil {
				Log("msg", "maybeBadConn close", "conn", c)
			}
			c.close(true)
		}
	}
	if errors.Is(err, driver.ErrBadConn) {
		cl()
		return driver.ErrBadConn
	}
	var cd interface{ Code() int }
	if errors.As(err, &cd) {
		// Yes, this is copied from rana/ora, but I've put it there, so it's mine. @tgulacsi
		switch cd.Code() {
		case 0:
			if strings.Contains(err.Error(), " DPI-1002: ") {
				cl()
				return driver.ErrBadConn
			}
			// cases by experience:
			// ORA-12170: TNS:Connect timeout occurred
			// ORA-12528: TNS:listener: all appropriate instances are blocking new connections
			// ORA-12545: Connect failed because target host or object does not exist
			// ORA-24315: illegal attribute type
			// ORA-28547: connection to server failed, probable Oracle Net admin error
		case 12170, 12528, 12545, 24315, 28547:

			//cases from https://github.com/oracle/odpi/blob/master/src/dpiError.c#L61-L94
		case 22, // invalid session ID; access denied
			28,    // your session has been killed
			31,    // your session has been marked for kill
			45,    // your session has been terminated with no replay
			378,   // buffer pools cannot be created as specified
			602,   // internal programming exception
			603,   // ORACLE server session terminated by fatal error
			609,   // could not attach to incoming connection
			1012,  // not logged on
			1041,  // internal error. hostdef extension doesn't exist
			1043,  // user side memory corruption
			1089,  // immediate shutdown or close in progress
			1092,  // ORACLE instance terminated. Disconnection forced
			2396,  // exceeded maximum idle time, please connect again
			3113,  // end-of-file on communication channel
			3114,  // not connected to ORACLE
			3122,  // attempt to close ORACLE-side window on user side
			3135,  // connection lost contact
			3136,  // inbound connection timed out
			12153, // TNS:not connected
			12537, // TNS:connection closed
			12547, // TNS:lost contact
			12570, // TNS:packet reader failure
			12583, // TNS:no reader
			27146, // post/wait initialization failed
			28511, // lost RPC connection
			56600: // an illegal OCI function call was issued
			cl()
			return driver.ErrBadConn
		}
	}
	return err
}

func (c *conn) setTraceTag(tt TraceTag) error {
	if c == nil || c.dpiConn == nil {
		return nil
	}
	for nm, vv := range map[string][2]string{
		"action":     {c.currentTT.Action, tt.Action},
		"module":     {c.currentTT.Module, tt.Module},
		"info":       {c.currentTT.ClientInfo, tt.ClientInfo},
		"identifier": {c.currentTT.ClientIdentifier, tt.ClientIdentifier},
		"op":         {c.currentTT.DbOp, tt.DbOp},
	} {
		if vv[0] == vv[1] {
			continue
		}
		v := vv[1]
		var s *C.char
		if v != "" {
			s = C.CString(v)
		}
		var rc C.int
		switch nm {
		case "action":
			rc = C.dpiConn_setAction(c.dpiConn, s, C.uint32_t(len(v)))
		case "module":
			rc = C.dpiConn_setModule(c.dpiConn, s, C.uint32_t(len(v)))
		case "info":
			rc = C.dpiConn_setClientInfo(c.dpiConn, s, C.uint32_t(len(v)))
		case "identifier":
			rc = C.dpiConn_setClientIdentifier(c.dpiConn, s, C.uint32_t(len(v)))
		case "op":
			rc = C.dpiConn_setDbOp(c.dpiConn, s, C.uint32_t(len(v)))
		}
		if s != nil {
			C.free(unsafe.Pointer(s))
		}
		if rc == C.DPI_FAILURE {
			return errors.Errorf("%s: %w", nm, c.getError())
		}
	}
	c.currentTT = tt
	return nil
}

const traceTagCtxKey = ctxKey("tracetag")

// ContextWithTraceTag returns a context with the specified TraceTag, which will
// be set on the session used.
func ContextWithTraceTag(ctx context.Context, tt TraceTag) context.Context {
	return context.WithValue(ctx, traceTagCtxKey, tt)
}

// TraceTag holds tracing information for the session. It can be set on the session
// with ContextWithTraceTag.
type TraceTag struct {
	// ClientIdentifier - specifies an end user based on the logon ID, such as HR.HR
	ClientIdentifier string
	// ClientInfo - client-specific info
	ClientInfo string
	// DbOp - database operation
	DbOp string
	// Module - specifies a functional block, such as Accounts Receivable or General Ledger, of an application
	Module string
	// Action - specifies an action, such as an INSERT or UPDATE operation, in a module
	Action string
}

const userpwCtxKey = ctxKey("userPw")

// ContextWithUserPassw returns a context with the specified user and password,
// to be used with heterogeneous pools.
func ContextWithUserPassw(ctx context.Context, user, password, connClass string) context.Context {
	return context.WithValue(ctx, userpwCtxKey, [3]string{user, password, connClass})
}

func (c *conn) ensureContextUser(ctx context.Context) error {
	if !(c.connParams.HeterogeneousPool || c.connParams.StandaloneConnection) {
		return nil
	}
	up, ok := ctx.Value(userpwCtxKey).([3]string)
	if !ok || up[0] == c.currentUser {
		return nil
	}

	if c.dpiConn != nil {
		if err := c.close(false); err != nil {
			return driver.ErrBadConn
		}
	}

	return c.acquireConn(up[0], up[1], up[2])
}

// StartupMode for the database.
type StartupMode C.dpiStartupMode

const (
	// StartupDefault is the default mode for startup which permits database access to all users.
	StartupDefault = StartupMode(C.DPI_MODE_STARTUP_DEFAULT)
	// StartupForce shuts down a running instance (using ABORT) before starting a new one. This mode should only be used in unusual circumstances.
	StartupForce = StartupMode(C.DPI_MODE_STARTUP_FORCE)
	// StartupRestrict only allows database access to users with both the CREATE SESSION and RESTRICTED SESSION privileges (normally the DBA).
	StartupRestrict = StartupMode(C.DPI_MODE_STARTUP_RESTRICT)
)

// Startup the database, equivalent to "startup nomount" in SQL*Plus.
// This should be called on PRELIM_AUTH (prelim=1) connection!
//
// See https://docs.oracle.com/en/database/oracle/oracle-database/18/lnoci/database-startup-and-shutdown.html#GUID-44B24F65-8C24-4DF3-8FBF-B896A4D6F3F3
func (c *conn) Startup(mode StartupMode) error {
	if C.dpiConn_startupDatabase(c.dpiConn, C.dpiStartupMode(mode)) == C.DPI_FAILURE {
		return errors.Errorf("startup(%v): %w", mode, c.getError())
	}
	return nil
}

// ShutdownMode for the database.
type ShutdownMode C.dpiShutdownMode

const (
	// ShutdownDefault - further connections to the database are prohibited. Wait for users to disconnect from the database.
	ShutdownDefault = ShutdownMode(C.DPI_MODE_SHUTDOWN_DEFAULT)
	// ShutdownTransactional - further connections to the database are prohibited and no new transactions are allowed to be started. Wait for active transactions to complete.
	ShutdownTransactional = ShutdownMode(C.DPI_MODE_SHUTDOWN_TRANSACTIONAL)
	// ShutdownTransactionalLocal - behaves the same way as ShutdownTransactional but only waits for local transactions to complete.
	ShutdownTransactionalLocal = ShutdownMode(C.DPI_MODE_SHUTDOWN_TRANSACTIONAL_LOCAL)
	// ShutdownImmediate - all uncommitted transactions are terminated and rolled back and all connections to the database are closed immediately.
	ShutdownImmediate = ShutdownMode(C.DPI_MODE_SHUTDOWN_IMMEDIATE)
	// ShutdownAbort - all uncommitted transactions are terminated and are not rolled back. This is the fastest way to shut down the database but the next database startup may require instance recovery.
	ShutdownAbort = ShutdownMode(C.DPI_MODE_SHUTDOWN_ABORT)
	// ShutdownFinal shuts down the database. This mode should only be used in the second call to dpiConn_shutdownDatabase().
	ShutdownFinal = ShutdownMode(C.DPI_MODE_SHUTDOWN_FINAL)
)

// Shutdown shuts down the database.
// Note that this must be done in two phases except in the situation where the instance is aborted.
//
// See https://docs.oracle.com/en/database/oracle/oracle-database/18/lnoci/database-startup-and-shutdown.html#GUID-44B24F65-8C24-4DF3-8FBF-B896A4D6F3F3
func (c *conn) Shutdown(mode ShutdownMode) error {
	if C.dpiConn_shutdownDatabase(c.dpiConn, C.dpiShutdownMode(mode)) == C.DPI_FAILURE {
		return errors.Errorf("shutdown(%v): %w", mode, c.getError())
	}
	return nil
}

// Timezone returns the connection's timezone.
func (c *conn) Timezone() *time.Location {
	return c.timeZone
}
