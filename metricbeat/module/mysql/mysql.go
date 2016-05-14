/*
Package mysql is Metricbeat module for MySQL server.
*/
package mysql

import (
	"database/sql"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
)

// CreateDSN creates a DSN (data source name) string out of hostname, username,
// password, and timeout. It validates the resulting DSN and returns an error
// if the DSN is invalid.
func CreateDSN(host, username, password string, timeout time.Duration) (string, error) {
	// Example: [username[:password]@][protocol[(address)]]/
	dsn := host

	if username != "" || password != "" {
		dsn = "@" + dsn
	}

	if password != "" {
		dsn = ":" + password + dsn
	}

	if username != "" {
		dsn = username + dsn
	}

	config, err := mysql.ParseDSN(dsn)
	if err != nil {
		return "", errors.Wrapf(err, "config error for host '%s'", host)
	}

	if timeout > 0 {
		// Add connection timeouts to the DSN.
		config.Timeout = timeout
		config.ReadTimeout = timeout
		config.WriteTimeout = timeout
	}

	return config.FormatDSN(), nil
}

// NewDB returns a new mysql database handle. The dsn value (data source name)
// must be valid, otherwise an error will be returned.
//
// Example DSN: [username[:password]@][protocol[(address)]]/
func NewDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, errors.Wrap(err, "sql open failed")
	}
	return db, nil
}
