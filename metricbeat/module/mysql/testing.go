package mysql

import (
	"os"

	"github.com/go-sql-driver/mysql"
)

// Helper functions for testing used in the mysql MetricSets.

// GetMySQLEnvDSN returns the MySQL server DSN to use for testing. It
// reads the value from the MYSQL_DSN environment variable and returns
// root@tcp(127.0.0.1:3306)/ if it is not set.
func GetMySQLEnvDSN() string {
	dsn := os.Getenv("MYSQL_DSN")

	if len(dsn) == 0 {
		c := &mysql.Config{
			Net:  "tcp",
			Addr: "127.0.0.1:3306",
			User: "root",
		}
		dsn = c.FormatDSN()
	}
	return dsn
}
