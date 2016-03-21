/*

Helper functions for testing used in the mysql metricsets

*/
package mysql

import (
	"os"
)

func GetMySQLEnvDSN() string {
	dsn := os.Getenv("MYSQL_DSN")

	if len(dsn) == 0 {
		dsn = CreateDSN("tcp(127.0.0.1:3306)/", "root", "")
	}
	return dsn
}
