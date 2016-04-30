package mysql

import (
	"os"
)

// Helper functions for testing used in the mysql metricsets

// GetMySQLEnvDSN returns the MySQL server DSN to use for testing. It
// reads the value from the MYSQL_DSN environment variable and returns
// a DSN for 127.0.0.1 if it is not set.
func GetMySQLEnvDSN() string {
	dsn := os.Getenv("MYSQL_DSN")

	if len(dsn) == 0 {
		dsn = CreateDSN("tcp(127.0.0.1:3306)/", "root", "")
	}
	return dsn
}
