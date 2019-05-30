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
//     prelim=0
//
// These are the defaults. Many advocate that a static session pool (min=max, incr=0)
// is better, with 1-10 sessions per CPU thread.
// See http://docs.oracle.com/cd/E82638_01/JJUCP/optimizing-real-world-performance.htm#JJUCP-GUID-BC09F045-5D80-4AF5-93F5-FEF0531E0E1D
// You may also use ConnectionParams to configure a connection.
//
// If you specify connectionClass, that'll reuse the same session pool
// without the connectionClass, but will specify it on each session acquire.
// Thus you can cluster the session pool with classes, or use POOLED for DRCP.
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

	"github.com/pkg/errors"
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

func init() {
	sql.Register("goracle", newDrv())
}

func newDrv() *drv {
	return &drv{pools: make(map[string]*C.dpiPool)}
}

var _ = driver.Driver((*drv)(nil))

type drv struct {
	clientVersion VersionInfo
	mu            sync.Mutex
	dpiContext    *C.dpiContext
	pools         map[string]*C.dpiPool
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
		return errors.Wrap(d.getError(), "getClientVersion")
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
	return conn, maybeBadConn(err)
}

func (d *drv) ClientVersion() (VersionInfo, error) {
	return d.clientVersion, nil
}

func (d *drv) openConn(P ConnectionParams) (*conn, error) {
	if err := d.init(); err != nil {
		return nil, err
	}

	c := conn{drv: d, connParams: P}
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
	if P.IsPrelim {
		// The shared memory may not exist when Oracle is shut down.
		P.ConnClass = ""
	}

	extAuth := C.int(b2i(P.Username == "" && P.Password == ""))
	var connCreateParams C.dpiConnCreateParams
	if C.dpiContext_initConnCreateParams(d.dpiContext, &connCreateParams) == C.DPI_FAILURE {
		return nil, errors.Wrap(d.getError(), "initConnCreateParams")
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
			if P.HeterogeneousPool {
				//Proxy authenticated connections to database will be provided by methods with context
				return &c, nil
			}

			if err := c.acquireConn("", ""); err != nil {
				return nil, err
			}

			return &c, c.init()
		}
	}

	var cUserName, cPassword *C.char
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
		if cSid != nil {
			C.free(unsafe.Pointer(cSid))
		}
		C.free(unsafe.Pointer(cUTF8))
		C.free(unsafe.Pointer(cConnClass))
		C.free(unsafe.Pointer(cDriverName))
	}()
	var commonCreateParams C.dpiCommonCreateParams
	if C.dpiContext_initCommonCreateParams(d.dpiContext, &commonCreateParams) == C.DPI_FAILURE {
		return nil, errors.Wrap(d.getError(), "initCommonCreateParams")
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
		dc := C.malloc(C.sizeof_void)
		if Log != nil {
			Log("C", "dpiConn_create", "username", P.Username, "sid", P.SID, "common", commonCreateParams, "conn", connCreateParams)
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
			return nil, errors.Wrapf(d.getError(), "username=%q sid=%q params=%+v", P.Username, P.SID, connCreateParams)
		}
		c.dpiConn = (*C.dpiConn)(dc)
		return &c, c.init()
	}
	var poolCreateParams C.dpiPoolCreateParams
	if C.dpiContext_initPoolCreateParams(d.dpiContext, &poolCreateParams) == C.DPI_FAILURE {
		return nil, errors.Wrap(d.getError(), "initPoolCreateParams")
	}
	poolCreateParams.minSessions = C.uint32_t(P.MinSessions)
	poolCreateParams.maxSessions = C.uint32_t(P.MaxSessions)
	poolCreateParams.sessionIncrement = C.uint32_t(P.PoolIncrement)
	if extAuth == 1 || P.HeterogeneousPool {
		poolCreateParams.homogeneous = 0
	}
	poolCreateParams.externalAuth = extAuth
	poolCreateParams.getMode = C.DPI_MODE_POOL_GET_TIMEDWAIT
	poolCreateParams.timeout = 300             // seconds before idle pool sessions got evicted
	poolCreateParams.waitTimeout = 3 * 1000    // milliseconds to wait for a session become available
	poolCreateParams.maxLifetimeSession = 3600 // maximum time in seconds till a pooled session may exist

	var dp *C.dpiPool
	if Log != nil {
		Log("C", "dpiPool_create", "username", P.Username, "sid", P.SID, "common", commonCreateParams, "pool", poolCreateParams)
	}
	//fmt.Println("POOL create", connString)
	if C.dpiPool_create(
		d.dpiContext,
		cUserName, C.uint32_t(len(P.Username)),
		cPassword, C.uint32_t(len(P.Password)),
		cSid, C.uint32_t(len(P.SID)),
		&commonCreateParams,
		&poolCreateParams,
		(**C.dpiPool)(unsafe.Pointer(&dp)),
	) == C.DPI_FAILURE {
		return nil, errors.Wrapf(d.getError(), "username=%q SID=%q minSessions=%d maxSessions=%d poolIncrement=%d extAuth=%d ",
			P.Username, P.SID,
			P.MinSessions, P.MaxSessions, P.PoolIncrement, extAuth)
	}
	C.dpiPool_setStmtCacheSize(dp, 40)
	d.mu.Lock()
	d.pools[connString] = dp
	d.mu.Unlock()

	return d.openConn(P)
}

func (c *conn) acquireConn(user, pass string) error {
	var connCreateParams C.dpiConnCreateParams
	if C.dpiContext_initConnCreateParams(c.dpiContext, &connCreateParams) == C.DPI_FAILURE {
		return errors.Wrap(c.getError(), "initConnCreateParams")
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
		pool,
		cUserName, C.uint32_t(len(user)), cPassword, C.uint32_t(len(pass)), nil,
		(**C.dpiConn)(unsafe.Pointer(&dc)),
	) == C.DPI_FAILURE {
		C.free(unsafe.Pointer(dc))
		return errors.Wrapf(c.getError(), "acquirePoolConnection")
	}

	c.dpiConn = (*C.dpiConn)(dc)
	c.currentUser = user

	return nil
}

// ConnectionParams holds the params for a connection (pool).
// You can use ConnectionParams{...}.StringWithPassword()
// as a connection string in sql.Open.
type ConnectionParams struct {
	Username, Password, SID, ConnClass      string
	MinSessions, MaxSessions, PoolIncrement int
	IsSysDBA, IsSysOper, IsSysASM, IsPrelim bool
	HeterogeneousPool                       bool
	StandaloneConnection                    bool
	EnableEvents                            bool
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
	cc := ""
	if class {
		cc = fmt.Sprintf("connectionClass=%s&", url.QueryEscape(P.ConnClass))
	}
	// params should be sorted lexicographically
	password := P.Password
	if !withPassword {
		hsh := fnv.New64()
		io.WriteString(hsh, P.Password)
		password = "SECRET-" + base64.URLEncoding.EncodeToString(hsh.Sum(nil))
	}
	return (&url.URL{
		Scheme: "oracle",
		User:   url.UserPassword(P.Username, password),
		Host:   host,
		Path:   path,
		RawQuery: cc +
			fmt.Sprintf("poolIncrement=%d&poolMaxSessions=%d&poolMinSessions=%d&"+
				"sysdba=%d&sysoper=%d&sysasm=%d&"+
				"standaloneConnection=%d&enableEvents=%d&"+
				"heterogeneousPool=%d&prelim=%d",
				P.PoolIncrement, P.MaxSessions, P.MinSessions,
				b2i(P.IsSysDBA), b2i(P.IsSysOper), b2i(P.IsSysASM),
				b2i(P.StandaloneConnection), b2i(P.EnableEvents),
				b2i(P.HeterogeneousPool), b2i(P.IsPrelim),
			),
	}).String()
}

// ParseConnString parses the given connection string into a struct.
func ParseConnString(connString string) (ConnectionParams, error) {
	P := ConnectionParams{
		MinSessions:   DefaultPoolMinSessions,
		MaxSessions:   DefaultPoolMaxSessions,
		PoolIncrement: DefaultPoolIncrement,
		ConnClass:     DefaultConnectionClass,
	}
	if !strings.HasPrefix(connString, "oracle://") {
		i := strings.IndexByte(connString, '/')
		if i < 0 {
			return P, errors.Errorf("no '/' in connection string")
		}
		P.Username, connString = connString[:i], connString[i+1:]
		if i = strings.IndexByte(connString, '@'); i >= 0 {
			P.Password, P.SID = connString[:i], connString[i+1:]
		} else {
			P.Password = connString
		}
		uSid := strings.ToUpper(P.SID)
		if P.IsSysDBA = strings.HasSuffix(uSid, " AS SYSDBA"); P.IsSysDBA {
			P.SID = P.SID[:len(P.SID)-10]
		} else if P.IsSysOper = strings.HasSuffix(uSid, " AS SYSOPER"); P.IsSysOper {
			P.SID = P.SID[:len(P.SID)-11]
		} else if P.IsSysASM = strings.HasSuffix(uSid, " AS SYSASM"); P.IsSysASM {
			P.SID = P.SID[:len(P.SID)-10]
		}
		if strings.HasSuffix(P.SID, ":POOLED") {
			P.ConnClass, P.SID = "POOLED", P.SID[:len(P.SID)-7]
		}
		return P, nil
	}
	u, err := url.Parse(connString)
	if err != nil {
		return P, errors.Wrap(err, connString)
	}
	if usr := u.User; usr != nil {
		P.Username = usr.Username()
		P.Password, _ = usr.Password()
	}
	P.SID = u.Hostname()
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
	P.StandaloneConnection = P.StandaloneConnection || P.ConnClass == NoConnectionPoolingConnectionClass
	if P.IsPrelim {
		P.ConnClass = ""
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
			return P, errors.Wrap(err, task.Key+"="+s)
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
