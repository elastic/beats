// Copyright 2017 Tamás Gulácsi
//
//
// SPDX-License-Identifier: UPL-1.0 OR Apache-2.0

package godror

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"math"
	"strconv"
	"sync"
	"time"

	errors "golang.org/x/xerrors"
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

// QueryColumn is the described column.
type QueryColumn struct {
	Name                           string
	Type, Length, Precision, Scale int
	Nullable                       bool
	//Schema string
	//CharsetID, CharsetForm         int
}

// Execer is the ExecContext of sql.Conn.
type Execer interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
}

// Querier is the QueryContext of sql.Conn.
type Querier interface {
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
}

// DescribeQuery describes the columns in the qry.
//
// This can help using unknown-at-compile-time, a.k.a.
// dynamic queries.
func DescribeQuery(ctx context.Context, db Execer, qry string) ([]QueryColumn, error) {
	c, err := getConn(ctx, db)
	if err != nil {
		return nil, err
	}
	defer c.close(false)

	stmt, err := c.PrepareContext(ctx, qry)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	st := stmt.(*statement)
	describeOnly(&st.stmtOptions)
	dR, err := st.QueryContext(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer dR.Close()
	r := dR.(*rows)
	cols := make([]QueryColumn, len(r.columns))
	for i, col := range r.columns {
		cols[i] = QueryColumn{
			Name:      col.Name,
			Type:      int(col.OracleType),
			Length:    int(col.Size),
			Precision: int(col.Precision),
			Scale:     int(col.Scale),
			Nullable:  col.Nullable,
		}
	}
	return cols, nil
}

// CompileError represents a compile-time error as in user_errors view.
type CompileError struct {
	Owner, Name, Type    string
	Line, Position, Code int64
	Text                 string
	Warning              bool
}

func (ce CompileError) Error() string {
	prefix := "ERROR "
	if ce.Warning {
		prefix = "WARN  "
	}
	return fmt.Sprintf("%s %s.%s %s %d:%d [%d] %s",
		prefix, ce.Owner, ce.Name, ce.Type, ce.Line, ce.Position, ce.Code, ce.Text)
}

type queryer interface {
	Query(string, ...interface{}) (*sql.Rows, error)
}

// GetCompileErrors returns the slice of the errors in user_errors.
//
// If all is false, only errors are returned; otherwise, warnings, too.
func GetCompileErrors(queryer queryer, all bool) ([]CompileError, error) {
	rows, err := queryer.Query(`
	SELECT USER owner, name, type, line, position, message_number, text, attribute
		FROM user_errors
		ORDER BY name, sequence`)
	if err != nil {
		return nil, err
	}
	var errors []CompileError
	var warn string
	for rows.Next() {
		var ce CompileError
		if err = rows.Scan(&ce.Owner, &ce.Name, &ce.Type, &ce.Line, &ce.Position, &ce.Code, &ce.Text, &warn); err != nil {
			return errors, err
		}
		ce.Warning = warn == "WARNING"
		if !ce.Warning || all {
			errors = append(errors, ce)
		}
	}
	return errors, rows.Err()
}

type preparer interface {
	PrepareContext(ctx context.Context, qry string) (*sql.Stmt, error)
}

// NamedToOrdered converts the query from named params (:paramname) to :%d placeholders + slice of params, copying the params verbatim.
func NamedToOrdered(qry string, namedParams map[string]interface{}) (string, []interface{}) {
	return MapToSlice(qry, func(k string) interface{} { return namedParams[k] })
}

// MapToSlice modifies query for map (:paramname) to :%d placeholders + slice of params.
//
// Calls metParam for each parameter met, and returns the slice of their results.
func MapToSlice(qry string, metParam func(string) interface{}) (string, []interface{}) {
	if metParam == nil {
		metParam = func(string) interface{} { return nil }
	}
	arr := make([]interface{}, 0, 16)
	var buf bytes.Buffer
	state, p, last := 0, 0, 0
	var prev rune

	Add := func(i int) {
		state = 0
		if i-p <= 1 { // :=
			return
		}
		arr = append(arr, metParam(qry[p+1:i]))
		param := fmt.Sprintf(":%d", len(arr))
		buf.WriteString(qry[last:p])
		buf.WriteString(param)
		last = i
	}

	for i, r := range qry {
		switch state {
		case 2:
			if r == '\n' {
				state = 0
			}
		case 3:
			if prev == '*' && r == '/' {
				state = 0
			}
		case 0:
			switch r {
			case '-':
				if prev == '-' {
					state = 2
				}
			case '*':
				if prev == '/' {
					state = 3
				}
			case ':':
				state = 1
				p = i
				// An identifier consists of a letter optionally followed by more letters, numerals, dollar signs, underscores, and number signs.
				// http://docs.oracle.com/cd/B19306_01/appdev.102/b14261/fundamentals.htm#sthref309
			}
		case 1:
			if !('A' <= r && r <= 'Z' || 'a' <= r && r <= 'z' ||
				(i-p > 1 && ('0' <= r && r <= '9' || r == '$' || r == '_' || r == '#'))) {

				Add(i)
			}
		}
		prev = r
	}
	if state == 1 {
		Add(len(qry))
	}
	if last <= len(qry)-1 {
		buf.WriteString(qry[last:])
	}
	return buf.String(), arr
}

// EnableDbmsOutput enables DBMS_OUTPUT buffering on the given connection.
// This is required if you want to retrieve the output with ReadDbmsOutput later.
func EnableDbmsOutput(ctx context.Context, conn Execer) error {
	qry := "BEGIN DBMS_OUTPUT.enable(1000000); END;"
	_, err := conn.ExecContext(ctx, qry)
	if err != nil {
		return errors.Errorf("%s: %w", qry, err)
	}
	return nil
}

// ReadDbmsOutput copies the DBMS_OUTPUT buffer into the given io.Writer.
func ReadDbmsOutput(ctx context.Context, w io.Writer, conn preparer) error {
	const maxNumLines = 128
	bw := bufio.NewWriterSize(w, maxNumLines*(32<<10))

	const qry = `BEGIN DBMS_OUTPUT.get_lines(:1, :2); END;`
	stmt, err := conn.PrepareContext(ctx, qry)
	if err != nil {
		return errors.Errorf("%s: %w", qry, err)
	}

	lines := make([]string, maxNumLines)
	var numLines int64
	params := []interface{}{
		PlSQLArrays,
		sql.Out{Dest: &lines}, sql.Out{Dest: &numLines, In: true},
	}
	for {
		numLines = int64(len(lines))
		if _, err = stmt.ExecContext(ctx, params...); err != nil {
			_ = bw.Flush()
			return errors.Errorf("%s: %w", qry, err)
		}
		for i := 0; i < int(numLines); i++ {
			_, _ = bw.WriteString(lines[i])
			if err = bw.WriteByte('\n'); err != nil {
				_ = bw.Flush()
				return err
			}
		}
		if int(numLines) < len(lines) {
			return bw.Flush()
		}
	}
}

// ClientVersion returns the VersionInfo from the DB.
func ClientVersion(ctx context.Context, ex Execer) (VersionInfo, error) {
	c, err := getConn(ctx, ex)
	if err != nil {
		return VersionInfo{}, err
	}
	return c.drv.ClientVersion()
}

// ServerVersion returns the VersionInfo of the client.
func ServerVersion(ctx context.Context, ex Execer) (VersionInfo, error) {
	c, err := getConn(ctx, ex)
	if err != nil {
		return VersionInfo{}, err
	}
	return c.Server, nil
}

// Conn is the interface for a connection, to be returned by DriverConn.
type Conn interface {
	driver.Conn
	driver.ConnBeginTx
	driver.ConnPrepareContext
	driver.Pinger

	Break() error
	Commit() error
	Rollback() error
	ClientVersion() (VersionInfo, error)
	ServerVersion() (VersionInfo, error)
	GetObjectType(name string) (ObjectType, error)
	NewSubscription(string, func(Event)) (*Subscription, error)
	Startup(StartupMode) error
	Shutdown(ShutdownMode) error
	NewData(baseType interface{}, SliceLen, BufSize int) ([]*Data, error)

	Timezone() *time.Location
}

// DriverConn returns the *godror.conn of the database/sql.Conn
func DriverConn(ctx context.Context, ex Execer) (Conn, error) {
	return getConn(ctx, ex)
}

var getConnMu sync.Mutex

func getConn(ctx context.Context, ex Execer) (*conn, error) {
	getConnMu.Lock()
	defer getConnMu.Unlock()
	var c interface{}
	if _, err := ex.ExecContext(ctx, getConnection, sql.Out{Dest: &c}); err != nil {
		return nil, errors.Errorf("getConnection: %w", err)
	}
	return c.(*conn), nil
}

// WrapRows transforms a driver.Rows into an *sql.Rows.
func WrapRows(ctx context.Context, q Querier, rset driver.Rows) (*sql.Rows, error) {
	return q.QueryContext(ctx, wrapResultset, rset)
}

func Timezone(ctx context.Context, ex Execer) (*time.Location, error) {
	c, err := getConn(ctx, ex)
	if err != nil {
		return nil, err
	}
	return c.Timezone(), nil
}
