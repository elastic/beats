/*
Package mysql is Metricbeat module for MySQL server.
*/
package mysql

import (
	"database/sql"

	// Register the MySQL driver.
	_ "github.com/go-sql-driver/mysql"
)

// CreateDSN creates a DSN (data source name) string out of hostname, username,
// and password.
func CreateDSN(host string, username string, password string) string {
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
	return dsn
}

// Connect creates a new DB connection. It expects a full MySQL DSN.
// Example DSN: [username[:password]@][protocol[(address)]]/
func Connect(dsn string) (*sql.DB, error) {
	return sql.Open("mysql", dsn)
}
