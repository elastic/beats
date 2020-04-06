// Copyright 2019 Tamás Gulácsi
//
//
// SPDX-License-Identifier: UPL-1.0 OR Apache-2.0

// Package godror is a database/sql/driver for Oracle DB.
//
// The connection string for the sql.Open("godror", connString) call can be
// the simple
//   login/password@sid [AS SYSDBA|AS SYSOPER]
//
// type (with sid being the sexp returned by tnsping),
// or in the form of
//   ora://login:password@sid/? \
//     sysdba=0& \
//     sysoper=0& \
//     poolMinSessions=1& \
//     poolMaxSessions=1000& \
//     poolIncrement=1& \
//     connectionClass=POOLED& \
//     standaloneConnection=0& \
//     enableEvents=0& \
//     heterogeneousPool=0& \
//     prelim=0& \
//     poolWaitTimeout=5m& \
//     poolSessionMaxLifetime=1h& \
//     poolSessionTimeout=30s& \
//     timezone=Local& \
//     newPassword= \
//     onInit=ALTER+SESSION+SET+current_schema%3Dmy_schema
//
// These are the defaults. Many advocate that a static session pool (min=max, incr=0)
// is better, with 1-10 sessions per CPU thread.
// See http://docs.oracle.com/cd/E82638_01/JJUCP/optimizing-real-world-performance.htm#JJUCP-GUID-BC09F045-5D80-4AF5-93F5-FEF0531E0E1D
// You may also use ConnectionParams to configure a connection.
//
// If you specify connectionClass, that'll reuse the same session pool
// without the connectionClass, but will specify it on each session acquire.
// Thus you can cluster the session pool with classes, or use POOLED for DRCP.
//
// For what can be used as "sid", see https://docs.oracle.com/en/database/oracle/oracle-database/19/netag/configuring-naming-methods.html#GUID-E5358DEA-D619-4B7B-A799-3D2F802500F1
package godror

/*
#cgo CFLAGS: -I./odpi/include -I./odpi/src -I./odpi/embed

#include <stdlib.h>

#include "dpi.c"
*/
import "C"

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/base64"
	"fmt"
	"hash/fnv"
	"io"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"

	errors "golang.org/x/xerrors"
)

const (
	// DefaultFetchRowCount is the number of prefetched rows by default (if not changed through FetchRowCount statement option).
	DefaultFetchRowCount = 1 << 8

	// DefaultArraySize is the length of the maximum PL/SQL array by default (if not changed through ArraySize statement option).
	DefaultArraySize = 1 << 10
)

const (
	// DpiMajorVersion is the wanted major version of the underlying ODPI-C library.
	DpiMajorVersion = C.DPI_MAJOR_VERSION
	// DpiMinorVersion is the wanted minor version of the underlying ODPI-C library.
	DpiMinorVersion = C.DPI_MINOR_VERSION
	// DpiPatchLevel is the patch level version of the underlying ODPI-C library
	DpiPatchLevel = C.DPI_PATCH_LEVEL
	// DpiVersionNumber is the underlying ODPI-C version as one number (Major * 10000 + Minor * 100 + Patch)
	DpiVersionNumber = C.DPI_VERSION_NUMBER

	// DriverName is set on the connection to be seen in the DB
	//
	// It cannot be longer than 30 bytes !
	DriverName = "godror : " + Version

	// DefaultPoolMinSessions specifies the default value for minSessions for pool creation.
	DefaultPoolMinSessions = 1
	// DefaultPoolMaxSessions specifies the default value for maxSessions for pool creation.
	DefaultPoolMaxSessions = 1000
	// DefaultPoolIncrement specifies the default value for increment for pool creation.
	DefaultPoolIncrement = 1
	// DefaultConnectionClass is the default connectionClass
	DefaultConnectionClass = "GODROR"
	// NoConnectionPoolingConnectionClass is a special connection class name to indicate no connection pooling.
	// It is the same as setting standaloneConnection=1
	NoConnectionPoolingConnectionClass = "NO-CONNECTION-POOLING"
	// DefaultSessionTimeout is the seconds before idle pool sessions get evicted
	DefaultSessionTimeout = 5 * time.Minute
	// DefaultWaitTimeout is the milliseconds to wait for a session to become available
	DefaultWaitTimeout = 30 * time.Second
	// DefaultMaxLifeTime is the maximum time in seconds till a pooled session may exist
	DefaultMaxLifeTime = 1 * time.Hour
)

// Log function. By default, it's nil, and thus logs nothing.
// If you want to change this, change it to a github.com/go-kit/kit/log.Swapper.Log
// or analog to be race-free.
var Log func(...interface{}) error

var defaultDrv = &drv{}

func init() {
	sql.Register("godror", defaultDrv)
}

var _ = driver.Driver((*drv)(nil))

type drv struct {
	mu            sync.Mutex
	dpiContext    *C.dpiContext
	pools         map[string]*connPool
	clientVersion VersionInfo
}

type connPool struct {
	dpiPool       *C.dpiPool
	timeZone      *time.Location
	tzOffSecs     int
	serverVersion VersionInfo
}

func (d *drv) init() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.pools == nil {
		d.pools = make(map[string]*connPool)
	}
	if d.dpiContext != nil {
		return nil
	}
	var errInfo C.dpiErrorInfo
	var dpiCtx *C.dpiContext
	if C.dpiContext_create(C.uint(DpiMajorVersion), C.uint(DpiMinorVersion),
		(**C.dpiContext)(unsafe.Pointer(&dpiCtx)), &errInfo,
	) == C.DPI_FAILURE {
		return fromErrorInfo(errInfo)
	}
	d.dpiContext = dpiCtx

	var v C.dpiVersionInfo
	if C.dpiContext_getClientVersion(d.dpiContext, &v) == C.DPI_FAILURE {
		return errors.Errorf("%s: %w", "getClientVersion", d.getError())
	}
	d.clientVersion.set(&v)
	return nil
}

// Open returns a new connection to the database.
// The name is a string in a driver-specific format.
func (d *drv) Open(connString string) (driver.Conn, error) {
	P, err := ParseConnString(connString)
	if err != nil {
		return nil, err
	}

	conn, err := d.openConn(P)
	return conn, maybeBadConn(err, conn)
}

func (d *drv) ClientVersion() (VersionInfo, error) {
	return d.clientVersion, nil
}

var cUTF8, cDriverName = C.CString("AL32UTF8"), C.CString(DriverName)

func (d *drv) openConn(P ConnectionParams) (*conn, error) {
	if err := d.init(); err != nil {
		return nil, err
	}

	P.Comb()
	c := &conn{drv: d, connParams: P, timeZone: time.Local, Client: d.clientVersion}
	connString := P.String()

	if Log != nil {
		defer func() {
			d.mu.Lock()
			Log("pools", d.pools, "conn", P.String(), "drv", fmt.Sprintf("%p", d))
			d.mu.Unlock()
		}()
	}

	if !(P.IsSysDBA || P.IsSysOper || P.IsSysASM || P.IsPrelim || P.StandaloneConnection) {
		d.mu.Lock()
		dp := d.pools[connString]
		d.mu.Unlock()
		if dp != nil {
			//Proxy authenticated connections to database will be provided by methods with context
			err := dp.acquireConn(c, P)
			return c, err
		}
	}

	extAuth := C.int(b2i(P.Username == "" && P.Password == ""))
	var cUserName, cPassword, cNewPassword, cConnClass *C.char
	if !(P.Username == "" && P.Password == "") {
		cUserName, cPassword = C.CString(P.Username), C.CString(P.Password)
	}
	var cSid *C.char
	if P.SID != "" {
		cSid = C.CString(P.SID)
	}
	defer func() {
		if cUserName != nil {
			C.free(unsafe.Pointer(cUserName))
			C.free(unsafe.Pointer(cPassword))
		}
		if cNewPassword != nil {
			C.free(unsafe.Pointer(cNewPassword))
		}
		if cSid != nil {
			C.free(unsafe.Pointer(cSid))
		}
		if cConnClass != nil {
			C.free(unsafe.Pointer(cConnClass))
		}
	}()
	var commonCreateParams C.dpiCommonCreateParams
	if C.dpiContext_initCommonCreateParams(d.dpiContext, &commonCreateParams) == C.DPI_FAILURE {
		return nil, errors.Errorf("initCommonCreateParams: %w", d.getError())
	}
	commonCreateParams.createMode = C.DPI_MODE_CREATE_DEFAULT | C.DPI_MODE_CREATE_THREADED
	if P.EnableEvents {
		commonCreateParams.createMode |= C.DPI_MODE_CREATE_EVENTS
	}
	commonCreateParams.encoding = cUTF8
	commonCreateParams.nencoding = cUTF8
	commonCreateParams.driverName = cDriverName
	commonCreateParams.driverNameLength = C.uint32_t(len(DriverName))

	if P.IsSysDBA || P.IsSysOper || P.IsSysASM || P.IsPrelim || P.StandaloneConnection {
		// no pool
		c.connParams = P
		return c, c.acquireConn(P.Username, P.Password, P.ConnClass)
	}
	var poolCreateParams C.dpiPoolCreateParams
	if C.dpiContext_initPoolCreateParams(d.dpiContext, &poolCreateParams) == C.DPI_FAILURE {
		return nil, errors.Errorf("initPoolCreateParams: %w", d.getError())
	}
	poolCreateParams.minSessions = DefaultPoolMinSessions
	if P.MinSessions >= 0 {
		poolCreateParams.minSessions = C.uint32_t(P.MinSessions)
	}
	poolCreateParams.maxSessions = DefaultPoolMaxSessions
	if P.MaxSessions > 0 {
		poolCreateParams.maxSessions = C.uint32_t(P.MaxSessions)
	}
	poolCreateParams.sessionIncrement = DefaultPoolIncrement
	if P.PoolIncrement > 0 {
		poolCreateParams.sessionIncrement = C.uint32_t(P.PoolIncrement)
	}
	if extAuth == 1 || P.HeterogeneousPool {
		poolCreateParams.homogeneous = 0
	}
	poolCreateParams.externalAuth = extAuth
	poolCreateParams.getMode = C.DPI_MODE_POOL_GET_TIMEDWAIT
	poolCreateParams.timeout = C.uint32_t(DefaultSessionTimeout / time.Second)
	if P.SessionTimeout > time.Second {
		poolCreateParams.timeout = C.uint32_t(P.SessionTimeout / time.Second) // seconds before idle pool sessions get evicted
	}
	poolCreateParams.waitTimeout = C.uint32_t(DefaultWaitTimeout / time.Millisecond)
	if P.WaitTimeout > time.Millisecond {
		poolCreateParams.waitTimeout = C.uint32_t(P.WaitTimeout / time.Millisecond) // milliseconds to wait for a session to become available
	}
	poolCreateParams.maxLifetimeSession = C.uint32_t(DefaultMaxLifeTime / time.Second)
	if P.MaxLifeTime > 0 {
		poolCreateParams.maxLifetimeSession = C.uint32_t(P.MaxLifeTime / time.Second) // maximum time in seconds till a pooled session may exist
	}

	var dp *C.dpiPool
	if Log != nil {
		Log("C", "dpiPool_create", "username", P.Username, "conn", connString, "sid", P.SID, "common", commonCreateParams, "pool", fmt.Sprintf("%#v", poolCreateParams))
	}
	if C.dpiPool_create(
		d.dpiContext,
		cUserName, C.uint32_t(len(P.Username)),
		cPassword, C.uint32_t(len(P.Password)),
		cSid, C.uint32_t(len(P.SID)),
		&commonCreateParams,
		&poolCreateParams,
		(**C.dpiPool)(unsafe.Pointer(&dp)),
	) == C.DPI_FAILURE {
		return nil, errors.Errorf("params=%s extAuth=%v: %w", P.String(), extAuth, d.getError())
	}
	C.dpiPool_setStmtCacheSize(dp, 40)
	pool := &connPool{dpiPool: dp}
	d.mu.Lock()
	d.pools[connString] = pool
	d.mu.Unlock()

	return c, pool.acquireConn(c, P)
}

func (dp *connPool) acquireConn(c *conn, P ConnectionParams) error {
	P.Comb()
	c.mu.Lock()
	c.connParams = P
	c.Client, c.Server = c.drv.clientVersion, dp.serverVersion
	c.timeZone, c.tzOffSecs = dp.timeZone, dp.tzOffSecs
	c.mu.Unlock()

	var connCreateParams C.dpiConnCreateParams
	if C.dpiContext_initConnCreateParams(c.drv.dpiContext, &connCreateParams) == C.DPI_FAILURE {
		return errors.Errorf("initConnCreateParams: %w", c.drv.getError())
	}
	if P.ConnClass != "" {
		cConnClass := C.CString(P.ConnClass)
		defer C.free(unsafe.Pointer(cConnClass))
		connCreateParams.connectionClass = cConnClass
		connCreateParams.connectionClassLength = C.uint32_t(len(P.ConnClass))
	}
	dc := C.malloc(C.sizeof_void)
	if C.dpiPool_acquireConnection(
		dp.dpiPool,
		nil, 0, nil, 0,
		&connCreateParams,
		(**C.dpiConn)(unsafe.Pointer(&dc)),
	) == C.DPI_FAILURE {
		C.free(unsafe.Pointer(dc))
		return errors.Errorf("acquirePoolConnection(user=%q, params=%#v): %w", P.Username, connCreateParams, c.getError())
	}

	c.mu.Lock()
	c.dpiConn = (*C.dpiConn)(dc)
	c.currentUser = P.Username
	c.newSession = connCreateParams.outNewSession == 1
	c.mu.Unlock()
	err := c.init(P.OnInit)
	if err == nil {
		c.mu.Lock()
		dp.serverVersion = c.Server
		dp.timeZone, dp.tzOffSecs = c.timeZone, c.tzOffSecs
		c.mu.Unlock()
	}

	return err
}

func (c *conn) acquireConn(user, pass, connClass string) error {
	P := c.connParams
	if !(P.IsSysDBA || P.IsSysOper || P.IsSysASM || P.IsPrelim || P.StandaloneConnection) {
		c.drv.mu.Lock()
		pool := c.drv.pools[P.String()]
		if Log != nil {
			Log("pools", c.drv.pools, "drv", fmt.Sprintf("%p", c.drv))
		}
		c.drv.mu.Unlock()
		if pool != nil {
			P.Username, P.Password, P.ConnClass = user, pass, connClass
			return pool.acquireConn(c, P)
		}
	}

	var connCreateParams C.dpiConnCreateParams
	if C.dpiContext_initConnCreateParams(c.drv.dpiContext, &connCreateParams) == C.DPI_FAILURE {
		return errors.Errorf("initConnCreateParams: %w", c.drv.getError())
	}
	var cUserName, cPassword, cNewPassword, cConnClass, cSid *C.char
	defer func() {
		if cUserName != nil {
			C.free(unsafe.Pointer(cUserName))
		}
		if cPassword != nil {
			C.free(unsafe.Pointer(cPassword))
		}
		if cNewPassword != nil {
			C.free(unsafe.Pointer(cNewPassword))
		}
		if cConnClass != nil {
			C.free(unsafe.Pointer(cConnClass))
		}
		if cSid != nil {
			C.free(unsafe.Pointer(cSid))
		}
	}()
	if user != "" {
		cUserName = C.CString(user)
	}
	if pass != "" {
		cPassword = C.CString(pass)
	}
	if connClass != "" {
		cConnClass = C.CString(connClass)
		connCreateParams.connectionClass = cConnClass
		connCreateParams.connectionClassLength = C.uint32_t(len(connClass))
	}
	var commonCreateParams C.dpiCommonCreateParams
	if C.dpiContext_initCommonCreateParams(c.drv.dpiContext, &commonCreateParams) == C.DPI_FAILURE {
		return errors.Errorf("initCommonCreateParams: %w", c.drv.getError())
	}
	commonCreateParams.createMode = C.DPI_MODE_CREATE_DEFAULT | C.DPI_MODE_CREATE_THREADED
	if P.EnableEvents {
		commonCreateParams.createMode |= C.DPI_MODE_CREATE_EVENTS
	}
	commonCreateParams.encoding = cUTF8
	commonCreateParams.nencoding = cUTF8
	commonCreateParams.driverName = cDriverName
	commonCreateParams.driverNameLength = C.uint32_t(len(DriverName))

	if P.SID != "" {
		cSid = C.CString(P.SID)
	}
	connCreateParams.authMode = P.authMode()
	extAuth := C.int(b2i(user == "" && pass == ""))
	connCreateParams.externalAuth = extAuth
	if P.NewPassword != "" {
		cNewPassword = C.CString(P.NewPassword)
		connCreateParams.newPassword = cNewPassword
		connCreateParams.newPasswordLength = C.uint32_t(len(P.NewPassword))
	}
	if Log != nil {
		Log("C", "dpiConn_create", "params", P.String(), "common", commonCreateParams, "conn", connCreateParams)
	}
	dc := C.malloc(C.sizeof_void)
	if C.dpiConn_create(
		c.drv.dpiContext,
		cUserName, C.uint32_t(len(user)),
		cPassword, C.uint32_t(len(pass)),
		cSid, C.uint32_t(len(P.SID)),
		&commonCreateParams,
		&connCreateParams,
		(**C.dpiConn)(unsafe.Pointer(&dc)),
	) == C.DPI_FAILURE {
		C.free(unsafe.Pointer(dc))
		return errors.Errorf("username=%q sid=%q params=%+v: %w", user, P.SID, connCreateParams, c.drv.getError())
	}
	c.mu.Lock()
	c.dpiConn = (*C.dpiConn)(dc)
	c.currentUser = user
	c.newSession = true
	P.Username, P.Password, P.ConnClass = user, pass, connClass
	if P.NewPassword != "" {
		P.Password, P.NewPassword = P.NewPassword, ""
	}
	c.connParams = P
	c.mu.Unlock()
	return c.init(P.OnInit)
}

// ConnectionParams holds the params for a connection (pool).
// You can use ConnectionParams{...}.StringWithPassword()
// as a connection string in sql.Open.
type ConnectionParams struct {
	OnInit                             []string
	Username, Password, SID, ConnClass string
	// NewPassword is used iff StandaloneConnection is true!
	NewPassword                              string
	MinSessions, MaxSessions, PoolIncrement  int
	WaitTimeout, MaxLifeTime, SessionTimeout time.Duration
	Timezone                                 *time.Location
	IsSysDBA, IsSysOper, IsSysASM, IsPrelim  bool
	HeterogeneousPool                        bool
	StandaloneConnection                     bool
	EnableEvents                             bool
}

// String returns the string representation of ConnectionParams.
// The password is replaced with a "SECRET" string!
func (P ConnectionParams) String() string {
	return P.string(true, false)
}

// StringNoClass returns the string representation of ConnectionParams, without class info.
// The password is replaced with a "SECRET" string!
func (P ConnectionParams) StringNoClass() string {
	return P.string(false, false)
}

// StringWithPassword returns the string representation of ConnectionParams (as String() does),
// but does NOT obfuscate the password, just prints it as is.
func (P ConnectionParams) StringWithPassword() string {
	return P.string(true, true)
}

func (P ConnectionParams) string(class, withPassword bool) string {
	host, path := P.SID, ""
	if i := strings.IndexByte(host, '/'); i >= 0 {
		host, path = host[:i], host[i:]
	}
	q := make(url.Values, 32)
	s := P.ConnClass
	if !class {
		s = ""
	}
	q.Add("connectionClass", s)

	password := P.Password
	if withPassword {
		q.Add("newPassword", P.NewPassword)
	} else {
		hsh := fnv.New64()
		io.WriteString(hsh, P.Password)
		password = "SECRET-" + base64.URLEncoding.EncodeToString(hsh.Sum(nil))
		if P.NewPassword != "" {
			hsh.Reset()
			io.WriteString(hsh, P.NewPassword)
			q.Add("newPassword", "SECRET-"+base64.URLEncoding.EncodeToString(hsh.Sum(nil)))
		}
	}
	s = ""
	if P.Timezone != nil {
		s = P.Timezone.String()
	}
	q.Add("timezone", s)
	B := func(b bool) string {
		if b {
			return "1"
		}
		return "0"
	}
	q.Add("poolMinSessions", strconv.Itoa(P.MinSessions))
	q.Add("poolMaxSessions", strconv.Itoa(P.MaxSessions))
	q.Add("poolIncrement", strconv.Itoa(P.PoolIncrement))
	q.Add("sysdba", B(P.IsSysDBA))
	q.Add("sysoper", B(P.IsSysOper))
	q.Add("sysasm", B(P.IsSysASM))
	q.Add("standaloneConnection", B(P.StandaloneConnection))
	q.Add("enableEvents", B(P.EnableEvents))
	q.Add("heterogeneousPool", B(P.HeterogeneousPool))
	q.Add("prelim", B(P.IsPrelim))
	q.Add("poolWaitTimeout", P.WaitTimeout.String())
	q.Add("poolSessionMaxLifetime", P.MaxLifeTime.String())
	q.Add("poolSessionTimeout", P.SessionTimeout.String())
	q["onInit"] = P.OnInit
	return (&url.URL{
		Scheme:   "oracle",
		User:     url.UserPassword(P.Username, password),
		Host:     host,
		Path:     path,
		RawQuery: q.Encode(),
	}).String()
}

func (P *ConnectionParams) Comb() {
	P.StandaloneConnection = P.StandaloneConnection || P.ConnClass == NoConnectionPoolingConnectionClass
	if P.IsPrelim || P.StandaloneConnection {
		// Prelim: the shared memory may not exist when Oracle is shut down.
		P.ConnClass = ""
		P.HeterogeneousPool = false
	}
}

// ParseConnString parses the given connection string into a struct.
func ParseConnString(connString string) (ConnectionParams, error) {
	P := ConnectionParams{
		MinSessions:    DefaultPoolMinSessions,
		MaxSessions:    DefaultPoolMaxSessions,
		PoolIncrement:  DefaultPoolIncrement,
		ConnClass:      DefaultConnectionClass,
		MaxLifeTime:    DefaultMaxLifeTime,
		WaitTimeout:    DefaultWaitTimeout,
		SessionTimeout: DefaultSessionTimeout,
	}
	if !strings.HasPrefix(connString, "oracle://") {
		i := strings.IndexByte(connString, '/')
		if i < 0 {
			return P, errors.New("no '/' in connection string")
		}
		P.Username, connString = connString[:i], connString[i+1:]

		uSid := strings.ToUpper(connString)
		//fmt.Printf("connString=%q SID=%q\n", connString, uSid)
		if strings.Contains(uSid, " AS ") {
			if P.IsSysDBA = strings.HasSuffix(uSid, " AS SYSDBA"); P.IsSysDBA {
				connString = connString[:len(connString)-10]
			} else if P.IsSysOper = strings.HasSuffix(uSid, " AS SYSOPER"); P.IsSysOper {
				connString = connString[:len(connString)-11]
			} else if P.IsSysASM = strings.HasSuffix(uSid, " AS SYSASM"); P.IsSysASM {
				connString = connString[:len(connString)-10]
			}
		}
		if i = strings.IndexByte(connString, '@'); i >= 0 {
			P.Password, P.SID = connString[:i], connString[i+1:]
		} else {
			P.Password = connString
		}
		if strings.HasSuffix(P.SID, ":POOLED") {
			P.ConnClass, P.SID = "POOLED", P.SID[:len(P.SID)-7]
		}
		//fmt.Printf("connString=%q params=%s\n", connString, P)
		return P, nil
	}
	u, err := url.Parse(connString)
	if err != nil {
		return P, errors.Errorf("%s: %w", connString, err)
	}
	if usr := u.User; usr != nil {
		P.Username = usr.Username()
		P.Password, _ = usr.Password()
	}
	P.SID = u.Hostname()
	// IPv6 literal address brackets are removed by u.Hostname,
	// so we have to put them back
	if strings.HasPrefix(u.Host, "[") && !strings.Contains(P.SID[1:], "]") {
		P.SID = "[" + P.SID + "]"
	}
	if u.Port() != "" {
		P.SID += ":" + u.Port()
	}
	if u.Path != "" && u.Path != "/" {
		P.SID += u.Path
	}
	q := u.Query()
	if vv, ok := q["connectionClass"]; ok {
		P.ConnClass = vv[0]
	}
	for _, task := range []struct {
		Dest *bool
		Key  string
	}{
		{&P.IsSysDBA, "sysdba"},
		{&P.IsSysOper, "sysoper"},
		{&P.IsSysASM, "sysasm"},
		{&P.IsPrelim, "prelim"},

		{&P.StandaloneConnection, "standaloneConnection"},
		{&P.EnableEvents, "enableEvents"},
		{&P.HeterogeneousPool, "heterogeneousPool"},
	} {
		*task.Dest = q.Get(task.Key) == "1"
	}
	if tz := q.Get("timezone"); tz != "" {
		if tz == "local" {
			P.Timezone = time.Local
		} else if strings.Contains(tz, "/") {
			if P.Timezone, err = time.LoadLocation(tz); err != nil {
				return P, errors.Errorf("%s: %w", tz, err)
			}
		} else if off, err := parseTZ(tz); err == nil {
			P.Timezone = time.FixedZone(tz, off)
		} else {
			return P, errors.Errorf("%s: %w", tz, err)
		}
	}

	for _, task := range []struct {
		Dest *int
		Key  string
	}{
		{&P.MinSessions, "poolMinSessions"},
		{&P.MaxSessions, "poolMaxSessions"},
		{&P.PoolIncrement, "poolIncrement"},
	} {
		s := q.Get(task.Key)
		if s == "" {
			continue
		}
		var err error
		*task.Dest, err = strconv.Atoi(s)
		if err != nil {
			return P, errors.Errorf("%s: %w", task.Key+"="+s, err)
		}
	}
	for _, task := range []struct {
		Dest *time.Duration
		Key  string
	}{
		{&P.SessionTimeout, "poolSessionTimeout"},
		{&P.WaitTimeout, "poolWaitTimeout"},
		{&P.MaxLifeTime, "poolSessionMaxLifetime"},
	} {
		s := q.Get(task.Key)
		if s == "" {
			continue
		}
		var err error
		*task.Dest, err = time.ParseDuration(s)
		if err != nil {
			if !strings.Contains(err.Error(), "time: missing unit in duration") {
				return P, errors.Errorf("%s: %w", task.Key+"="+s, err)
			}
			i, err := strconv.Atoi(s)
			if err != nil {
				return P, errors.Errorf("%s: %w", task.Key+"="+s, err)
			}
			base := time.Second
			if task.Key == "poolWaitTimeout" {
				base = time.Millisecond
			}
			*task.Dest = time.Duration(i) * base
		}
	}
	if P.MinSessions > P.MaxSessions {
		P.MinSessions = P.MaxSessions
	}
	if P.MinSessions == P.MaxSessions {
		P.PoolIncrement = 0
	} else if P.PoolIncrement < 1 {
		P.PoolIncrement = 1
	}
	P.OnInit = q["onInit"]

	P.Comb()
	if P.StandaloneConnection {
		P.NewPassword = q.Get("newPassword")
	}

	return P, nil
}

// SetSessionParamOnInit adds an "ALTER SESSION k=v" to the OnInit task list.
func (P *ConnectionParams) SetSessionParamOnInit(k, v string) {
	P.OnInit = append(P.OnInit, fmt.Sprintf("ALTER SESSION SET %s = q'(%s)'", k, strings.Replace(v, "'", "''", -1)))
}

func (P ConnectionParams) authMode() C.dpiAuthMode {
	authMode := C.dpiAuthMode(C.DPI_MODE_AUTH_DEFAULT)
	// OR all the modes together
	for _, elt := range []struct {
		Is   bool
		Mode C.dpiAuthMode
	}{
		{P.IsSysDBA, C.DPI_MODE_AUTH_SYSDBA},
		{P.IsSysOper, C.DPI_MODE_AUTH_SYSOPER},
		{P.IsSysASM, C.DPI_MODE_AUTH_SYSASM},
		{P.IsPrelim, C.DPI_MODE_AUTH_PRELIM},
	} {
		if elt.Is {
			authMode |= elt.Mode
		}
	}
	return authMode
}

// OraErr is an error holding the ORA-01234 code and the message.
type OraErr struct {
	message string
	code    int
}

// AsOraErr returns the underlying *OraErr and whether it succeeded.
func AsOraErr(err error) (*OraErr, bool) {
	var oerr *OraErr
	ok := errors.As(err, &oerr)
	return oerr, ok
}

var _ = error((*OraErr)(nil))

// Code returns the OraErr's error code.
func (oe *OraErr) Code() int { return oe.code }

// Message returns the OraErr's message.
func (oe *OraErr) Message() string { return oe.message }
func (oe *OraErr) Error() string {
	msg := oe.Message()
	if oe.code == 0 && msg == "" {
		return ""
	}
	return fmt.Sprintf("ORA-%05d: %s", oe.code, oe.message)
}
func fromErrorInfo(errInfo C.dpiErrorInfo) *OraErr {
	oe := OraErr{
		code:    int(errInfo.code),
		message: strings.TrimSpace(C.GoString(errInfo.message)),
	}
	if oe.code == 0 && strings.HasPrefix(oe.message, "ORA-") &&
		len(oe.message) > 9 && oe.message[9] == ':' {
		if i, _ := strconv.Atoi(oe.message[4:9]); i > 0 {
			oe.code = i
		}
	}
	oe.message = strings.TrimPrefix(oe.message, fmt.Sprintf("ORA-%05d: ", oe.Code()))
	return &oe
}

// newErrorInfo is just for testing: testing cannot use Cgo...
func newErrorInfo(code int, message string) C.dpiErrorInfo {
	return C.dpiErrorInfo{code: C.int32_t(code), message: C.CString(message)}
}

// against deadcode
var _ = newErrorInfo

func (d *drv) getError() *OraErr {
	if d == nil || d.dpiContext == nil {
		return &OraErr{code: -12153, message: driver.ErrBadConn.Error()}
	}
	var errInfo C.dpiErrorInfo
	C.dpiContext_getError(d.dpiContext, &errInfo)
	return fromErrorInfo(errInfo)
}

func b2i(b bool) uint8 {
	if b {
		return 1
	}
	return 0
}

// VersionInfo holds version info returned by Oracle DB.
type VersionInfo struct {
	ServerRelease                                           string
	Version, Release, Update, PortRelease, PortUpdate, Full uint8
}

func (V *VersionInfo) set(v *C.dpiVersionInfo) {
	*V = VersionInfo{
		Version: uint8(v.versionNum),
		Release: uint8(v.releaseNum), Update: uint8(v.updateNum),
		PortRelease: uint8(v.portReleaseNum), PortUpdate: uint8(v.portUpdateNum),
		Full: uint8(v.fullVersionNum),
	}
}
func (V VersionInfo) String() string {
	var s string
	if V.ServerRelease != "" {
		s = " [" + V.ServerRelease + "]"
	}
	return fmt.Sprintf("%d.%d.%d.%d.%d%s", V.Version, V.Release, V.Update, V.PortRelease, V.PortUpdate, s)
}

var timezones = make(map[[2]C.int8_t]*time.Location)
var timezonesMu sync.RWMutex

func timeZoneFor(hourOffset, minuteOffset C.int8_t) *time.Location {
	if hourOffset == 0 && minuteOffset == 0 {
		return time.UTC
	}
	key := [2]C.int8_t{hourOffset, minuteOffset}
	timezonesMu.RLock()
	tz := timezones[key]
	timezonesMu.RUnlock()
	if tz == nil {
		timezonesMu.Lock()
		if tz = timezones[key]; tz == nil {
			tz = time.FixedZone(
				fmt.Sprintf("%02d:%02d", hourOffset, minuteOffset),
				int(hourOffset)*3600+int(minuteOffset)*60,
			)
			timezones[key] = tz
		}
		timezonesMu.Unlock()
	}
	return tz
}

type ctxKey string

const logCtxKey = ctxKey("godror.Log")

type logFunc func(...interface{}) error

func ctxGetLog(ctx context.Context) logFunc {
	if lgr, ok := ctx.Value(logCtxKey).(func(...interface{}) error); ok {
		return lgr
	}
	return Log
}

// ContextWithLog returns a context with the given log function.
func ContextWithLog(ctx context.Context, logF func(...interface{}) error) context.Context {
	return context.WithValue(ctx, logCtxKey, logF)
}

var _ = driver.DriverContext((*drv)(nil))
var _ = driver.Connector((*connector)(nil))

type connector struct {
	drv    *drv
	onInit func(driver.Conn) error
	ConnectionParams
}

// OpenConnector must parse the name in the same format that Driver.Open
// parses the name parameter.
func (d *drv) OpenConnector(name string) (driver.Connector, error) {
	P, err := ParseConnString(name)
	if err != nil {
		return nil, err
	}

	return connector{ConnectionParams: P, drv: d}, nil
}

// Connect returns a connection to the database.
// Connect may return a cached connection (one previously
// closed), but doing so is unnecessary; the sql package
// maintains a pool of idle connections for efficient re-use.
//
// The provided context.Context is for dialing purposes only
// (see net.DialContext) and should not be stored or used for
// other purposes.
//
// The returned connection is only used by one goroutine at a
// time.
func (c connector) Connect(context.Context) (driver.Conn, error) {
	conn, err := c.drv.openConn(c.ConnectionParams)
	if err != nil || c.onInit == nil || !conn.newSession {
		return conn, err
	}
	if err = c.onInit(conn); err != nil {
		conn.close(true)
		return nil, err
	}
	return conn, nil
}

// Driver returns the underlying Driver of the Connector,
// mainly to maintain compatibility with the Driver method
// on sql.DB.
func (c connector) Driver() driver.Driver { return c.drv }

// NewConnector returns a driver.Connector to be used with sql.OpenDB,
// which calls the given onInit if the connection is new.
//
// For an onInit example, see NewSessionIniter.
func (d *drv) NewConnector(name string, onInit func(driver.Conn) error) (driver.Connector, error) {
	cxr, err := d.OpenConnector(name)
	if err != nil {
		return nil, err
	}
	cx := cxr.(connector)
	cx.onInit = onInit
	return cx, err
}

// NewConnector returns a driver.Connector to be used with sql.OpenDB,
// (for the default Driver registered with godror)
// which calls the given onInit if the connection is new.
//
// For an onInit example, see NewSessionIniter.
func NewConnector(name string, onInit func(driver.Conn) error) (driver.Connector, error) {
	return defaultDrv.NewConnector(name, onInit)
}

// NewSessionIniter returns a function suitable for use in NewConnector as onInit,
// which calls "ALTER SESSION SET <key>='<value>'" for each element of the given map.
func NewSessionIniter(m map[string]string) func(driver.Conn) error {
	return func(cx driver.Conn) error {
		for k, v := range m {
			qry := fmt.Sprintf("ALTER SESSION SET %s = q'(%s)'", k, strings.Replace(v, "'", "''", -1))
			st, err := cx.Prepare(qry)
			if err != nil {
				return errors.Errorf("%s: %w", qry, err)
			}
			_, err = st.Exec(nil) //lint:ignore SA1019 it's hard to use ExecContext here
			st.Close()
			if err != nil {
				return err
			}
		}
		return nil
	}
}
