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
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	goracle "gopkg.in/goracle.v2"
)

// go install && go test -c && ./goracle.v2.test -test.run=^$ -test.bench=Insert25 -test.cpuprofile=/tmp/insert25.prof && go tool pprof ./goracle.v2.test /tmp/insert25.prof

func BenchmarkPlSQLArrayInsert25(b *testing.B) {
	defer func() {
		//testDb.Exec("DROP TABLE tst_bench_25_tbl")
		testDb.Exec("DROP PACKAGE tst_bench_25")
	}()

	for _, qry := range []string{
		//`DROP TABLE tst_bench_25_tbl`,
		/*`CREATE TABLE tst_bench_25_tbl (dt DATE, st VARCHAR2(255),
		  ip NUMBER(12), zone NUMBER(3), plan NUMBER(3), banner NUMBER(3),
		  referrer VARCHAR2(255), country VARCHAR2(80), region VARCHAR2(10))`,*/

		`CREATE OR REPLACE PACKAGE tst_bench_25 IS
TYPE cx_array_date IS TABLE OF DATE INDEX BY BINARY_INTEGER;

TYPE cx_array_string IS TABLE OF VARCHAR2 (1000) INDEX BY BINARY_INTEGER;

TYPE cx_array_num IS TABLE OF NUMBER INDEX BY BINARY_INTEGER;

PROCEDURE P_BULK_INSERT_IMP (VIMP_DATES       cx_array_date,
                                VIMP_KEYS        cx_array_string,
                                VIMP_IP          cx_array_num,
                                VIMP_ZONE        cx_array_num,
                                VIMP_PLAN        cx_array_num,
                                VIMP_BANNER      cx_array_num,
                                VIMP_REFERRER    cx_array_string,
                                VIMP_COUNTRY     cx_array_string,
                                VIMP_REGION      cx_array_string);
END;`,
		`CREATE OR REPLACE PACKAGE BODY tst_bench_25 IS
PROCEDURE P_BULK_INSERT_IMP (VIMP_DATES       cx_array_date,
                             VIMP_KEYS        cx_array_string,
                             VIMP_IP          cx_array_num,
                             VIMP_ZONE        cx_array_num,
                             VIMP_PLAN        cx_array_num,
                             VIMP_BANNER      cx_array_num,
                             VIMP_REFERRER    cx_array_string,
                             VIMP_COUNTRY     cx_array_string,
                             VIMP_REGION      cx_array_string) IS
  i PLS_INTEGER;
BEGIN
  i := vimp_dates.FIRST;
  WHILE i IS NOT NULL LOOP
  /*
    INSERT INTO tst_bench_25_tbl
	  (dt, st, ip, zone, plan, banner, referrer, country, region)
	  VALUES (vimp_dates(i), vimp_keys(i), vimp_ip(i), vimp_zone(i), vimp_plan(i),
	          vimp_banner(i), vimp_referrer(i), vimp_country(i), vimp_region(i));
  */
    i := vimp_dates.NEXT(i);
  END LOOP;

END;

END tst_bench_25;`,
	} {

		if _, err := testDb.Exec(qry); err != nil {
			if strings.HasPrefix(qry, "DROP TABLE ") {
				continue
			}
			b.Fatal(errors.Wrap(err, qry))
		}
	}

	qry := `BEGIN tst_bench_25.P_BULK_INSERT_IMP (:1, :2, :3, :4, :5, :6, :7, :8, :9); END;`

	pt1 := time.Now()
	n := 512
	dates := make([]time.Time, n)
	keys := make([]string, n)
	ips := make([]int, n)
	zones := make([]int, n)
	plans := make([]int, n)
	banners := make([]int, n)
	referrers := make([]string, n)
	countries := make([]string, n)
	regions := make([]string, n)
	for i := range dates {
		dates[i] = pt1.Add(time.Duration(i) * time.Second)
		keys[i] = "key"
		ips[i] = 123456
		zones[i] = i % 256
		plans[i] = (i / 2) % 1000
		banners[i] = (i * 3) % 1000
		referrers[i] = "referrer"
		countries[i] = "country"
		regions[i] = "region"
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx = goracle.ContextWithLog(ctx, nil)
	tx, err := testDb.BeginTx(ctx, nil)
	if err != nil {
		b.Fatal(err)
	}
	defer tx.Rollback()

	b.ResetTimer()
	for i := 0; i < b.N; i += n {
		if _, err := tx.ExecContext(ctx, qry,
			goracle.PlSQLArrays,
			dates, keys, ips, zones, plans, banners, referrers, countries, regions,
		); err != nil {
			if strings.Contains(err.Error(), "PLS-00905") || strings.Contains(err.Error(), "ORA-06508") {
				b.Log(goracle.GetCompileErrors(testDb, false))
			}
			//b.Log(dates, keys, ips, zones, plans, banners, referrers, countries, regions)
			b.Fatal(err)
		}
	}
	b.StopTimer()
}

// go install && go test -c && ./goracle.v2.test -test.run=^. -test.bench=InOut -test.cpuprofile=/tmp/inout.prof && go tool pprof -cum ./goracle.v2.test /tmp/inout.prof

func BenchmarkPlSQLArrayInOut(b *testing.B) {
	defer func() {
		testDb.Exec("DROP PACKAGE tst_bench_inout")
	}()

	for _, qry := range []string{
		`CREATE OR REPLACE PACKAGE tst_bench_inout IS
TYPE cx_array_date IS TABLE OF DATE INDEX BY BINARY_INTEGER;

TYPE cx_array_string IS TABLE OF VARCHAR2 (1000) INDEX BY BINARY_INTEGER;

TYPE cx_array_num IS TABLE OF NUMBER INDEX BY BINARY_INTEGER;

PROCEDURE P_BULK_INSERT_IMP (VIMP_DATES       IN OUT NOCOPY cx_array_date,
                             VIMP_KEYS        IN OUT NOCOPY cx_array_string,
                             VIMP_IP          IN OUT NOCOPY cx_array_num,
                             VIMP_ZONE        IN OUT NOCOPY cx_array_num,
                             VIMP_PLAN        IN OUT NOCOPY cx_array_num,
                             VIMP_BANNER      IN OUT NOCOPY cx_array_num,
                             VIMP_REFERRER    IN OUT NOCOPY cx_array_string,
                             VIMP_COUNTRY     IN OUT NOCOPY cx_array_string,
                             VIMP_REGION      IN OUT NOCOPY cx_array_string);
END;`,
		`CREATE OR REPLACE PACKAGE BODY tst_bench_inout IS
PROCEDURE P_BULK_INSERT_IMP (VIMP_DATES       IN OUT NOCOPY cx_array_date,
                             VIMP_KEYS        IN OUT NOCOPY cx_array_string,
                             VIMP_IP          IN OUT NOCOPY cx_array_num,
                             VIMP_ZONE        IN OUT NOCOPY cx_array_num,
                             VIMP_PLAN        IN OUT NOCOPY cx_array_num,
                             VIMP_BANNER      IN OUT NOCOPY cx_array_num,
                             VIMP_REFERRER    IN OUT NOCOPY cx_array_string,
                             VIMP_COUNTRY     IN OUT NOCOPY cx_array_string,
                             VIMP_REGION      IN OUT NOCOPY cx_array_string) IS
  i PLS_INTEGER;
BEGIN
  i := vimp_dates.FIRST;
  WHILE i IS NOT NULL LOOP
    vimp_dates(i) := vimp_dates(i) + 1;
	vimp_keys(i) := vimp_keys(i)||' '||i;
	vimp_ip(i) := -vimp_ip(i);
	vimp_zone(i) := -vimp_zone(i);
	vimp_plan(i) := -vimp_plan(i);
	vimp_banner(i) := -vimp_banner(i);
	vimp_referrer(i) := vimp_referrer(i)||' '||i;
	vimp_country(i) := vimp_country(i)||' '||i;
	vimp_region(i) := vimp_region(i)||' '||i;
    i := vimp_dates.NEXT(i);
  END LOOP;

END;

END tst_bench_inout;`,
	} {

		if _, err := testDb.Exec(qry); err != nil {
			if strings.HasPrefix(qry, "DROP TABLE ") {
				continue
			}
			b.Fatal(errors.Wrap(err, qry))
		}
	}

	qry := `BEGIN tst_bench_inout.P_BULK_INSERT_IMP (:1, :2, :3, :4, :5, :6, :7, :8, :9); END;`

	pt1 := time.Now()
	n := 512
	dates := make([]time.Time, n)
	keys := make([]string, n)
	ips := make([]int, n)
	zones := make([]int, n)
	plans := make([]int, n)
	banners := make([]int, n)
	referrers := make([]string, n)
	countries := make([]string, n)
	regions := make([]string, n)
	for i := range dates {
		dates[i] = pt1.Add(time.Duration(i) * time.Second)
		keys[i] = "key"
		ips[i] = 123456
		zones[i] = i % 256
		plans[i] = (i / 2) % 1000
		banners[i] = (i * 3) % 1000
		referrers[i] = "referrer"
		countries[i] = "country"
		regions[i] = "region"
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx = goracle.ContextWithLog(ctx, nil)
	tx, err := testDb.BeginTx(ctx, nil)
	if err != nil {
		b.Fatal(err)
	}
	defer tx.Rollback()

	params := []interface{}{
		goracle.PlSQLArrays,
		sql.Out{Dest: &dates, In: true},
		sql.Out{Dest: &keys, In: true},
		sql.Out{Dest: &ips, In: true},
		sql.Out{Dest: &zones, In: true},
		sql.Out{Dest: &plans, In: true},
		sql.Out{Dest: &banners, In: true},
		sql.Out{Dest: &referrers, In: true},
		sql.Out{Dest: &countries, In: true},
		sql.Out{Dest: &regions, In: true},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i += n {
		if _, err := tx.ExecContext(ctx, qry, params...); err != nil {
			if strings.Contains(err.Error(), "PLS-00905") || strings.Contains(err.Error(), "ORA-06508") {
				b.Log(goracle.GetCompileErrors(testDb, false))
			}
			//b.Log(dates, keys, ips, zones, plans, banners, referrers, countries, regions)
			b.Fatal(err)
		}
	}
	b.StopTimer()
}

func shortenFloat(s string) string {
	i := strings.IndexByte(s, '.')
	if i < 0 {
		return s
	}
	for j := i + 1; j < len(s); j++ {
		if s[j] != '0' {
			return s
		}
	}
	return s[:i]
}

const bFloat = 12345.6789

func BenchmarkSprintfFloat(b *testing.B) {
	var length int64
	for i := 0; i < b.N; i++ {
		s := fmt.Sprintf("%f", bFloat)
		s = shortenFloat(s)
		length += int64(len(s))
	}
	b.Logf("total length: %d", length)
}

/*
func BenchmarkAppendFloat(b *testing.B) {
	var length int64
	for i := 0; i < b.N; i++ {
		s := printFloat(bFloat)
		length += int64(len(s))
	}
}
*/

func createGeoTable(tableName string, rowCount int) error {
	var cnt int64
	if err := testDb.QueryRow(
		"SELECT COUNT(0) FROM " + tableName, //nolint:gas
	).Scan(&cnt); err == nil && cnt == int64(rowCount) {
		return nil
	}
	testDb.Exec("ALTER SESSION SET NLS_NUMERIC_CHARACTERS = '.,'")
	testDb.Exec("DROP TABLE " + tableName)
	if _, err := testDb.Exec(`CREATE TABLE ` + tableName + ` (` + //nolint:gas
		` id NUMBER(9) NOT NULL,
	"RECORD_ID" NUMBER(*,0) NOT NULL ENABLE,
	"PERSON_ID" NUMBER(*,0),
	"PERSON_ACCOUNT_ID" NUMBER(*,0),
	"ORGANIZATION_ID" NUMBER(*,0),
	"ORGANIZATION_MEMBERSHIP_ID" NVARCHAR2(45),
	"LOCATION" NVARCHAR2(2000) NOT NULL ENABLE,
	"DEVICE_ID" NVARCHAR2(45),
	"DEVICE_REGISTRATION_ID" NVARCHAR2(500),
	"DEVICE_NAME" NVARCHAR2(45),
	"DEVICE_TYPE" NVARCHAR2(45),
	"DEVICE_OS_NAME" NVARCHAR2(45),
	"DEVICE_TOKEN" NVARCHAR2(45),
	"DEVICE_OTHER_DETAILS" NVARCHAR2(100)
	)`,
	); err != nil {
		return err
	}
	testData := [][]string{
		{"1", "8.37064876162908E16", "8.37064898728264E16", "12", "6506", "POINT(30.5518407 104.0685472)", "a71223186cef459b", "", "Samsung SCH-I545", "Mobile", "Android 4.4.2", "", ""},
		{"2", "8.37064876162908E16", "8.37064898728264E16", "12", "6506", "POINT(30.5520498 104.0686355)", "a71223186cef459b", "", "Samsung SCH-I545", "Mobile", "Android 4.4.2", "", ""},
		{"3", "8.37064876162908E16", "8.37064898728264E16", "12", "6506", "POINT(30.5517747 104.0684895)", "a71223186cef459b", "", "Samsung SCH-I545", "Mobile", "Android 4.4.2", "", ""},
		{"4", "8.64522675633357E16", "8.64522734353613E16", "", "1220457", "POINT(30.55187 104.06856)", "3A9D1838-3B2D-4119-9E07-77C6CDAC53C5", "noUwBnWojdY:APA91bE8aGLEECS9_Q1EKrp8i2B36H1X8GwIj3v58KUcuXglhf0rXJb8Ez5meQ6D5MgTAQghYEe3s9vOntU3pYPQoc6ASNw3QzhzQevAqlMQC2ukUMNyLD8Rve-IA1-6lttsCXYsYIKh", "User3’s iPhone", "iPhone", "iPhone OS", "", "DeviceID:3A9D1838-3B2D-4119-9E07-77C6CDAC53C5, SystemVersion:8.4, LocalizedModel:iPhone"},
		{"5", "8.37064876162908E16", "8.37064898728264E16", "12", "6506", "POINT(30.5517458 104.0685809)", "a71223186cef459b", "", "Samsung SCH-I545", "Mobile", "Android 4.4.2", "", ""},
		{"6", "8.37064876162908E16", "8.37064898728264E16", "12", "6506", "POINT(30.551802 104.0685301)", "a71223186cef459b", "", "Samsung SCH-I545", "Mobile", "Android 4.4.2", "", ""},
		{"7", "8.64522675633357E16", "8.64522734353613E16", "", "1220457", "POINT(30.55187 104.06856)", "3A9D1838-3B2D-4119-9E07-77C6CDAC53C5", "noUwBnWojdY:APA91bE8aGLEECS9_Q1EKrp8i2B36H1X8GwIj3v58KUcuXglhf0rXJb8Ez5meQ6D5MgTAQghYEe3s9vOnt,3pYPQoc6ASNw3QzhzQevAqlMQC2ukUMNyLD8Rve-IA1-6lttsCXYsYIKh", "User3’s iPhone", "iPhone", "iPhone OS", "", "DeviceID:3A9D1838-3B2D-4119-9E07-77C6CDAC53C5, SystemVersion:8.4, LocalizedModel:iPhone"},
		{"8", "8.37064876162908E16", "8.37064898728264E16", "12", "6506", "POINT(30.551952 104.0685893)", "a71223186cef459b", "", "Samsung SCH-I545", "Mobile", "Android 4.4.2", "", ""},
		{"9", "8.37064876162908E16", "8.37064898728264E16", "12", "6506", "POINT(30.5518439 104.0685473)", "a71223186cef459b", "", "Samsung SCH-I545", "Mobile", "Android 4.4.2", "", ""},
		{"10", "8.37064876162908E16", "8.37064898728264E16", "12", "6506", "POINT(30.5518439 104.0685473)", "a71223186cef459b", "", "Samsung SCH-I545", "Mobile", "Android 4.4.2", "", ""},
	}
	cols := make([]interface{}, len(testData[0])+1)
	for i := range cols {
		cols[i] = make([]string, rowCount)
	}
	for i := 0; i < rowCount; i++ {
		row := testData[i%len(testData)]
		for j, col := range cols {
			if j == 0 {
				(col.([]string))[i] = strconv.Itoa(i)
			} else {
				(col.([]string))[i] = row[j-1]
			}
		}
	}

	stmt, err := testDb.Prepare("INSERT INTO " + tableName + //nolint:gas
		` (ID,RECORD_ID,PERSON_ID,PERSON_ACCOUNT_ID,ORGANIZATION_ID,ORGANIZATION_MEMBERSHIP_ID,
   LOCATION,DEVICE_ID,DEVICE_REGISTRATION_ID,DEVICE_NAME,DEVICE_TYPE,
   DEVICE_OS_NAME,DEVICE_TOKEN,DEVICE_OTHER_DETAILS)
   VALUES (:1,:2,:3,:4,:5,
           :6,:7,:8,:9,:10,
		   :11,:12, :13, :14)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	if _, err := stmt.Exec(cols...); err != nil {
		return fmt.Errorf("%v\n%q", err, cols)
	}
	return nil
}

func TestSelectOrder(t *testing.T) {
	t.Parallel()
	const limit = 1013
	var cnt int64
	tbl := "user_objects"
	start := time.Now()
	if err := testDb.QueryRow(
		"SELECT count(0) FROM " + tbl, //nolint:gas
	).Scan(&cnt); err != nil {
		t.Fatal(err)
	}
	t.Logf("%s rowcount=%d (%s)", tbl, cnt, time.Since(start))
	if cnt == 0 {
		cnt = 10
		tbl = "(SELECT 1 FROM DUAL " + strings.Repeat("\nUNION ALL SELECT 1 FROM DUAL ", int(cnt)-1) + ")" //nolint:gas
	}
	qry := "SELECT ROWNUM FROM " + tbl //nolint:gas
	for i := cnt; i < limit; i *= cnt {
		qry += ", " + tbl
	}
	t.Logf("qry=%s", qry)
	rows, err := testDb.Query(qry)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	i := 0
	for rows.Next() {
		var rn int
		if err = rows.Scan(&rn); err != nil {
			t.Fatal(err)
		}
		i++
		if rn != i {
			t.Errorf("got %d, wanted %d.", rn, i)
		}
		if i > limit {
			break
		}
	}
	for rows.Next() {
	}
}

// go test -c && ./goracle.v2.test -test.run=^$ -test.bench=Date -test.cpuprofile=/tmp/cpu.prof && go tool pprof goracle.v2.test /tmp/cpu.prof
func BenchmarkSelectDate(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; {
		b.StopTimer()
		rows, err := testDb.Query(`SELECT CAST(TO_DATE('2006-01-02 15:04:05', 'YYYY-MM-DD HH24:MI:SS') AS DATE) dt
		FROM
		(select 1 from dual union all select 1 from dual union all select 1 from dual union all select 1 from dual union all select 1 from dual union all select 1 from dual union all select 1 from dual union all select 1 from dual union all select 1 from dual union all select 1 from dual),
		(select 1 from dual union all select 1 from dual union all select 1 from dual union all select 1 from dual union all select 1 from dual union all select 1 from dual union all select 1 from dual union all select 1 from dual union all select 1 from dual union all select 1 from dual)
		`)
		if err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
		for rows.Next() && i < b.N {
			var dt time.Time
			if err = rows.Scan(&dt); err != nil {
				rows.Close()
				b.Fatal(err)
			}
			i++
		}
		b.StopTimer()
		rows.Close()
	}
}

func BenchmarkSelect(b *testing.B) {
	geoTableName := "test_geo" + tblSuffix
	const geoTableRowCount = 100000
	if err := createGeoTable(geoTableName, geoTableRowCount); err != nil {
		b.Fatal(err)
	}
	defer testDb.Exec("DROP TABLE " + geoTableName)

	for _, i := range []int{1, 10, 100, 1000} {
		b.Run(fmt.Sprintf("Prefetch%d", i), func(b *testing.B) { benchSelect(b, geoTableName, i) })
	}
}

func benchSelect(b *testing.B, geoTableName string, prefetchLen int) {
	b.ResetTimer()
	for i := 0; i < b.N; {
		b.StopTimer()
		rows, err := testDb.Query(
			"SELECT location FROM "+geoTableName, //nolint:gas
			goracle.FetchRowCount(prefetchLen))
		if err != nil {
			b.Fatal(err)
		}
		var readBytes, recNo int64
		b.StartTimer()
		for rows.Next() && i < b.N {
			var loc string
			if err = rows.Scan(&loc); err != nil {
				rows.Close()
				b.Fatal(err)
			}
			i++
			readBytes += int64(len(loc))
			recNo++
		}
		b.StopTimer()
		b.SetBytes(readBytes / recNo)
		rows.Close()
	}
}
