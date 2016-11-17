/*
Package postgresql is Metricbeat module for PostgreSQL server.
*/
package postgresql

import (
	"database/sql"
	"fmt"
	"net"
	nurl "net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/pkg/errors"
)

func QueryStats(db *sql.DB, query string) ([]map[string]interface{}, error) {

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}

	columns, err := rows.Columns()
	if err != nil {
		return nil, errors.Wrap(err, "scanning columns")
	}
	vals := make([][]byte, len(columns))
	valPointers := make([]interface{}, len(columns))
	for i := range vals {
		valPointers[i] = &vals[i]
	}

	results := []map[string]interface{}{}

	for rows.Next() {
		err = rows.Scan(valPointers...)
		if err != nil {
			return nil, errors.Wrap(err, "scanning row")
		}

		result := map[string]interface{}{}
		for i, col := range columns {
			result[col] = string(vals[i])
		}

		logp.Debug("postgresql", "Result: %v", result)
		results = append(results, result)
	}
	return results, nil
}

// ParseURL parses the given URL and overrides the values of username, password and timeout
// if given. Returns a connection string in the form of `user=pass` ready to be passed to the
// sql.Open call.
// Code adapted from the pg driver: https://github.com/lib/pq/blob/master/url.go#L32
func ParseURL(url, username, password string, timeout time.Duration) (string, error) {
	u, err := nurl.Parse(url)
	if err != nil {
		return "", err
	}

	if u.Scheme != "postgres" && u.Scheme != "postgresql" {
		return "", fmt.Errorf("invalid connection protocol: %s", u.Scheme)
	}

	var kvs []string
	escaper := strings.NewReplacer(` `, `\ `, `'`, `\'`, `\`, `\\`)
	accrue := func(k, v string) {
		if v != "" {
			kvs = append(kvs, k+"="+escaper.Replace(v))
		}
	}

	if len(username) > 0 {
		accrue("user", username)
		accrue("password", password)
	} else {
		if u.User != nil {
			v := u.User.Username()
			accrue("user", v)

			v, _ = u.User.Password()
			accrue("password", v)
		}
	}

	if host, port, err := net.SplitHostPort(u.Host); err != nil {
		accrue("host", u.Host)
	} else {
		accrue("host", host)
		accrue("port", port)
	}

	if u.Path != "" {
		accrue("dbname", u.Path[1:])
	}

	q := u.Query()
	for k := range q {
		if k == "connect_timeout" && timeout != 0 {
			continue
		}
		accrue(k, q.Get(k))
	}
	if timeout != 0 {
		accrue("connect_timeout", strconv.Itoa(int(timeout.Seconds())))
	}

	sort.Strings(kvs) // Makes testing easier (not a performance concern)
	return strings.Join(kvs, " "), nil
}
