// Copyright 2019 Walter Wanderley
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
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	goracle "gopkg.in/goracle.v2"
)

var _ goracle.ObjectScanner = new(MyRecord)
var _ goracle.ObjectWriter = new(MyRecord)

var _ goracle.ObjectScanner = new(MyTable)

// MYRecord represents TEST_PKG_TYPES.MY_RECORD
type MyRecord struct {
	*goracle.Object
	ID  int64
	Txt string
}

func (r *MyRecord) Scan(src interface{}) error {

	switch obj := src.(type) {
	case *goracle.Object:
		id, err := obj.Get("ID")
		if err != nil {
			return err
		}
		r.ID = id.(int64)

		txt, err := obj.Get("TXT")
		if err != nil {
			return err
		}
		r.Txt = string(txt.([]byte))

	default:
		return fmt.Errorf("Cannot scan from type %T", src)
	}

	return nil
}

// WriteObject update goracle.Object with struct attributes values.
// Implement this method if you need the record as an input parameter.
func (r MyRecord) WriteObject() error {
	// all attributes must be initialized or you get an "ORA-21525: attribute number or (collection element at index) %s violated its constraints"
	err := r.ResetAttributes()
	if err != nil {
		return err
	}

	var data goracle.Data
	err = r.GetAttribute(&data, "ID")
	if err != nil {
		return err
	}
	data.SetInt64(r.ID)
	r.SetAttribute("ID", &data)

	if r.Txt != "" {
		err = r.GetAttribute(&data, "TXT")
		if err != nil {
			return err
		}

		data.SetBytes([]byte(r.Txt))
		r.SetAttribute("TXT", &data)
	}

	return nil
}

// MYTable represents TEST_PKG_TYPES.MY_TABLE
type MyTable struct {
	*goracle.ObjectCollection
	Items []*MyRecord
}

func (t *MyTable) Scan(src interface{}) error {

	switch obj := src.(type) {
	case *goracle.Object:
		collection := goracle.ObjectCollection{Object: obj}
		t.Items = make([]*MyRecord, 0)
		for i, err := collection.First(); err == nil; i, err = collection.Next(i) {
			var data goracle.Data
			err = collection.Get(&data, i)
			if err != nil {
				return err
			}

			o := data.GetObject()
			defer o.Close()

			var item MyRecord
			err = item.Scan(o)
			if err != nil {
				return err
			}
			t.Items = append(t.Items, &item)
		}

	default:
		return fmt.Errorf("Cannot scan from type %T", src)
	}

	return nil
}

func createPackages(ctx context.Context) error {
	qry := []string{`CREATE OR REPLACE PACKAGE test_pkg_types AS
	TYPE my_record IS RECORD (
		id    NUMBER(5),
		txt   VARCHAR2(200)
	);
	TYPE my_table IS
		TABLE OF my_record;
	END test_pkg_types;`,

		`CREATE OR REPLACE PACKAGE test_pkg_sample AS
	PROCEDURE test_record (
		id    IN    NUMBER,
		txt   IN    VARCHAR,
		rec   OUT   test_pkg_types.my_record
	);
	
	PROCEDURE test_record_in (
		rec IN OUT test_pkg_types.my_record
	);
	
	FUNCTION test_table (
		x NUMBER
	) RETURN test_pkg_types.my_table;
	
	END test_pkg_sample;`,

		`CREATE OR REPLACE PACKAGE BODY test_pkg_sample AS
	
	PROCEDURE test_record (
		id    IN    NUMBER,
		txt   IN    VARCHAR,
		rec   OUT   test_pkg_types.my_record
	) IS
	BEGIN
		rec.id := id;
		rec.txt := txt;
	END test_record;
	
	PROCEDURE test_record_in (
		rec IN OUT test_pkg_types.my_record
	) IS
	BEGIN
		rec.id := rec.id + 1;
		rec.txt := rec.txt || ' changed';
	END test_record_in;
	
	FUNCTION test_table (
		x NUMBER
	) RETURN test_pkg_types.my_table IS
		tb     test_pkg_types.my_table;
		item   test_pkg_types.my_record;
	BEGIN
		tb := test_pkg_types.my_table();
		FOR c IN (
			SELECT
				level "LEV"
			FROM
				"SYS"."DUAL" "A1"
			CONNECT BY
				level <= x
		) LOOP
			item.id := c.lev;
			item.txt := 'test - ' || ( c.lev * 2 );
			tb.extend();
			tb(tb.count) := item;
		END LOOP;
	
		RETURN tb;
	END test_table;
	
	END test_pkg_sample;`}

	for _, ddl := range qry {
		_, err := testDb.ExecContext(ctx, ddl)
		if err != nil {
			return err

		}
	}

	return nil
}

func dropPackages(ctx context.Context) {
	testDb.ExecContext(ctx, `DROP PACKAGE test_pkg_types`)
	testDb.ExecContext(ctx, `DROP PACKAGE test_pkg_sample`)
}

func TestPLSQLTypes(t *testing.T) {
	t.Parallel()

	serverVersion, err := goracle.ServerVersion(testDb)
	if err != nil {
		t.Fatal(err)
	}
	clientVersion, err := goracle.ClientVersion(testDb)
	if err != nil {
		t.Fatal(err)
	}

	if serverVersion.Version < 12 || clientVersion.Version < 12 {
		t.Skip("client or server < 12")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = createPackages(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer dropPackages(ctx)

	conn, err := goracle.DriverConn(testDb)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("Record", func(t *testing.T) {
		// you must have execute privilege on package and use uppercase
		objType, err := conn.GetObjectType("TEST_PKG_TYPES.MY_RECORD")
		if err != nil {
			t.Fatal(err)
		}
		defer objType.Close()

		obj, err := objType.NewObject()
		if err != nil {
			t.Fatal(err)
		}
		defer obj.Close()

		for tName, tCase := range map[string]struct {
			ID   int64
			txt  string
			want MyRecord
		}{
			"default":    {ID: 1, txt: "test", want: MyRecord{obj, 1, "test"}},
			"emptyTxt":   {ID: 2, txt: "", want: MyRecord{obj, 2, ""}},
			"zeroValues": {want: MyRecord{Object: obj}},
		} {
			rec := MyRecord{Object: obj}
			params := []interface{}{
				sql.Named("id", tCase.ID),
				sql.Named("txt", tCase.txt),
				sql.Named("rec", sql.Out{Dest: &rec}),
			}
			_, err = testDb.ExecContext(ctx, `begin test_pkg_sample.test_record(:id, :txt, :rec); end;`, params...)
			if err != nil {
				t.Fatal(err)
			}

			if rec != tCase.want {
				t.Errorf("%s: record got %v, wanted %v", tName, rec, tCase.want)
			}
		}
	})

	t.Run("Record IN OUT", func(t *testing.T) {
		// you must have execute privilege on package and use uppercase
		objType, err := conn.GetObjectType("TEST_PKG_TYPES.MY_RECORD")
		if err != nil {
			t.Fatal(err)
		}
		defer objType.Close()

		for tName, tCase := range map[string]struct {
			in      MyRecord
			wantID  int64
			wantTxt string
		}{
			"zeroValues": {in: MyRecord{}, wantID: 1, wantTxt: " changed"},
			"default":    {in: MyRecord{ID: 1, Txt: "test"}, wantID: 2, wantTxt: "test changed"},
			"emptyTxt":   {in: MyRecord{ID: 2, Txt: ""}, wantID: 3, wantTxt: " changed"},
		} {

			obj, err := objType.NewObject()
			if err != nil {
				t.Fatal(err)
			}
			defer obj.Close()

			rec := MyRecord{Object: obj, ID: tCase.in.ID, Txt: tCase.in.Txt}
			params := []interface{}{
				sql.Named("rec", sql.Out{Dest: &rec, In: true}),
			}
			_, err = testDb.ExecContext(ctx, `begin test_pkg_sample.test_record_in(:rec); end;`, params...)
			if err != nil {
				t.Fatal(err)
			}

			if rec.ID != tCase.wantID {
				t.Errorf("%s: ID got %d, wanted %d", tName, rec.ID, tCase.wantID)
			}
			if rec.Txt != tCase.wantTxt {
				t.Errorf("%s: Txt got %s, wanted %s", tName, rec.Txt, tCase.wantTxt)
			}
		}
	})

	t.Run("Table Of", func(t *testing.T) {
		// you must have execute privilege on package and use uppercase
		objType, err := conn.GetObjectType("TEST_PKG_TYPES.MY_TABLE")
		if err != nil {
			t.Fatal(err)
		}
		defer objType.Close()

		items := []*MyRecord{&MyRecord{ID: 1, Txt: "test - 2"}, &MyRecord{ID: 2, Txt: "test - 4"}}

		for tName, tCase := range map[string]struct {
			in   int64
			want MyTable
		}{
			"one": {in: 1, want: MyTable{Items: items[:1]}},
			"two": {in: 2, want: MyTable{Items: items}},
		} {

			obj, err := objType.NewObject()
			if err != nil {
				t.Fatal(err)
			}
			defer obj.Close()

			tb := MyTable{ObjectCollection: &goracle.ObjectCollection{obj}}
			params := []interface{}{
				sql.Named("x", tCase.in),
				sql.Named("tb", sql.Out{Dest: &tb}),
			}
			_, err = testDb.ExecContext(ctx, `begin :tb := test_pkg_sample.test_table(:x); end;`, params...)
			if err != nil {
				t.Fatal(err)
			}

			if len(tb.Items) != len(tCase.want.Items) {
				t.Errorf("%s: table got %v items, wanted %v items", tName, len(tb.Items), len(tCase.want.Items))
			} else {
				for i := 0; i < len(tb.Items); i++ {
					got := tb.Items[i]
					want := tCase.want.Items[i]
					if got.ID != want.ID {
						t.Errorf("%s: record ID got %v, wanted %v", tName, got.ID, want.ID)
					}
					if got.Txt != want.Txt {
						t.Errorf("%s: record TXT got %v, wanted %v", tName, got.Txt, want.Txt)
					}
				}
			}
		}
	})

}
