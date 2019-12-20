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

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"sync"

	"github.com/pkg/errors"
)

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
	c, err := getConn(db)
	if err != nil {
		return nil, err
	}

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
	return errors.Wrap(err, qry)
}

// ReadDbmsOutput copies the DBMS_OUTPUT buffer into the given io.Writer.
func ReadDbmsOutput(ctx context.Context, w io.Writer, conn preparer) error {
	const maxNumLines = 128
	bw := bufio.NewWriterSize(w, maxNumLines*(32<<10))

	const qry = `BEGIN DBMS_OUTPUT.get_lines(:1, :2); END;`
	stmt, err := conn.PrepareContext(ctx, qry)
	if err != nil {
		return errors.Wrap(err, qry)
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
			return errors.Wrap(err, qry)
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
func ClientVersion(ex Execer) (VersionInfo, error) {
	c, err := getConn(ex)
	if err != nil {
		return VersionInfo{}, err
	}
	return c.drv.ClientVersion()
}

// ServerVersion returns the VersionInfo of the client.
func ServerVersion(ex Execer) (VersionInfo, error) {
	c, err := getConn(ex)
	if err != nil {
		return VersionInfo{}, err
	}
	return c.Server, nil
}

// Conn is the interface for a connection, to be returned by DriverConn.
type Conn interface {
	driver.Conn
	driver.Pinger
	Break() error
	BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error)
	PrepareContext(ctx context.Context, query string) (driver.Stmt, error)
	Commit() error
	Rollback() error
	ServerVersion() (VersionInfo, error)
	GetObjectType(name string) (ObjectType, error)
	NewSubscription(string, func(Event)) (*Subscription, error)
	Startup(StartupMode) error
	Shutdown(ShutdownMode) error
}

// DriverConn returns the *goracle.conn of the database/sql.Conn
func DriverConn(ex Execer) (Conn, error) {
	return getConn(ex)
}

var getConnMu sync.Mutex

func getConn(ex Execer) (*conn, error) {
	getConnMu.Lock()
	defer getConnMu.Unlock()
	var c interface{}
	if _, err := ex.ExecContext(context.Background(), getConnection, sql.Out{Dest: &c}); err != nil {
		return nil, errors.Wrap(err, "getConnection")
	}
	return c.(*conn), nil
}

// WrapRows transforms a driver.Rows into an *sql.Rows.
func WrapRows(ctx context.Context, q Querier, rset driver.Rows) (*sql.Rows, error) {
	return q.QueryContext(ctx, wrapResultset, rset)
}
