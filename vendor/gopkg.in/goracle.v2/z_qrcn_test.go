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
	"strings"
	"testing"

	"github.com/pkg/errors"
	goracle "gopkg.in/goracle.v2"
)

func TestQRCN(t *testing.T) {
	conn, err := goracle.DriverConn(testDb)
	if err != nil {
		t.Fatal(err)
	}

	testDb.Exec("DROP TABLE test_subscr")
	if _, err = testDb.Exec("CREATE TABLE test_subscr (i NUMBER)"); err != nil {
		t.Fatal(err)
	}
	defer testDb.Exec("DROP TABLE test_subscr")

	var events []goracle.Event
	cb := func(e goracle.Event) {
		t.Log(e)
		events = append(events, e)
	}
	s, err := conn.NewSubscription("subscr", cb)
	if err != nil {
		errS := errors.Cause(err).Error()
		if strings.Contains(errS, "ORA-29970:") {
			t.Skip(err.Error())
		} else if strings.Contains(errS, "ORA-29972:") {
			t.Log("See \"https://docs.oracle.com/database/121/ADFNS/adfns_cqn.htm#ADFNS553\"")
			var User string
			_ = testDb.QueryRow("SELECT USER FROM DUAL").Scan(&User)
			//t.Log("GRANT EXECUTE ON DBMS_CQ_NOTIFICATION TO "+User)
			t.Log("GRANT CHANGE NOTIFICATION TO " + User + ";")
			t.Skip(err.Error())
		}
		t.Fatalf("%+v", err)
	}
	defer s.Close()
	if err := s.Register("SELECT COUNT(0) FROM test_subscr"); err != nil {
		t.Fatalf("%+v", err)
	}
	qry := "SELECT regid, table_name FROM USER_CHANGE_NOTIFICATION_REGS"
	rows, err := testDb.Query(qry)
	if err != nil {
		t.Fatal(errors.Wrap(err, qry))
	}
	t.Log("--- Registrations ---")
	for rows.Next() {
		var regID, table string
		if err := rows.Scan(&regID,&table); err != nil {
			t.Error(err)
			break
		}
		t.Logf("%s: %s", regID, table)
	}
	t.Log("---------------------")
	rows.Close()
	testDb.Exec("INSERT INTO test_subscr (i) VALUES (1)")
	testDb.Exec("INSERT INTO test_subscr (i) VALUES (0)")
	t.Log("events:", events)
}
