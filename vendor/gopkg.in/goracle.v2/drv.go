// Copyright 2019 Tamás Gulácsi
//
//
// SPDX-License-Identifier: UPL-1.0 OR Apache-2.0

// Package goracle is a database/sql/driver for Oracle DB.
//
// The connection string for the sql.Open("goracle", connString) call can be
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
//     newPassword=
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
package goracle

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
	"math"
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
	DriverName = "gopkg.in/goracle.v2 : " + Version

	// DefaultPoolMinSessions specifies the default value for minSessions for pool creation.
	DefaultPoolMinSessions = 1
	// DefaultPoolMaxSessions specifies the default value for maxSessions for pool creation.
	DefaultPoolMaxSessions = 1000
	// DefaultPoolIncrement specifies the default value for increment for pool creation.
	DefaultPoolIncrement = 1
	// DefaultConnectionClass is the default connectionClass
	DefaultConnectionClass = "GORACLE"
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

// Number as string
type Number string

var (
	// Int64 for converting to-from int64.
	Int64 = intType{}
	// Float64 for converting to-from float64.
	Float64 = floatType{}
	// Num for converting to-from Number (string)
	Num = numType{}
)

type intType struct{}

func (intType) String() string { return "Int64" }
func (intType) ConvertValue(v interface{}) (driver.Value, error) {
	if Log != nil {
		Log("ConvertValue", "Int64", "value", v)
	}
	switch x := v.(type) {
	case int8:
		return int64(x), nil
	case int16:
		return int64(x), nil
	case int32:
		return int64(x), nil
	case int64:
		return x, nil
	case uint16:
		return int64(x), nil
	case uint32:
		return int64(x), nil
	case uint64:
		return int64(x), nil
	case float32:
		if _, f := math.Modf(float64(x)); f != 0 {
			return int64(x), errors.Errorf("non-zero fractional part: %f", f)
		}
		return int64(x), nil
	case float64:
		if _, f := math.Modf(x); f != 0 {
			return int64(x), errors.Errorf("non-zero fractional part: %f", f)
		}
		return int64(x), nil
	case string:
		if x == "" {
			return 0, nil
		}
		return strconv.ParseInt(x, 10, 64)
	case Number:
		if x == "" {
			return 0, nil
		}
		return strconv.ParseInt(string(x), 10, 64)
	default:
		return nil, errors.Errorf("unknown type %T", v)
	}
}

type floatType struct{}

func (floatType) String() string { return "Float64" }
func (floatType) ConvertValue(v interface{}) (driver.Value, error) {
	if Log != nil {
		Log("ConvertValue", "Float64", "value", v)
	}
	switch x := v.(type) {
	case int8:
		return float64(x), nil
	case int16:
		return float64(x), nil
	case int32:
		return float64(x), nil
	case uint16:
		return float64(x), nil
	case uint32:
		return float64(x), nil
	case int64:
		return float64(x), nil
	case uint64:
		return float64(x), nil
	case float32:
		return float64(x), nil
	case float64:
		return x, nil
	case string:
		if x == "" {
			return 0, nil
		}
		return strconv.ParseFloat(x, 64)
	case Number:
		if x == "" {
			return 0, nil
		}
		return strconv.ParseFloat(string(x), 64)
	default:
		return nil, errors.Errorf("unknown type %T", v)
	}
}

type numType struct{}

func (numType) String() string { return "Num" }
func (numType) ConvertValue(v interface{}) (driver.Value, error) {
	if Log != nil {
		Log("ConvertValue", "Num", "value", v)
	}
	switch x := v.(type) {
	case string:
		if x == "" {
			return 0, nil
		}
		return x, nil
	case Number:
		if x == "" {
			return 0, nil
		}
		return string(x), nil
	case int8, int16, int32, int64, uint16, uint32, uint64:
		return fmt.Sprintf("%d", x), nil
	case float32, float64:
		return fmt.Sprintf("%f", x), nil
	default:
		return nil, errors.Errorf("unknown type %T", v)
	}
}
func (n Number) String() string { return string(n) }

// Value returns the Number as driver.Value
func (n Number) Value() (driver.Value, error) {
	return string(n), nil
}

// Scan into the Number from a driver.Value.
func (n *Number) Scan(v interface{}) error {
	if v == nil {
		*n = ""
		return nil
	}
	switch x := v.(type) {
	case string:
		*n = Number(x)
	case Number:
		*n = x
	case int8, int16, int32, int64, uint16, uint32, uint64:
		*n = Number(fmt.Sprintf("%d", x))
	case float32, float64:
		*n = Number(fmt.Sprintf("%f", x))
	default:
		return errors.Errorf("unknown type %T", v)
	}
	return nil
}

// MarshalText marshals a Number to text.
func (n Number) MarshalText() ([]byte, error) { return []byte(n), nil }

// UnmarshalText parses text into a Number.
func (n *Number) UnmarshalText(p []byte) error {
	var dotNum int
	for i, c := range p {
		if !(c == '-' && i == 0 || '0' <= c && c <= '9') {
			if c == '.' {
				dotNum++
				if dotNum == 1 {
					continue
				}
			}
			return errors.Errorf("unknown char %c in %q", c, p)
		}
	}
	*n = Number(p)
	return nil
}

// MarshalJSON marshals a Number into a JSON string.
func (n Number) MarshalJSON() ([]byte, error) {
	b, err := n.MarshalText()
	b2 := make([]byte, 1, 1+len(b)+1)
	b2[0] = '"'
	b2 = append(b2, b...)
	b2 = append(b2, '"')
	return b2, err
}

// UnmarshalJSON parses a JSON string into the Number.
func (n *Number) UnmarshalJSON(p []byte) error {
	*n = Number("")
	if len(p) == 0 {
		return nil
	}
	if len(p) > 2 && p[0] == '"' && p[len(p)-1] == '"' {
		p = p[1 : len(p)-1]
	}
	return n.UnmarshalText(p)
}

// Log function. By default, it's nil, and thus logs nothing.
// If you want to change this, change it to a github.com/go-kit/kit/log.Swapper.Log
// or analog to be race-free.
var Log func(...interface{}) error

var defaultDrv *drv

func init() {
	defaultDrv = newDrv()
	sql.Register("goracle", defaultDrv)
}

func newDrv() *drv {
	return &drv{pools: make(map[string]*connPool)}
}

var _ = driver.Driver((*drv)(nil))

type drv struct {
	clientVersion VersionInfo
	mu            sync.Mutex
	dpiContext    *C.dpiContext
	pools         map[string]*connPool
}

type connPool struct {
	dpiPool       *C.dpiPool
	serverVersion VersionInfo
	timeZone      *time.Location
	tzOffSecs     int
}

func (d *drv) init() error {
	d.mu.Lock()
	defer d.mu.Unlock()
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

func (d *drv) openConn(P ConnectionParams) (*conn, error) {
	if err := d.init(); err != nil {
		return nil, err
	}

	c := conn{drv: d, connParams: P, timeZone: time.Local}
	connString := P.String()

	defer func() {
		d.mu.Lock()
		if Log != nil {
			Log("pools", d.pools, "conn", P.String())
		}
		d.mu.Unlock()
	}()

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
	P.StandaloneConnection = P.StandaloneConnection || P.ConnClass == NoConnectionPoolingConnectionClass
	if P.IsPrelim || P.StandaloneConnection {
		// Prelim: the shared memory may not exist when Oracle is shut down.
		P.ConnClass = ""
	}

	extAuth := C.int(b2i(P.Username == "" && P.Password == ""))
	var connCreateParams C.dpiConnCreateParams
	if C.dpiContext_initConnCreateParams(d.dpiContext, &connCreateParams) == C.DPI_FAILURE {
		return nil, errors.Errorf("initConnCreateParams: %w", d.getError())
	}
	connCreateParams.authMode = authMode
	connCreateParams.externalAuth = extAuth
	if P.ConnClass != "" {
		cConnClass := C.CString(P.ConnClass)
		defer C.free(unsafe.Pointer(cConnClass))
		connCreateParams.connectionClass = cConnClass
		connCreateParams.connectionClassLength = C.uint32_t(len(P.ConnClass))
	}
	if !(P.IsSysDBA || P.IsSysOper || P.IsSysASM || P.IsPrelim || P.StandaloneConnection) {
		d.mu.Lock()
		dp := d.pools[connString]
		d.mu.Unlock()
		if dp != nil {
			//Proxy authenticated connections to database will be provided by methods with context
			c.mu.Lock()
			c.Client, c.Server = d.clientVersion, dp.serverVersion
			c.timeZone, c.tzOffSecs = dp.timeZone, dp.tzOffSecs
			c.mu.Unlock()
			if err := c.acquireConn("", ""); err != nil {
				return nil, err
			}
			err := c.init()
			if err == nil {
				c.mu.Lock()
				dp.serverVersion = c.Server
				dp.timeZone, dp.tzOffSecs = c.timeZone, c.tzOffSecs
				c.mu.Unlock()
			}
			return &c, err
		}
	}

	var cUserName, cPassword, cNewPassword *C.char
	if !(P.Username == "" && P.Password == "") {
		cUserName, cPassword = C.CString(P.Username), C.CString(P.Password)
	}
	var cSid *C.char
	if P.SID != "" {
		cSid = C.CString(P.SID)
	}
	cUTF8, cConnClass := C.CString("AL32UTF8"), C.CString(P.ConnClass)
	cDriverName := C.CString(DriverName)
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
		C.free(unsafe.Pointer(cUTF8))
		C.free(unsafe.Pointer(cConnClass))
		C.free(unsafe.Pointer(cDriverName))
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
		if P.NewPassword != "" {
			cNewPassword = C.CString(P.NewPassword)
			connCreateParams.newPassword = cNewPassword
			connCreateParams.newPasswordLength = C.uint32_t(len(P.NewPassword))
		}
		dc := C.malloc(C.sizeof_void)
		if Log != nil {
			Log("C", "dpiConn_create", "params", P.String(), "common", commonCreateParams, "conn", connCreateParams)
		}
		if C.dpiConn_create(
			d.dpiContext,
			cUserName, C.uint32_t(len(P.Username)),
			cPassword, C.uint32_t(len(P.Password)),
			cSid, C.uint32_t(len(P.SID)),
			&commonCreateParams,
			&connCreateParams,
			(**C.dpiConn)(unsafe.Pointer(&dc)),
		) == C.DPI_FAILURE {
			C.free(unsafe.Pointer(dc))
			return nil, errors.Errorf("username=%q sid=%q params=%+v: %w", P.Username, P.SID, connCreateParams, d.getError())
		}
		c.dpiConn = (*C.dpiConn)(dc)
		c.currentUser = P.Username
		c.newSession = true
		err := c.init()
		return &c, err
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
	d.mu.Lock()
	d.pools[connString] = &connPool{dpiPool: dp}
	d.mu.Unlock()

	return d.openConn(P)
}

func (c *conn) acquireConn(user, pass string) error {
	var connCreateParams C.dpiConnCreateParams
	if C.dpiContext_initConnCreateParams(c.dpiContext, &connCreateParams) == C.DPI_FAILURE {
		return errors.Errorf("initConnCreateParams: %w", "", c.getError())
	}

	dc := C.malloc(C.sizeof_void)
	if Log != nil {
		Log("C", "dpiPool_acquirePoolConnection", "conn", connCreateParams)
	}
	var cUserName, cPassword *C.char
	defer func() {
		if cUserName != nil {
			C.free(unsafe.Pointer(cUserName))
		}
		if cPassword != nil {
			C.free(unsafe.Pointer(cPassword))
		}
	}()
	if user != "" {
		cUserName = C.CString(user)
	}
	if pass != "" {
		cPassword = C.CString(pass)
	}

	c.drv.mu.Lock()
	pool := c.pools[c.connParams.String()]
	c.drv.mu.Unlock()
	if C.dpiPool_acquireConnection(
		pool.dpiPool,
		cUserName, C.uint32_t(len(user)), cPassword, C.uint32_t(len(pass)),
		&connCreateParams,
		(**C.dpiConn)(unsafe.Pointer(&dc)),
	) == C.DPI_FAILURE {
		C.free(unsafe.Pointer(dc))
		return errors.Errorf("acquirePoolConnection: %w", c.getError())
	}

	c.mu.Lock()
	c.dpiConn = (*C.dpiConn)(dc)
	c.currentUser = user
	c.newSession = connCreateParams.outNewSession == 1
	c.Client, c.Server = c.drv.clientVersion, pool.serverVersion
	c.timeZone, c.tzOffSecs = pool.timeZone, pool.tzOffSecs
	c.mu.Unlock()
	err := c.init()
	if err == nil {
		c.mu.Lock()
		pool.serverVersion = c.Server
		pool.timeZone, pool.tzOffSecs = c.timeZone, c.tzOffSecs
		c.mu.Unlock()
	}

	return err
}

// ConnectionParams holds the params for a connection (pool).
// You can use ConnectionParams{...}.StringWithPassword()
// as a connection string in sql.Open.
type ConnectionParams struct {
	Username, Password, SID, ConnClass string
	// NewPassword is used iff StandaloneConnection is true!
	NewPassword                              string
	MinSessions, MaxSessions, PoolIncrement  int
	WaitTimeout, MaxLifeTime, SessionTimeout time.Duration
	IsSysDBA, IsSysOper, IsSysASM, IsPrelim  bool
	HeterogeneousPool                        bool
	StandaloneConnection                     bool
	EnableEvents                             bool
	Timezone                                 *time.Location
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
	return (&url.URL{
		Scheme:   "oracle",
		User:     url.UserPassword(P.Username, password),
		Host:     host,
		Path:     path,
		RawQuery: q.Encode(),
	}).String()
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

	P.StandaloneConnection = P.StandaloneConnection || P.ConnClass == NoConnectionPoolingConnectionClass
	if P.IsPrelim {
		P.ConnClass = ""
	}
	if P.StandaloneConnection {
		P.NewPassword = q.Get("newPassword")
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
	return P, nil
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
	Version, Release, Update, PortRelease, PortUpdate, Full int
}

func (V *VersionInfo) set(v *C.dpiVersionInfo) {
	*V = VersionInfo{
		Version: int(v.versionNum),
		Release: int(v.releaseNum), Update: int(v.updateNum),
		PortRelease: int(v.portReleaseNum), PortUpdate: int(v.portUpdateNum),
		Full: int(v.fullVersionNum),
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

const logCtxKey = ctxKey("goracle.Log")

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
