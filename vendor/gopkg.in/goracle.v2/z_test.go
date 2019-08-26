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

package goracle_test

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"math/rand"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	goracle "gopkg.in/goracle.v2"
)

var (
	testDb *sql.DB
	tl     = &testLogger{}

	clientVersion, serverVersion goracle.VersionInfo
	testConStr                   string
)

var tblSuffix = "_" + strings.Replace(runtime.Version(), ".", "#", -1)

const maxSessions = 64

func init() {
	logger := &log.SwapLogger{}
	goracle.Log = logger.Log
	if os.Getenv("VERBOSE") == "1" {
		logger.Swap(tl)
	}

	P := goracle.ConnectionParams{
		Username:    os.Getenv("GORACLE_DRV_TEST_USERNAME"),
		Password:    os.Getenv("GORACLE_DRV_TEST_PASSWORD"),
		SID:         os.Getenv("GORACLE_DRV_TEST_DB"),
		MinSessions: 1, MaxSessions: maxSessions, PoolIncrement: 1,
		ConnClass:    "POOLED",
		EnableEvents: true,
	}
	if strings.HasSuffix(strings.ToUpper(P.Username), " AS SYSDBA") {
		P.IsSysDBA, P.Username = true, P.Username[:len(P.Username)-10]
	}
	testConStr = P.StringWithPassword()
	var err error
	if testDb, err = sql.Open("goracle", testConStr); err != nil {
		fmt.Printf("ERROR: %+v\n", err)
		return
		//panic(err)
	}

	if testDb != nil {
		if clientVersion, err = goracle.ClientVersion(testDb); err != nil {
			fmt.Printf("ERROR: %+v\n", err)
			return
		}
		if serverVersion, err = goracle.ServerVersion(testDb); err != nil {
			fmt.Printf("ERROR: %+v\n", err)
			return
		}
		fmt.Println("Server:", serverVersion)
		fmt.Println("Client:", clientVersion)
	}
}

var bufPool = sync.Pool{New: func() interface{} { return bytes.NewBuffer(make([]byte, 0, 1024)) }}

type testLogger struct {
	sync.RWMutex
	Ts       []*testing.T
	beHelped []*testing.T
}

func (tl *testLogger) Log(args ...interface{}) error {
	fmt.Println(args...)
	return tl.GetLog()(args)
}
func (tl *testLogger) GetLog() func(keyvals ...interface{}) error {
	return func(keyvals ...interface{}) error {
		buf := bufPool.Get().(*bytes.Buffer)
		defer bufPool.Put(buf)
		buf.Reset()
		if len(keyvals)%2 != 0 {
			keyvals = append(append(make([]interface{}, 0, len(keyvals)+1), "msg"), keyvals...)
		}
		for i := 0; i < len(keyvals); i += 2 {
			fmt.Fprintf(buf, "%s=%#v ", keyvals[i], keyvals[i+1])
		}

		tl.Lock()
		for _, t := range tl.beHelped {
			t.Helper()
		}
		tl.beHelped = tl.beHelped[:0]
		tl.Unlock()

		tl.RLock()
		defer tl.RUnlock()
		for _, t := range tl.Ts {
			t.Helper()
			t.Log(buf.String())
		}

		return nil
	}
}
func (tl *testLogger) enableLogging(t *testing.T) func() {
	tl.Lock()
	tl.Ts = append(tl.Ts, t)
	tl.beHelped = append(tl.beHelped, t)
	tl.Unlock()

	return func() {
		tl.Lock()
		defer tl.Unlock()
		for i, f := range tl.Ts {
			if f == t {
				tl.Ts[i] = tl.Ts[0]
				tl.Ts = tl.Ts[1:]
				break
			}
		}
		for i, f := range tl.beHelped {
			if f == t {
				tl.beHelped[i] = tl.beHelped[0]
				tl.beHelped = tl.beHelped[1:]
				break
			}
		}
	}
}

func TestDescribeQuery(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	const qry = "SELECT * FROM user_tab_cols"
	cols, err := goracle.DescribeQuery(ctx, testDb, qry)
	if err != nil {
		t.Fatal(errors.Wrap(err, qry))
	}
	t.Log(cols)
}

func TestParseOnly(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tbl := "test_not_exist" + tblSuffix
	cnt := func() int {
		var cnt int64
		if err := testDb.QueryRowContext(ctx,
			"SELECT COUNT(0) FROM user_tables WHERE table_name = UPPER('"+tbl+"')").Scan(&cnt); //nolint:gas
		err != nil {
			t.Fatal(err)
		}
		return int(cnt)
	}

	if cnt() != 0 {
		if _, err := testDb.ExecContext(ctx, "DROP TABLE "+tbl); err != nil {
			t.Error(err)
		}
	}
	if _, err := testDb.ExecContext(ctx, "CREATE TABLE "+tbl+"(t VARCHAR2(1))", goracle.ParseOnly()); err != nil {
		t.Fatal(err)
	}
	if got := cnt(); got != 1 {
		t.Errorf("got %d, wanted 0", got)
	}
}

func TestInputArray(t *testing.T) {
	t.Parallel()
	defer tl.enableLogging(t)()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pkg := strings.ToUpper("test_in_pkg" + tblSuffix)
	qry := `CREATE OR REPLACE PACKAGE ` + pkg + ` AS
TYPE int_tab_typ IS TABLE OF BINARY_INTEGER INDEX BY PLS_INTEGER;
TYPE num_tab_typ IS TABLE OF NUMBER INDEX BY PLS_INTEGER;
TYPE vc_tab_typ IS TABLE OF VARCHAR2(100) INDEX BY PLS_INTEGER;
TYPE dt_tab_typ IS TABLE OF DATE INDEX BY PLS_INTEGER;
--TYPE lob_tab_typ IS TABLE OF CLOB INDEX BY PLS_INTEGER;

FUNCTION in_int(p_int IN int_tab_typ) RETURN VARCHAR2;
FUNCTION in_num(p_num IN num_tab_typ) RETURN VARCHAR2;
FUNCTION in_vc(p_vc IN vc_tab_typ) RETURN VARCHAR2;
FUNCTION in_dt(p_dt IN dt_tab_typ) RETURN VARCHAR2;
END;
`
	if _, err := testDb.ExecContext(ctx, qry); err != nil {
		t.Fatal(err, qry)
	}
	defer testDb.Exec("DROP PACKAGE " + pkg)

	qry = `CREATE OR REPLACE PACKAGE BODY ` + pkg + ` AS
FUNCTION in_int(p_int IN int_tab_typ) RETURN VARCHAR2 IS
  v_idx PLS_INTEGER;
  v_res VARCHAR2(32767);
BEGIN
  v_idx := p_int.FIRST;
  WHILE v_idx IS NOT NULL LOOP
    v_res := v_res||v_idx||':'||p_int(v_idx)||CHR(10);
    v_idx := p_int.NEXT(v_idx);
  END LOOP;
  RETURN(v_res);
END;

FUNCTION in_num(p_num IN num_tab_typ) RETURN VARCHAR2 IS
  v_idx PLS_INTEGER;
  v_res VARCHAR2(32767);
BEGIN
  v_idx := p_num.FIRST;
  WHILE v_idx IS NOT NULL LOOP
    v_res := v_res||v_idx||':'||p_num(v_idx)||CHR(10);
    v_idx := p_num.NEXT(v_idx);
  END LOOP;
  RETURN(v_res);
END;

FUNCTION in_vc(p_vc IN vc_tab_typ) RETURN VARCHAR2 IS
  v_idx PLS_INTEGER;
  v_res VARCHAR2(32767);
BEGIN
  v_idx := p_vc.FIRST;
  WHILE v_idx IS NOT NULL LOOP
    v_res := v_res||v_idx||':'||p_vc(v_idx)||CHR(10);
    v_idx := p_vc.NEXT(v_idx);
  END LOOP;
  RETURN(v_res);
END;
FUNCTION in_dt(p_dt IN dt_tab_typ) RETURN VARCHAR2 IS
  v_idx PLS_INTEGER;
  v_res VARCHAR2(32767);
BEGIN
  v_idx := p_dt.FIRST;
  WHILE v_idx IS NOT NULL LOOP
    v_res := v_res||v_idx||':'||TO_CHAR(p_dt(v_idx), 'YYYY-MM-DD"T"HH24:MI:SS')||CHR(10);
    v_idx := p_dt.NEXT(v_idx);
  END LOOP;
  RETURN(v_res);
END;
END;
`
	if _, err := testDb.ExecContext(ctx, qry); err != nil {
		t.Fatal(err, qry)
	}
	compileErrors, err := goracle.GetCompileErrors(testDb, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(compileErrors) != 0 {
		t.Logf("compile errors: %v", compileErrors)
		for _, ce := range compileErrors {
			if strings.Contains(ce.Error(), pkg) {
				t.Fatal(ce)
			}
		}
	}

	epoch := time.Date(2017, 11, 20, 12, 14, 21, 0, time.Local)
	for name, tC := range map[string]struct {
		In   interface{}
		Want string
	}{
		//"int_0":{In:[]int32{}, Want:""},
		"num_0": {In: []goracle.Number{}, Want: ""},
		"vc_0":  {In: []string{}, Want: ""},
		"dt_0":  {In: []time.Time{}, Want: ""},

		"num_3": {
			In:   []goracle.Number{"1", "2.72", "-3.14"},
			Want: "1:1\n2:2.72\n3:-3.14\n",
		},
		"vc_3": {
			In:   []string{"a", "", "cCc"},
			Want: "1:a\n2:\n3:cCc\n",
		},
		"dt_3": {
			In:   []time.Time{epoch, epoch.AddDate(0, 0, -1), epoch.AddDate(0, 0, -2)},
			Want: "1:2017-11-20T12:14:21\n2:2017-11-19T12:14:21\n3:2017-11-18T12:14:21\n",
		},
	} {
		typ := strings.SplitN(name, "_", 2)[0]
		qry := "BEGIN :1 := " + pkg + ".in_" + typ + "(:2); END;"
		var res string
		if _, err := testDb.ExecContext(ctx, qry, goracle.PlSQLArrays,
			sql.Out{Dest: &res}, tC.In,
		); err != nil {
			t.Error(errors.Wrapf(err, "%q. %s %+v", name, qry, tC.In))
		}
		t.Logf("%q. %q", name, res)
		if typ == "num" {
			res = strings.Replace(res, ",", ".", -1)
		}
		if res != tC.Want {
			t.Errorf("%q. got %q, wanted %q.", name, res, tC.Want)
		}
	}
}

func TestDbmsOutput(t *testing.T) {
	defer tl.enableLogging(t)()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := testDb.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := goracle.EnableDbmsOutput(ctx, conn); err != nil {
		t.Fatal(err)
	}

	txt := `árvíztűrő tükörfúrógép`
	qry := "BEGIN DBMS_OUTPUT.PUT_LINE('" + txt + "'); END;"
	if _, err := conn.ExecContext(ctx, qry); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := goracle.ReadDbmsOutput(ctx, &buf, conn); err != nil {
		t.Error(err)
	}
	t.Log(buf.String())
	if buf.String() != txt+"\n" {
		t.Errorf("got %q, wanted %q", buf.String(), txt+"\n")
	}
}

func TestInOutArray(t *testing.T) {
	t.Parallel()
	defer tl.enableLogging(t)()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	conn, err := testDb.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	pkg := strings.ToUpper("test_pkg" + tblSuffix)
	qry := `CREATE OR REPLACE PACKAGE ` + pkg + ` AS
TYPE int_tab_typ IS TABLE OF BINARY_INTEGER INDEX BY PLS_INTEGER;
TYPE num_tab_typ IS TABLE OF NUMBER INDEX BY PLS_INTEGER;
TYPE vc_tab_typ IS TABLE OF VARCHAR2(100) INDEX BY PLS_INTEGER;
TYPE dt_tab_typ IS TABLE OF DATE INDEX BY PLS_INTEGER;
TYPE lob_tab_typ IS TABLE OF CLOB INDEX BY PLS_INTEGER;

PROCEDURE inout_int(p_int IN OUT int_tab_typ);
PROCEDURE inout_num(p_num IN OUT num_tab_typ);
PROCEDURE inout_vc(p_vc IN OUT vc_tab_typ);
PROCEDURE inout_dt(p_dt IN OUT dt_tab_typ);
PROCEDURE p2(
	--p_int IN OUT int_tab_typ,
	p_num IN OUT num_tab_typ, p_vc IN OUT vc_tab_typ, p_dt IN OUT dt_tab_typ);
END;
`
	if _, err = conn.ExecContext(ctx, qry); err != nil {
		t.Fatal(err, qry)
	}
	defer testDb.Exec("DROP PACKAGE " + pkg)

	qry = `CREATE OR REPLACE PACKAGE BODY ` + pkg + ` AS
PROCEDURE inout_int(p_int IN OUT int_tab_typ) IS
  v_idx PLS_INTEGER;
BEGIN
  DBMS_OUTPUT.PUT_LINE('p_int.COUNT='||p_int.COUNT||' FIRST='||p_int.FIRST||' LAST='||p_int.LAST);
  v_idx := p_int.FIRST;
  WHILE v_idx IS NOT NULL LOOP
    p_int(v_idx) := NVL(p_int(v_idx) * 2, 1);
	v_idx := p_int.NEXT(v_idx);
  END LOOP;
  p_int(NVL(p_int.LAST, 0)+1) := p_int.COUNT;
END;

PROCEDURE inout_num(p_num IN OUT num_tab_typ) IS
  v_idx PLS_INTEGER;
BEGIN
  DBMS_OUTPUT.PUT_LINE('p_num.COUNT='||p_num.COUNT||' FIRST='||p_num.FIRST||' LAST='||p_num.LAST);
  v_idx := p_num.FIRST;
  WHILE v_idx IS NOT NULL LOOP
    p_num(v_idx) := NVL(p_num(v_idx) / 2, 0.5);
	v_idx := p_num.NEXT(v_idx);
  END LOOP;
  p_num(NVL(p_num.LAST, 0)+1) := p_num.COUNT;
END;

PROCEDURE inout_vc(p_vc IN OUT vc_tab_typ) IS
  v_idx PLS_INTEGER;
BEGIN
  DBMS_OUTPUT.PUT_LINE('p_vc.COUNT='||p_vc.COUNT||' FIRST='||p_vc.FIRST||' LAST='||p_vc.LAST);
  v_idx := p_vc.FIRST;
  WHILE v_idx IS NOT NULL LOOP
    p_vc(v_idx) := NVL(p_vc(v_idx) ||' +', '-');
	v_idx := p_vc.NEXT(v_idx);
  END LOOP;
  p_vc(NVL(p_vc.LAST, 0)+1) := p_vc.COUNT;
END;

PROCEDURE inout_dt(p_dt IN OUT dt_tab_typ) IS
  v_idx PLS_INTEGER;
BEGIN
  DBMS_OUTPUT.PUT_LINE('p_dt.COUNT='||p_dt.COUNT||' FIRST='||p_dt.FIRST||' LAST='||p_dt.LAST);
  v_idx := p_dt.FIRST;
  WHILE v_idx IS NOT NULL LOOP
  DBMS_OUTPUT.PUT_LINE(v_idx||'='||TO_CHAR(p_dt(v_idx), 'YYYY-MM-DD HH24:MI:SS'));
    p_dt(v_idx) := NVL(p_dt(v_idx) + 1, TRUNC(SYSDATE)-v_idx);
	v_idx := p_dt.NEXT(v_idx);
  END LOOP;
  p_dt(NVL(p_dt.LAST, 0)+1) := TRUNC(SYSDATE);
END;

PROCEDURE p2(
	--p_int IN OUT int_tab_typ,
	p_num IN OUT num_tab_typ,
	p_vc IN OUT vc_tab_typ,
	p_dt IN OUT dt_tab_typ
--, p_lob IN OUT lob_tab_typ
) IS
BEGIN
  --inout_int(p_int);
  inout_num(p_num);
  inout_vc(p_vc);
  inout_dt(p_dt);
  --p_lob := NULL;
END p2;
END;
`
	if _, err = testDb.ExecContext(ctx, qry); err != nil {
		t.Fatal(err, qry)
	}
	compileErrors, err := goracle.GetCompileErrors(testDb, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(compileErrors) != 0 {
		t.Logf("compile errors: %v", compileErrors)
		for _, ce := range compileErrors {
			if strings.Contains(ce.Error(), pkg) {
				t.Fatal(ce)
			}
		}
	}

	intgr := []int32{3, 1, 4, 0, 0}[:3]
	intgrWant := []int32{3 * 2, 1 * 2, 4 * 2, 3}
	_ = intgrWant
	num := []goracle.Number{"3.14", "-2.48", ""}[:2]
	numWant := []goracle.Number{"1.57", "-1.24", "2"}
	vc := []string{"string", "bring", ""}[:2]
	vcWant := []string{"string +", "bring +", "2"}
	dt := []time.Time{time.Date(2017, 6, 18, 7, 5, 51, 0, time.Local), {}, {}}[:2]
	today := time.Now().Truncate(24 * time.Hour)
	today = time.Date(today.Year(), today.Month(), today.Day(), today.Hour(), today.Minute(), today.Second(), 0, time.Local)
	dtWant := []time.Time{
		dt[0].Add(24 * time.Hour),
		today.Add(-2 * 24 * time.Hour),
		today,
	}

	goracle.EnableDbmsOutput(ctx, testDb)

	opts := []cmp.Option{
		cmp.Comparer(func(x, y time.Time) bool {
			d := x.Sub(y)
			if d < 0 {
				d *= -1
			}
			return d <= 2*time.Hour
		}),
	}

	for _, tC := range []struct {
		Name     string
		In, Want interface{}
	}{
		{Name: "vc", In: vc, Want: vcWant},
		{Name: "num", In: num, Want: numWant},
		{Name: "dt", In: dt, Want: dtWant},
		//{Name: "int", In: intgr, Want: intgrWant},
		{Name: "vc-1", In: vc[:1], Want: []string{"string +", "1"}},
		{Name: "vc-0", In: vc[:0], Want: []string{"0"}},
	} {
		tC := tC
		t.Run("inout_"+tC.Name, func(t *testing.T) {
			t.Logf("%s=%s", tC.Name, tC.In)
			nm := strings.SplitN(tC.Name, "-", 2)[0]
			qry = "BEGIN " + pkg + ".inout_" + nm + "(:1); END;"
			dst := copySlice(tC.In)
			if _, err := testDb.ExecContext(ctx, qry,
				goracle.PlSQLArrays,
				sql.Out{Dest: dst, In: true},
			); err != nil {
				t.Fatalf("%s\n%+v", qry, err)
			}

			got := reflect.ValueOf(dst).Elem().Interface()
			if cmp.Equal(got, tC.Want, opts...) {
				return
			}
			t.Errorf("%s: %s", tC.Name, cmp.Diff(got, tC.Want))
			var buf bytes.Buffer
			if err := goracle.ReadDbmsOutput(ctx, &buf, testDb); err != nil {
				t.Error(err)
			}
			t.Log("OUTPUT:", buf.String())
		})
	}

	//lob := []goracle.Lob{goracle.Lob{IsClob: true, Reader: strings.NewReader("abcdef")}}
	t.Run("p2", func(t *testing.T) {
		if _, err := testDb.ExecContext(ctx,
			"BEGIN "+pkg+".p2(:1, :2, :3); END;",
			goracle.PlSQLArrays,
			//sql.Out{Dest: &intgr, In: true},
			sql.Out{Dest: &num, In: true},
			sql.Out{Dest: &vc, In: true},
			sql.Out{Dest: &dt, In: true},
			//sql.Out{Dest: &lob, In: true},
		); err != nil {
			t.Fatal(err)
		}
		t.Logf("int=%#v num=%#v vc=%#v dt=%#v", intgr, num, vc, dt)
		//if d := cmp.Diff(intgr, intgrWant); d != "" {
		//	t.Errorf("int: %s", d)
		//}
		if d := cmp.Diff(num, numWant); d != "" {
			t.Errorf("num: %s", d)
		}
		if d := cmp.Diff(vc, vcWant); d != "" {
			t.Errorf("vc: %s", d)
		}
		if !cmp.Equal(dt, dtWant, opts...) {
			if d := cmp.Diff(dt, dtWant); d != "" {
				t.Errorf("dt: %s", d)
			}
		}
	})
}

func TestOutParam(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := testDb.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	pkg := strings.ToUpper("test_p1" + tblSuffix)
	qry := `CREATE OR REPLACE PROCEDURE
` + pkg + `(p_int IN OUT INTEGER, p_num IN OUT NUMBER, p_vc IN OUT VARCHAR2, p_dt IN OUT DATE, p_lob IN OUT CLOB)
IS
BEGIN
  p_int := NVL(p_int * 2, 1);
  p_num := NVL(p_num / 2, 0.5);
  p_vc := NVL(p_vc ||' +', '-');
  p_dt := NVL(p_dt + 1, SYSDATE);
  p_lob := NULL;
END;`
	if _, err = testDb.ExecContext(ctx, qry); err != nil {
		t.Fatal(err, qry)
	}
	defer testDb.Exec("DROP PROCEDURE " + pkg)

	qry = "BEGIN " + pkg + "(:1, :2, :3, :4, :5); END;"
	stmt, err := testDb.PrepareContext(ctx, qry)
	if err != nil {
		t.Fatal(errors.Wrap(err, qry))
	}
	defer stmt.Close()

	var intgr int = 3
	num := goracle.Number("3.14")
	var vc string = "string"
	var dt time.Time = time.Date(2017, 6, 18, 7, 5, 51, 0, time.Local)
	var lob goracle.Lob = goracle.Lob{IsClob: true, Reader: strings.NewReader("abcdef")}
	if _, err := stmt.ExecContext(ctx,
		sql.Out{Dest: &intgr, In: true},
		sql.Out{Dest: &num, In: true},
		sql.Out{Dest: &vc, In: true},
		sql.Out{Dest: &dt, In: true},
		sql.Out{Dest: &lob, In: true},
	); err != nil {
		t.Fatal(errors.Wrap(err, qry))
	}
	t.Logf("int=%#v num=%#v vc=%#v dt=%#v", intgr, num, vc, dt)
	if intgr != 6 {
		t.Errorf("int: got %d, wanted %d", intgr, 6)
	}
	if num != "1.57" {
		t.Errorf("num: got %q, wanted %q", num, "1.57")
	}
	if vc != "string +" {
		t.Errorf("vc: got %q, wanted %q", vc, "string +")
	}
}

func TestSelectRefCursor(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	rows, err := testDb.QueryContext(ctx, "SELECT CURSOR(SELECT object_name, object_type, object_id, created FROM all_objects WHERE ROWNUM <= 10) FROM DUAL")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var intf interface{}
		if err := rows.Scan(&intf); err != nil {
			t.Error(err)
			continue
		}
		t.Logf("%T", intf)
		sub := intf.(driver.RowsColumnTypeScanType)
		cols := sub.Columns()
		t.Log("Columns", cols)
		dests := make([]driver.Value, len(cols))
		for {
			if err := sub.Next(dests); err != nil {
				if err == io.EOF {
					break
				}
				t.Error(err)
				break
			}
			//fmt.Println(dests)
			t.Log(dests)
		}
		sub.Close()
	}
}

func TestSelectRefCursorWrap(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	rows, err := testDb.QueryContext(ctx, "SELECT CURSOR(SELECT object_name, object_type, object_id, created FROM all_objects WHERE ROWNUM <= 10) FROM DUAL")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var intf interface{}
		if err := rows.Scan(&intf); err != nil {
			t.Error(err)
			continue
		}
		t.Logf("%T", intf)
		sub, err := goracle.WrapRows(ctx, testDb, intf.(driver.Rows))
		if err != nil {
			t.Fatal(err)
		}
		t.Log("Sub", sub)
		for sub.Next() {
			var oName, oType, oID string
			var created time.Time
			if err := sub.Scan(&oName, &oType, &oID, &created); err != nil {
				t.Error(err)
				break
			}
			t.Log(oName, oType, oID, created)
		}
		sub.Close()
	}
}

func TestExecuteMany(t *testing.T) {
	t.Parallel()
	defer tl.enableLogging(t)()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := testDb.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	tbl := "test_em" + tblSuffix
	conn.ExecContext(ctx, "DROP TABLE "+tbl)
	conn.ExecContext(ctx, "CREATE TABLE "+tbl+" (f_id INTEGER, f_int INTEGER, f_num NUMBER, f_num_6 NUMBER(6), F_num_5_2 NUMBER(5,2), f_vc VARCHAR2(30), F_dt DATE)")
	defer testDb.Exec("DROP TABLE " + tbl)

	const num = 1000
	ints := make([]int, num)
	nums := make([]goracle.Number, num)
	int32s := make([]int32, num)
	floats := make([]float64, num)
	strs := make([]string, num)
	dates := make([]time.Time, num)
	// This is instead of now: a nice moment in time right before the summer time shift
	now := time.Date(2017, 10, 29, 1, 27, 53, 0, time.Local).Truncate(time.Second)
	ids := make([]int, num)
	for i := range nums {
		ids[i] = i
		ints[i] = i << 1
		nums[i] = goracle.Number(strconv.Itoa(i))
		int32s[i] = int32(i)
		floats[i] = float64(i) / float64(3.14)
		strs[i] = fmt.Sprintf("%x", i)
		dates[i] = now.Add(-time.Duration(i) * time.Hour)
	}
	for i, tc := range []struct {
		Name  string
		Value interface{}
	}{
		{"f_int", ints},
		{"f_num", nums},
		{"f_num_6", int32s},
		{"f_num_5_2", floats},
		{"f_vc", strs},
		{"f_dt", dates},
	} {
		res, execErr := conn.ExecContext(ctx,
			"INSERT INTO "+tbl+" ("+tc.Name+") VALUES (:1)", //nolint:gas
			tc.Value)
		if execErr != nil {
			t.Fatalf("%d. INSERT INTO "+tbl+" (%q) VALUES (%+v): %#v", //nolint:gas
				i, tc.Name, tc.Value, execErr)
		}
		ra, raErr := res.RowsAffected()
		if raErr != nil {
			t.Error(raErr)
		} else if ra != num {
			t.Errorf("%d. %q: wanted %d rows, got %d", i, tc.Name, num, ra)
		}
	}

	conn.ExecContext(ctx, "TRUNCATE TABLE "+tbl+"")

	res, err := conn.ExecContext(ctx,
		`INSERT INTO `+tbl+ //nolint:gas
			` (f_id, f_int, f_num, f_num_6, F_num_5_2, F_vc, F_dt)
			VALUES
			(:1, :2, :3, :4, :5, :6, :7)`,
		ids, ints, nums, int32s, floats, strs, dates)
	if err != nil {
		t.Fatalf("%#v", err)
	}
	ra, err := res.RowsAffected()
	if err != nil {
		t.Error(err)
	} else if ra != num {
		t.Errorf("wanted %d rows, got %d", num, ra)
	}

	rows, err := conn.QueryContext(ctx,
		"SELECT * FROM "+tbl+" ORDER BY F_id", //nolint:gas
	)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	i := 0
	for rows.Next() {
		var id, Int int
		var num string
		var vc string
		var num6 int32
		var num52 float64
		var dt time.Time
		if err := rows.Scan(&id, &Int, &num, &num6, &num52, &vc, &dt); err != nil {
			t.Fatal(err)
		}
		if id != i {
			t.Fatalf("ID got %d, wanted %d.", id, i)
		}
		if Int != ints[i] {
			t.Errorf("%d. INT got %d, wanted %d.", i, Int, ints[i])
		}
		if num != string(nums[i]) {
			t.Errorf("%d. NUM got %q, wanted %q.", i, num, nums[i])
		}
		if num6 != int32s[i] {
			t.Errorf("%d. NUM_6 got %v, wanted %v.", i, num6, int32s[i])
		}
		rounded := float64(int64(floats[i]/0.005+0.5)) * 0.005
		if math.Abs(num52-rounded) > 0.05 {
			t.Errorf("%d. NUM_5_2 got %v, wanted %v.", i, num52, rounded)
		}
		if vc != strs[i] {
			t.Errorf("%d. VC got %q, wanted %q.", i, vc, strs[i])
		}
		t.Logf("%d. dt=%v", i, dt)
		if dt != dates[i] {
			t.Errorf("%d. got DT %v, wanted %v.", i, dt, dates[i])
		}
		i++
	}
}
func TestReadWriteLob(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := testDb.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	tbl := "test_lob" + tblSuffix
	conn.ExecContext(ctx, "DROP TABLE "+tbl)
	conn.ExecContext(ctx,
		"CREATE TABLE "+tbl+" (f_id NUMBER(6), f_blob BLOB, f_clob CLOB)", //nolint:gas
	)
	defer testDb.Exec(
		"DROP TABLE " + tbl, //nolint:gas
	)

	stmt, err := conn.PrepareContext(ctx,
		"INSERT INTO "+tbl+" (F_id, f_blob, F_clob) VALUES (:1, :2, :3)", //nolint:gas
	)
	if err != nil {
		t.Fatal(err)
	}
	defer stmt.Close()

	for tN, tC := range []struct {
		Bytes  []byte
		String string
	}{
		{[]byte{0, 1, 2, 3, 4, 5}, "12345"},
	} {

		if _, err = stmt.Exec(tN*2, tC.Bytes, tC.String); err != nil {
			t.Errorf("%d/1. (%v, %q): %v", tN, tC.Bytes, tC.String, err)
			continue
		}
		if _, err = stmt.Exec(tN*2+1,
			goracle.Lob{Reader: bytes.NewReader(tC.Bytes)},
			goracle.Lob{Reader: strings.NewReader(tC.String), IsClob: true},
		); err != nil {
			t.Errorf("%d/2. (%v, %q): %v", tN, tC.Bytes, tC.String, err)
		}

		var rows *sql.Rows
		rows, err = conn.QueryContext(ctx,
			"SELECT F_id, F_blob, F_clob FROM "+tbl+" WHERE F_id IN (:1, :2)", //nolint:gas
			goracle.LobAsReader(),
			2*tN, 2*tN+1)
		if err != nil {
			t.Errorf("%d/3. %v", tN, err)
			continue
		}
		for rows.Next() {
			var id, blob, clob interface{}
			if err = rows.Scan(&id, &blob, &clob); err != nil {
				rows.Close()
				t.Errorf("%d/3. scan: %v", tN, err)
				continue
			}
			t.Logf("%d. blob=%+v clob=%+v", id, blob, clob)
			if clob, ok := clob.(*goracle.Lob); !ok {
				t.Errorf("%d. %T is not LOB", id, blob)
			} else {
				var got []byte
				got, err = ioutil.ReadAll(clob)
				if err != nil {
					t.Errorf("%d. %v", id, err)
				} else if got := string(got); got != tC.String {
					t.Errorf("%d. got %q for CLOB, wanted %q", id, got, tC.String)
				}
			}
			if blob, ok := blob.(*goracle.Lob); !ok {
				t.Errorf("%d. %T is not LOB", id, blob)
			} else {
				var got []byte
				got, err = ioutil.ReadAll(blob)
				if err != nil {
					t.Errorf("%d. %v", id, err)
				} else if !bytes.Equal(got, tC.Bytes) {
					t.Errorf("%d. got %v for BLOB, wanted %v", id, got, tC.Bytes)
				}
			}
		}
		rows.Close()
	}

	rows, err := conn.QueryContext(ctx,
		"SELECT F_clob FROM "+tbl+"", //nolint:gas
		goracle.ClobAsString())
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var s string
		if err = rows.Scan(&s); err != nil {
			t.Error(err)
		}
		t.Logf("clobAsString: %q", s)
	}

	qry := "SELECT CURSOR(SELECT f_id, f_clob FROM " + tbl + " WHERE ROWNUM <= 10) FROM DUAL"
	rows, err = testDb.QueryContext(ctx, qry, goracle.ClobAsString())
	if err != nil {
		t.Fatal(errors.Wrap(err, qry))
	}
	defer rows.Close()
	for rows.Next() {
		var intf interface{}
		if err := rows.Scan(&intf); err != nil {
			t.Error(err)
			continue
		}
		t.Logf("%T", intf)
		sub := intf.(driver.RowsColumnTypeScanType)
		cols := sub.Columns()
		t.Log("Columns", cols)
		dests := make([]driver.Value, len(cols))
		for {
			if err := sub.Next(dests); err != nil {
				if err == io.EOF {
					break
				}
				t.Error(err)
				break
			}
			//fmt.Println(dests)
			t.Log(dests)
		}
		sub.Close()
	}

}

func copySlice(orig interface{}) interface{} {
	ro := reflect.ValueOf(orig)
	rc := reflect.New(reflect.TypeOf(orig)).Elem() // *[]s
	rc.Set(reflect.MakeSlice(ro.Type(), ro.Len(), ro.Cap()))
	for i := 0; i < ro.Len(); i++ {
		rc.Index(i).Set(ro.Index(i))
	}
	return rc.Addr().Interface()
}

func TestObject(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := testDb.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	pkg := strings.ToUpper("test_pkg_obj" + tblSuffix)
	qry := `CREATE OR REPLACE PACKAGE ` + pkg + ` IS
  TYPE int_tab_typ IS TABLE OF PLS_INTEGER INDEX BY PLS_INTEGER;
  TYPE rec_typ IS RECORD (int PLS_INTEGER, num NUMBER, vc VARCHAR2(1000), c CHAR(1000), dt DATE);
  TYPE tab_typ IS TABLE OF rec_typ INDEX BY PLS_INTEGER;
END;`
	if _, err = conn.ExecContext(ctx, qry); err != nil {
		t.Fatal(errors.Wrap(err, qry))
	}
	defer testDb.Exec("DROP PACKAGE " + pkg)

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()

	defer tl.enableLogging(t)()
	ot, err := goracle.GetObjectType(tx, pkg+".int_tab_typ")
	if err != nil {
		if clientVersion.Version >= 12 && serverVersion.Version >= 12 {
			t.Fatal(fmt.Sprintf("%+v", err))
		}
		t.Log(err)
		t.Skip("client or server version < 12")
	}
	t.Log(ot)
}

func TestOpenClose(t *testing.T) {
	t.Parallel()
	cs, err := goracle.ParseConnString(testConStr)
	if err != nil {
		t.Fatal(err)
	}
	cs.MinSessions, cs.MaxSessions = 1, 5
	t.Log(cs.String())
	db, err := sql.Open("goracle", cs.StringWithPassword())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxIdleConns(1)
	db.SetMaxOpenConns(3)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	const module = "goracle.v2.test-OpenClose "
	stmt, err := db.PrepareContext(ctx, "SELECT COUNT(0) FROM v$session WHERE module LIKE '"+module+"%'")
	if err != nil {
		if strings.Contains(err.Error(), "ORA-12516:") {
			t.Skip(err)
		}
		t.Fatal(err)
	}
	defer stmt.Close()
	sessCount := func() (int, error) {
		var n int
		qErr := stmt.QueryRowContext(ctx).Scan(&n)
		return n, qErr
	}
	n, err := sessCount()
	if err != nil {
		t.Skip(err)
	}
	if n > 0 {
		t.Logf("sessCount=%d at start!", n)
	}
	var tt goracle.TraceTag
	for i := 0; i < 10; i++ {
		tt.Module = fmt.Sprintf("%s%d", module, 2*i)
		ctx = goracle.ContextWithTraceTag(ctx, tt)
		tx1, err1 := db.BeginTx(ctx, nil)
		if err1 != nil {
			t.Fatal(err1)
		}
		tt.Module = fmt.Sprintf("%s%d", module, 2*i+1)
		ctx = goracle.ContextWithTraceTag(ctx, tt)
		tx2, err2 := db.BeginTx(ctx, nil)
		if err2 != nil {
			if strings.Contains(err2.Error(), "ORA-12516:") {
				tx1.Rollback()
				break
			}
			t.Fatal(err2)
		}
		if n, err = sessCount(); err != nil {
			t.Log(err)
		} else if n == 0 {
			t.Error("sessCount=0, want at least 2")
		} else {
			t.Log(n)
		}
		tx1.Rollback()
		tx2.Rollback()
	}
	if n, err = sessCount(); err != nil {
		t.Log(err)
	} else if n > 4 {
		t.Error("sessCount:", n)
	}
}

func TestOpenBadMemory(t *testing.T) {
	var mem runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&mem)
	t.Log("Allocated 0:", mem.Alloc)
	zero := mem.Alloc
	for i := 0; i < 100; i++ {
		badConStr := strings.Replace(testConStr, "@", fmt.Sprintf("BAD%dBAD@", i), 1)
		db, err := sql.Open("goracle", badConStr)
		if err != nil {
			t.Fatalf("bad connection string %q didn't produce error!", badConStr)
		}
		db.Close()
		runtime.GC()
		runtime.ReadMemStats(&mem)
		t.Logf("Allocated %d: %d", i+1, mem.Alloc)
	}
	d := mem.Alloc - zero
	t.Logf("atlast: %d", d)
	if d > 64<<10 {
		t.Errorf("Consumed more than 64KiB of memory: %d", d)
	}
}

func TestSelectFloat(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	tbl := "test_numbers" + tblSuffix
	qry := `CREATE TABLE ` + tbl + ` (
  INT_COL     NUMBER,
  FLOAT_COL  NUMBER,
  EMPTY_INT_COL NUMBER
)`
	testDb.Exec("DROP TABLE " + tbl)
	if _, err := testDb.ExecContext(ctx, qry); err != nil {
		t.Fatal(errors.Wrap(err, qry))
	}
	defer testDb.Exec("DROP TABLE " + tbl)

	const INT, FLOAT = 1234567, 4.5
	qry = `INSERT INTO ` + tbl + //nolint:gas
		` (INT_COL, FLOAT_COL, EMPTY_INT_COL)
     VALUES (1234567, 45/10, NULL)`
	if _, err := testDb.ExecContext(ctx, qry); err != nil {
		t.Fatal(errors.Wrap(err, qry))
	}

	qry = "SELECT int_col, float_col, empty_int_col FROM " + tbl //nolint:gas
	type numbers struct {
		Int     int
		Int64   int64
		Float   float64
		NInt    sql.NullInt64
		String  string
		NString sql.NullString
		Number  goracle.Number
	}
	var n numbers
	var i1, i2, i3 interface{}
	for tName, tC := range map[string]struct {
		Dest [3]interface{}
		Want numbers
	}{
		"int,float,nstring": {
			Dest: [3]interface{}{&n.Int, &n.Float, &n.NString},
			Want: numbers{Int: INT, Float: FLOAT},
		},
		"inf,float,Number": {
			Dest: [3]interface{}{&n.Int, &n.Float, &n.Number},
			Want: numbers{Int: INT, Float: FLOAT},
		},
		"int64,float,nullInt": {
			Dest: [3]interface{}{&n.Int64, &n.Float, &n.NInt},
			Want: numbers{Int64: INT, Float: FLOAT},
		},
		"intf,intf,intf": {
			Dest: [3]interface{}{&i1, &i2, &i3},
			Want: numbers{Int64: INT, Float: FLOAT},
		},
		"int,float,string": {
			Dest: [3]interface{}{&n.Int, &n.Float, &n.String},
			Want: numbers{Int: INT, Float: FLOAT},
		},
	} {
		i1, i2, i3 = nil, nil, nil
		n = numbers{}
		F := func() error {
			return errors.Wrap(
				testDb.QueryRowContext(ctx, qry).Scan(tC.Dest[0], tC.Dest[1], tC.Dest[2]),
				qry)
		}
		if err := F(); err != nil {
			if strings.HasSuffix(err.Error(), "unsupported Scan, storing driver.Value type <nil> into type *string") {
				t.Log("WARNING:", err)
				continue
			}
			noLogging := tl.enableLogging(t)
			err = F()
			t.Errorf("%q: %v", tName, errors.Wrap(err, qry))
			noLogging()
			continue
		}
		if tName == "intf,intf,intf" {
			t.Logf("%q: %#v, %#v, %#v", tName, i1, i2, i3)
			continue
		}
		t.Logf("%q: %+v", tName, n)
		if n != tC.Want {
			t.Errorf("%q:\ngot\t%+v,\nwanted\t%+v.", tName, n, tC.Want)
		}
	}
}

func TestNumInputs(t *testing.T) {
	t.Parallel()
	var a, b string
	if err := testDb.QueryRow("SELECT :1, :2 FROM DUAL", 'a', 'b').Scan(&a, &b); err != nil {
		t.Errorf("two inputs: %+v", err)
	}
	if err := testDb.QueryRow("SELECT :a, :b FROM DUAL", 'a', 'b').Scan(&a, &b); err != nil {
		t.Errorf("two named inputs: %+v", err)
	}
	if err := testDb.QueryRow("SELECT :a, :a FROM DUAL", sql.Named("a", a)).Scan(&a, &b); err != nil {
		t.Errorf("named inputs: %+v", err)
	}
}

func TestPtrArg(t *testing.T) {
	t.Parallel()
	s := "dog"
	rows, err := testDb.Query("SELECT * FROM user_objects WHERE object_name=:1", &s)
	if err != nil {
		t.Fatal(err)
	}
	rows.Close()
}
func TestORA1000(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rows, err := testDb.QueryContext(ctx, "SELECT * FROM user_objects")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for i := 0; i < 1000; i++ {
		var n int64
		if err := testDb.QueryRowContext(ctx,
			"SELECT /*"+strconv.Itoa(i)+"*/ 1 FROM DUAL", //nolint:gas
		).Scan(&n); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRanaOraIssue244(t *testing.T) {
	tableName := "test_ora_issue_244" + tblSuffix
	qry := "CREATE TABLE " + tableName + " (FUND_ACCOUNT VARCHAR2(18) NOT NULL, FUND_CODE VARCHAR2(6) NOT NULL, BUSINESS_FLAG NUMBER(10) NOT NULL, MONEY_TYPE VARCHAR2(3) NOT NULL)"
	testDb.Exec("DROP TABLE " + tableName)
	if _, err := testDb.Exec(qry); err != nil {
		t.Fatal(errors.Wrap(err, qry))
	}
	var max int
	ctx, cancel := context.WithCancel(context.Background())
	txs := make([]*sql.Tx, 0, maxSessions)
	for max = 0; max < maxSessions; max++ {
		tx, err := testDb.BeginTx(ctx, nil)
		if err != nil {
			max--
			break
		}
		txs = append(txs, tx)
	}
	cancel()
	for _, tx := range txs {
		tx.Rollback()
	}
	t.Logf("maxSessions=%d max=%d", maxSessions, max)

	defer testDb.Exec("DROP TABLE " + tableName)
	const bf = "143"
	const sc = "270004"
	qry = "INSERT INTO " + tableName + " (fund_account, fund_code, business_flag, money_type) VALUES (:1, :2, :3, :4)" //nolint:gas
	stmt, err := testDb.Prepare(qry)
	if err != nil {
		t.Fatal(errors.Wrap(err, qry))
	}
	fas := []string{"14900666", "1868091", "1898964", "14900397"}
	for _, v := range fas {
		if _, err := stmt.Exec(v, sc, bf, "0"); err != nil {
			stmt.Close()
			t.Fatal(err)
		}
	}
	stmt.Close()

	dur := time.Minute / 2
	if testing.Short() {
		dur = 10 * time.Second
	}
	ctx, cancel = context.WithTimeout(context.Background(), dur)
	defer cancel()

	qry = `SELECT fund_account, money_type FROM ` + tableName + ` WHERE business_flag = :1 AND fund_code = :2 AND fund_account = :3` //nolint:gas
	grp, ctx := errgroup.WithContext(ctx)
	for i := 0; i < max; i++ {
		index := rand.Intn(len(fas))
		i := i
		grp.Go(func() error {
			tx, err := testDb.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
			if err != nil {
				return err
			}
			defer tx.Rollback()

			stmt, err := tx.Prepare(qry)
			if err != nil {
				return errors.Wrapf(err, "%d.Prepare %q", i, err)
			}
			defer stmt.Close()

			for {
				index = (index + 1) % len(fas)
				rows, err := stmt.Query(bf, sc, fas[index])
				if err != nil {
					return errors.Wrapf(err, "%d.tx=%p stmt=%p %q", i, tx, stmt, qry)
				}

				for rows.Next() {
					if err = ctx.Err(); err != nil {
						rows.Close()
						return err
					}
					var acc, mt string
					if err = rows.Scan(&acc, &mt); err != nil {
						rows.Close()
						return err
					}

					if acc != fas[index] {
						rows.Close()
						return errors.Errorf("got acc %q, wanted %q", acc, fas[index])
					}
					if mt != "0" {
						rows.Close()
						return errors.Errorf("got mt %q, wanted 0", mt)
					}
				}
				if err = rows.Err(); err != nil {
					return err
				}
			}
		})
	}
	if err := grp.Wait(); err != nil && errors.Cause(err) != context.DeadlineExceeded {
		errS := errors.Cause(err).Error()
		switch errS {
		case "sql: statement is closed",
			"sql: transaction has already been committed or rolled back":
			return
		}
		if strings.Contains(errS, "ORA-12516:") {
			t.Log(err)
		} else {
			t.Error(err)
		}
	}
}

func TestNumberMarshal(t *testing.T) {
	t.Parallel()
	var n goracle.Number
	if err := testDb.QueryRow("SELECT 6000370006565900000073 FROM DUAL").Scan(&n); err != nil {
		t.Fatal(err)
	}
	t.Log(n.String())
	b, err := n.MarshalJSON()
	t.Logf("%s", b)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(b, []byte{'e'}) {
		t.Errorf("got %q, wanted without scientific notation", b)
	}
	if b, err = json.Marshal(struct {
		N goracle.Number
	}{N: n},
	); err != nil {
		t.Fatal(err)
	}
	t.Logf("%s", b)
}

func TestExecHang(t *testing.T) {
	defer tl.enableLogging(t)()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	done := make(chan error, 3)
	var wg sync.WaitGroup
	for i := 0; i < cap(done); i++ {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()
			if err := ctx.Err(); err != nil {
				done <- err
				return
			}
			_, err := testDb.ExecContext(ctx, "DECLARE v_deadline DATE := SYSDATE + 3/24/3600; v_db PLS_INTEGER; BEGIN LOOP SELECT COUNT(0) INTO v_db FROM cat; EXIT WHEN SYSDATE >= v_deadline; END LOOP; END;")
			if err == nil {
				done <- errors.Errorf("%d. wanted timeout got %v", i, err)
			}
			t.Logf("%d. %v", i, err)
		}()
	}
	wg.Wait()
	close(done)
	if err := <-done; err != nil {
		t.Fatal(err)
	}

}

func TestNumberNull(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	testDb.Exec("DROP TABLE number_test")
	qry := `CREATE TABLE number_test (
		caseNum NUMBER(3),
		precisionNum NUMBER(5),
      precScaleNum NUMBER(5, 0),
		normalNum NUMBER
		)`
	if _, err := testDb.ExecContext(ctx, qry); err != nil {
		t.Fatal(errors.Wrap(err, qry))
	}
	defer testDb.Exec("DROP TABLE number_test")

	qry = `
		INSERT ALL
		INTO number_test (caseNum, precisionNum, precScaleNum, normalNum) VALUES (1, 4, 65, 123)
		INTO number_test (caseNum, precisionNum, precScaleNum, normalNum) VALUES (2, NULL, NULL, NULL)
		INTO number_test (caseNum, precisionNum, precScaleNum, normalNum) VALUES (3, NULL, NULL, NULL)
		INTO number_test (caseNum, precisionNum, precScaleNum, normalNum) VALUES (4, NULL, 42, NULL)
		INTO number_test (caseNum, precisionNum, precScaleNum, normalNum) VALUES (5, NULL, NULL, 31)
		INTO number_test (caseNum, precisionNum, precScaleNum, normalNum) VALUES (6, 3, 3, 4)
		INTO number_test (caseNum, precisionNum, precScaleNum, normalNum) VALUES (7, NULL, NULL, NULL)
		INTO number_test (caseNum, precisionNum, precScaleNum, normalNum) VALUES (8, 6, 9, 7)
		SELECT 1 FROM DUAL`
	if _, err := testDb.ExecContext(ctx, qry); err != nil {
		t.Fatal(errors.Wrap(err, qry))
	}
	qry = "SELECT precisionNum, precScaleNum, normalNum FROM number_test ORDER BY caseNum"
	rows, err := testDb.Query(qry)
	if err != nil {
		t.Fatal(errors.Wrap(err, qry))
	}
	defer rows.Close()

	for rows.Next() {
		var precisionNum, recScaleNum, normalNum sql.NullInt64
		if err = rows.Scan(&precisionNum, &recScaleNum, &normalNum); err != nil {
			t.Fatal(err)
		}

		t.Log(precisionNum, recScaleNum, normalNum)

		if precisionNum.Int64 == 0 && precisionNum.Valid {
			t.Errorf("precisionNum=%v, wanted {0 false}", precisionNum)
		}
		if recScaleNum.Int64 == 0 && recScaleNum.Valid {
			t.Errorf("recScaleNum=%v, wanted {0 false}", recScaleNum)
		}
	}

	rows, err = testDb.Query(qry)
	if err != nil {
		t.Fatal(errors.Wrap(err, qry))
	}
	defer rows.Close()

	for rows.Next() {
		var precisionNumStr, recScaleNumStr, normalNumStr sql.NullString
		if err = rows.Scan(&precisionNumStr, &recScaleNumStr, &normalNumStr); err != nil {
			t.Fatal(err)
		}
		t.Log(precisionNumStr, recScaleNumStr, normalNumStr)
	}
}

func TestNullFloat(t *testing.T) {
	t.Parallel()
	testDb.Exec("DROP TABLE test_char")
	if _, err := testDb.Exec(`CREATE TABLE test_char (
			CHARS VARCHAR2(10 BYTE),
			FLOATS NUMBER(10, 2)
		)`); err != nil {
		t.Fatal(err)
	}
	defer testDb.Exec("DROP TABLE test_char")

	tx, err := testDb.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		"INSERT INTO test_char VALUES(:CHARS, :FLOATS)",
		[]string{"dog", "", "cat"},
		/*[]sql.NullString{sql.NullString{"dog", true},
		sql.NullString{"", false},
		sql.NullString{"cat", true}},*/
		[]sql.NullFloat64{
			{Float64: 3.14, Valid: true},
			{Float64: 12.36, Valid: true},
			{Float64: 0.0, Valid: false},
		},
	)
	if err != nil {
		t.Error(err)
	}
}

func TestColumnSize(t *testing.T) {
	t.Parallel()
	testDb.Exec("DROP TABLE test_column_size")
	if _, err := testDb.Exec(`CREATE TABLE test_column_size (
		vc20b VARCHAR2(20 BYTE),
		vc1b VARCHAR2(1 BYTE),
		nvc20 NVARCHAR2(20),
		nvc1 NVARCHAR2(1),
		vc20c VARCHAR2(20 CHAR),
		vc1c VARCHAR2(1 CHAR)
	)`); err != nil {
		t.Fatal(err)
	}
	defer testDb.Exec("DROP TABLE test_column_size")

	r, err := testDb.Query("SELECT * FROM test_column_size")
	if err != nil {
		t.Fatal(err)
	}
	rts, err := r.ColumnTypes()
	if err != nil {
		t.Fatal(err)
	}
	for _, col := range rts {
		l, _ := col.Length()

		t.Logf("Column %q has length %v", col.Name(), l)
	}
}

func TestReturning(t *testing.T) {
	t.Parallel()
	defer tl.enableLogging(t)()
	testDb.Exec("DROP TABLE test_returning")
	if _, err := testDb.Exec("CREATE TABLE test_returning (a VARCHAR2(20))"); err != nil {
		t.Fatal(err)
	}
	defer testDb.Exec("DROP TABLE test_returning")

	want := "abraca dabra"
	var got string
	if _, err := testDb.Exec(
		`INSERT INTO test_returning (a) VALUES (UPPER(:1)) RETURNING a INTO :2`,
		want, sql.Out{Dest: &got},
	); err != nil {
		t.Fatal(err)
	}
	want = strings.ToUpper(want)
	if want != got {
		t.Errorf("got %q, wanted %q", got, want)
	}

	if _, err := testDb.Exec(
		`UPDATE test_returning SET a = '1' WHERE 1=0 RETURNING a /*LASTINSERTID*/ INTO :1`,
		sql.Out{Dest: &got},
	); err != nil {
		t.Fatal(err)
	}
	t.Logf("RETURNING (zero set): %v", got)
}

func TestMaxOpenCursors(t *testing.T) {
	var openCursors sql.NullInt64
	const qry1 = "SELECT p.value FROM v$parameter p WHERE p.name = 'open_cursors'"
	if err := testDb.QueryRow(qry1).Scan(&openCursors); err != nil {
		t.Log(errors.Wrap(err, qry1))
	} else {
		t.Logf("open_cursors=%v", openCursors)
	}
	n := int(openCursors.Int64)
	if n <= 0 {
		n = 1000
	}
	n *= 2
	for i := 0; i < n; i++ {
		var cnt int64
		const qry2 = "DECLARE cnt PLS_INTEGER; BEGIN SELECT COUNT(0) INTO cnt FROM DUAL; :1 := cnt; END;"
		if _, err := testDb.Exec(qry2, sql.Out{Dest: &cnt}); err != nil {
			t.Fatal(errors.Wrapf(err, "%d. %s", i, qry2))
		}
	}
}

func TestRO(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tx, err := testDb.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable, ReadOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err = tx.QueryContext(ctx, "SELECT 1 FROM DUAL"); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.ExecContext(ctx, "CREATE TABLE test_table (i INTEGER)"); err == nil {
		t.Log("RO allows CREATE TABLE ?")
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
}

func TestNullIntoNum(t *testing.T) {
	t.Parallel()
	testDb.Exec("DROP TABLE test_null_num")
	qry := "CREATE TABLE test_null_num (i NUMBER(3))"
	if _, err := testDb.Exec(qry); err != nil {
		t.Fatal(errors.Wrap(err, qry))
	}
	defer testDb.Exec("DROP TABLE test_null_num")

	qry = "INSERT INTO test_null_num (i) VALUES (:1)"
	var i *int
	if _, err := testDb.Exec(qry, i); err != nil {
		t.Fatal(errors.Wrap(err, qry))
	}
}

func TestPing(t *testing.T) {
	t.Parallel()
	badDB, err := sql.Open("goracle", "bad/passw@1.1.1.1")
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	dl, _ := ctx.Deadline()
	err = badDB.PingContext(ctx)
	ok := dl.After(time.Now())
	if err != nil {
		t.Log(err)
	} else {
		t.Log("ping succeeded")
		if !ok {
			t.Error("ping succeeded after deadline!")
		}
	}
}

func TestNoConnectionPooling(t *testing.T) {
	t.Parallel()
	db, err := sql.Open("goracle",
		strings.Replace(
			strings.Replace(testConStr, "POOLED", goracle.NoConnectionPoolingConnectionClass, 1),
			"standaloneConnection=0", "standaloneConnection=1", 1,
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()
}

func TestExecTimeout(t *testing.T) {
	t.Parallel()
	defer tl.enableLogging(t)()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if _, err := testDb.ExecContext(ctx, "SELECT COUNT(DISTINCT ORA_HASH(A.table_name)) from cat, cat, cat A"); err != nil {
		t.Log(err)
	}
}

func TestQueryTimeout(t *testing.T) {
	t.Parallel()
	defer tl.enableLogging(t)()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if _, err := testDb.QueryContext(ctx, "SELECT COUNT(0) FROM all_objects, all_objects"); err != nil {
		t.Log(err)
	}
}

func TestSDO(t *testing.T) {
	//t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	innerQry := `SELECT MDSYS.SDO_GEOMETRY(
	3001,
	NULL,
	NULL,
	MDSYS.SDO_ELEM_INFO_ARRAY(
		1,1,1,4,1,0,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL
	),
	MDSYS.SDO_ORDINATE_ARRAY(
		480736.567,10853969.692,0,0.998807402795312,-0.0488238888381834,0,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL)
		) SHAPE FROM DUAL`
	selectQry := `SELECT shape, DUMP(shape), CASE WHEN shape IS NULL THEN 'I' ELSE 'N' END FROM (` + innerQry + ")"
	rows, err := testDb.QueryContext(ctx, selectQry)
	if err != nil {
		if !strings.Contains(err.Error(), `ORA-00904: "MDSYS"."SDO_GEOMETRY"`) {
			t.Fatal(errors.Wrap(err, selectQry))
		}
		for _, qry := range []string{
			`CREATE TYPE test_sdo_point_type AS OBJECT (
			   X NUMBER,
			   Y NUMBER,
			   Z NUMBER)`,
			"CREATE TYPE test_sdo_elem_info_array AS VARRAY (1048576) of NUMBER",
			"CREATE TYPE test_sdo_ordinate_array AS VARRAY (1048576) of NUMBER",
			`CREATE TYPE test_sdo_geometry AS OBJECT (
			 SDO_GTYPE NUMBER,
			 SDO_SRID NUMBER,
			 SDO_POINT test_SDO_POINT_TYPE,
			 SDO_ELEM_INFO test_SDO_ELEM_INFO_ARRAY,
			 SDO_ORDINATES test_SDO_ORDINATE_ARRAY)`,

			`CREATE TABLE test_sdo(
					id INTEGER not null,
					shape test_SDO_GEOMETRY not null
				)`,
		} {
			var drop string
			if strings.HasPrefix(qry, "CREATE TYPE") {
				drop = "DROP TYPE " + qry[12:strings.Index(qry, " AS")] + " FORCE"
			} else {
				drop = "DROP TABLE " + qry[13:strings.Index(qry, "(")]
			}
			testDb.ExecContext(ctx, drop)
			t.Log(drop)
			if _, err := testDb.ExecContext(ctx, qry); err != nil {
				err = errors.Wrap(err, qry)
				t.Log(err)
				if !strings.Contains(err.Error(), "ORA-01031:") {
					t.Fatal(err)
				}
				t.Skip(err)
			}
			defer testDb.ExecContext(ctx, drop)
		}

		selectQry = strings.Replace(selectQry, "MDSYS.SDO_", "test_SDO_", -1)
		if rows, err = testDb.QueryContext(ctx, selectQry); err != nil {
			t.Fatal(errors.Wrap(err, selectQry))
		}

	}
	defer rows.Close()
	if false {
		goracle.Log = func(kv ...interface{}) error {
			t.Helper()
			t.Log(kv)
			return nil
		}
	}
	for rows.Next() {
		var dmp, isNull string
		var intf interface{}
		if err = rows.Scan(&intf, &dmp, &isNull); err != nil {
			t.Error(errors.Wrap(err, "scan"))
		}
		t.Log(dmp, isNull)
		obj := intf.(*goracle.Object)
		//t.Log("obj:", obj)
		printObj(t, "", obj)
	}
	if err = rows.Err(); err != nil {
		t.Fatal(err)
	}
}

func printObj(t *testing.T, name string, obj *goracle.Object) {
	if obj == nil {
		return
	}
	for key := range obj.Attributes {
		sub, err := obj.Get(key)
		t.Logf("%s.%s. %+v (err=%+v)\n", name, key, sub, err)
		if err != nil {
			t.Errorf("ERROR: %+v", err)
		}
		if ss, ok := sub.(*goracle.Object); ok {
			printObj(t, name+"."+key, ss)
		} else if coll, ok := sub.(*goracle.ObjectCollection); ok {
			slice, err := coll.AsSlice(nil)
			t.Logf("%s.%s. %+v", name, key, slice)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
}

var _ = driver.Valuer((*Custom)(nil))

type Custom struct {
	Num int64
}

func (t *Custom) Value() (driver.Value, error) {
	return t.Num, nil
}

func TestSelectCustomType(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := testDb.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	tbl := "test_custom_type" + tblSuffix
	conn.ExecContext(ctx, "DROP TABLE "+tbl)
	qry := "CREATE TABLE " + tbl + " (nm VARCHAR2(30), typ VARCHAR2(30), id NUMBER(6), created DATE)"
	if _, err = conn.ExecContext(ctx, qry); err != nil {
		t.Fatal(errors.Wrap(err, qry))
	}
	defer testDb.Exec("DROP TABLE " + tbl)

	n := 1000
	nms, typs, ids, createds := make([]string, n), make([]string, n), make([]int, n), make([]time.Time, n)
	now := time.Now()
	for i := range nms {
		nms[i], typs[i], ids[i], createds[i] = fmt.Sprintf("obj-%d", i), "OBJECT", i, now.Add(-time.Duration(i)*time.Second)
	}
	qry = "INSERT INTO " + tbl + " (nm, typ, id, created) VALUES (:1, :2, :3, :4)"
	if _, err = conn.ExecContext(ctx, qry, nms, typs, ids, createds); err != nil {
		t.Fatal(errors.Wrap(err, qry))
	}

	const num = 10
	nums := &Custom{Num: num}
	type underlying int64
	numbers := underlying(num)
	rows, err := conn.QueryContext(ctx,
		"SELECT nm, typ, id, created FROM "+tbl+" WHERE ROWNUM < COALESCE(:alpha, :beta, 2) ORDER BY id",
		sql.Named("alpha", nums),
		goracle.MagicTypeConversion(), sql.Named("beta", numbers),
	)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	n = 0
	oldOid := int64(0)
	for rows.Next() {
		var tbl, typ string
		var oid int64
		var created time.Time
		if err := rows.Scan(&tbl, &typ, &oid, &created); err != nil {
			t.Fatal(err)
		}
		t.Log(tbl, typ, oid, created)
		if tbl == "" {
			t.Fatal("empty tbl")
		}
		n++
		if oldOid > oid {
			t.Errorf("got oid=%d, wanted sth < %d.", oid, oldOid)
		}
		oldOid = oid
	}
	if n == 0 || n > num {
		t.Errorf("got %d rows, wanted %d", n, num)
	}
}

func TestExecInt64(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	qry := `CREATE OR REPLACE PROCEDURE test_i64_out(p_int NUMBER, p_out1 OUT NUMBER, p_out2 OUT NUMBER) IS
	BEGIN p_out1 := p_int; p_out2 := p_int; END;`
	if _, err := testDb.ExecContext(ctx, qry); err != nil {
		t.Fatal(err)
	}
	defer testDb.ExecContext(ctx, "DROP PROCEDURE test_i64_out")

	qry = "BEGIN test_i64_out(:1, :2, :3); END;"
	var num sql.NullInt64
	var str string
	defer tl.enableLogging(t)()
	if _, err := testDb.ExecContext(ctx, qry, 3.14, sql.Out{Dest: &num}, sql.Out{Dest: &str}); err != nil {
		t.Fatal(err)
	}
	t.Log("num:", num, "str:", str)
}

func TestImplicitResults(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	const qry = `declare
			c0 sys_refcursor;
            c1 sys_refcursor;
            c2 sys_refcursor;
        begin
			:1 := c0;
            open c1 for
            select 1 from DUAL;
            dbms_sql.return_result(c1);
            open c2 for
            select 'A' from DUAL;
            dbms_sql.return_result(c2);
        end;`
	var rows driver.Rows
	if _, err := testDb.ExecContext(ctx, qry, sql.Out{Dest: &rows}); err != nil {
		if strings.Contains(err.Error(), "PLS-00302:") {
			t.Skip()
		}
		t.Fatal(errors.Wrap(err, qry))
	}
	r := rows.(driver.RowsNextResultSet)
	for r.HasNextResultSet() {
		if err := r.NextResultSet(); err != nil {
			t.Error(err)
		}
	}
}

func TestStartupShutdown(t *testing.T) {
	if os.Getenv("GORACLE_DB_SHUTDOWN") != "1" {
		t.Skip("GORACLE_DB_SHUTDOWN != 1, skipping shutdown/startup test")
	}
	p, err := goracle.ParseConnString(testConStr)
	if err != nil {
		t.Fatal(errors.Wrap(err, testConStr))
	}
	if !(p.IsSysDBA || p.IsSysOper) {
		p.IsSysDBA = true
	}
	if !p.IsPrelim {
		p.IsPrelim = true
	}
	db, err := sql.Open("goracle", p.StringWithPassword())
	if err != nil {
		t.Fatal(err, p.StringWithPassword())
	}
	defer db.Close()
	conn, err := goracle.DriverConn(db)
	if err != nil {
		t.Fatal(err)
	}
	if err = conn.Shutdown(goracle.ShutdownTransactionalLocal); err != nil {
		t.Error(err)
	}
	if err = conn.Shutdown(goracle.ShutdownFinal); err != nil {
		t.Error(err)
	}
	if err = conn.Startup(goracle.StartupDefault); err != nil {
		t.Error(err)
	}
}

func TestIssue134(t *testing.T) {
	const crea = `CREATE OR REPLACE TYPE test_PRJ_TASK_OBJ_TYPE AS OBJECT (
	PROJECT_NUMBER VARCHAR2(100)
	,SOURCE_ID VARCHAR2(100)
	,TASK_NAME VARCHAR2(300)
	,TASK_DESCRIPTION VARCHAR2(2000)
	,TASK_START_DATE DATE
	,TASK_END_DATE DATE
	,TASK_COST NUMBER
	,SOURCE_PARENT_ID NUMBER
	,TASK_TYPE VARCHAR2(100)
	,QUANTITY NUMBER );
CREATE OR REPLACE TYPE test_PRJ_TASK_TAB_TYPE IS TABLE OF test_PRJ_TASK_OBJ_TYPE;
CREATE OR REPLACE PROCEDURE test_CREATE_TASK_ACTIVITY (p_create_task_i IN PRJ_TASK_TAB_TYPE,
	p_create_activity_i IN PRJ_ACTIVITY_TAB_TYPE,
	p_project_id_i IN NUMBER) IS BEGIN NULL; END;`
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for _, qry := range strings.Split(crea, ";\n") {
		if strings.HasSuffix(qry, " END") {
			qry += ";"
		}
		if _, err := testDb.ExecContext(ctx, qry); err != nil {
			t.Fatal(errors.Wrap(err, qry))
		}
	}
	defer func() {
		for _, qry := range []string{
			`DROP TYPE test_prj_task_tab_type`,
			`DROP TYPE test_prj_task_obj_type`,
			`DROP PROCEDURE test_create_task_activity`,
		} {
			testDb.Exec(qry)
		}
	}()

	var o1, o2 goracle.Object
	qry := "BEGIN :1 := test_prj_task_tab_type(); END;"
	if _, err := testDb.ExecContext(ctx, qry, sql.Out{Dest: &o1}); err != nil {
		t.Fatal(errors.Wrap(err, qry))
	}
	if _, err := testDb.ExecContext(ctx, qry, sql.Out{Dest: &o2}); err != nil {
		t.Fatal(errors.Wrap(err, qry))
	}
	qry = "BEGIN test_create_task_activity(:1, :2, :3); END;"
	if _, err := testDb.ExecContext(ctx, qry, o1, o2, 1); err != nil {
		t.Error(err)
	}
}

func TestTsTZ(t *testing.T) {
	t.Parallel()
	qry := "SELECT FROM_TZ(TO_TIMESTAMP('2019-05-01 09:39:12', 'YYYY-MM-DD HH24:MI:SS'), '{{.TZ}}') FROM DUAL"
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	defer tl.enableLogging(t)()
	var ts time.Time
	{
		qry := strings.Replace(qry, "{{.TZ}}", "01:00", 1)
		if err := testDb.QueryRowContext(ctx, qry).Scan(&ts); err != nil {
			t.Fatal(errors.Wrap(err, qry))
		}
	}
	qry = strings.Replace(qry, "{{.TZ}}", "Europe/Berlin", 1)
	err := testDb.QueryRowContext(ctx, qry).Scan(&ts)
	if err != nil {
		t.Log(errors.Wrap(err, qry))
	}
	t.Log(ts)
	if !ts.IsZero() {
		return
	}

	qry = "SELECT filename, version FROM v$timezone_file"
	rows, err := testDb.QueryContext(ctx, qry)
	if err != nil {
		t.Log(qry, err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var fn, ver string
		if err := rows.Scan(&fn, &ver); err != nil {
			t.Log(qry, err)
			continue
		}
		t.Log(fn, ver)
	}
	t.Skip("wanted non-zero time")
}

func TestGetDBTimeZone(t *testing.T) {
	t.Parallel()
	defer tl.enableLogging(t)()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	qry := "SELECT SESSIONTIMEZONE FROM DUAL"
	var tz string
	if err := testDb.QueryRowContext(ctx, qry).Scan(&tz); err != nil {
		t.Fatal(errors.Wrap(err, qry))
	}
	t.Log("timezone:", tz)

	for _, timS := range []string{"2006-07-08", "2006-01-02"} {
		localTime, err := time.ParseInLocation("2006-01-02", timS, time.Local)
		if err != nil {
			t.Fatal(err)
		}
		qry = "SELECT TO_DATE('" + timS + " 00:00:00', 'YYYY-MM-DD HH24:MI:SS') FROM DUAL"
		var dbTime time.Time
		t.Log("local:", localTime.Format(time.RFC3339))
		if err := testDb.QueryRowContext(ctx, qry).Scan(&dbTime); err != nil {
			t.Fatal(errors.Wrap(err, qry))
		}
		t.Log("db:", dbTime.Format(time.RFC3339))
		if !dbTime.Equal(localTime) {
			t.Errorf("db says %s, local is %s", dbTime.Format(time.RFC3339), localTime.Format(time.RFC3339))
		}
	}
}
