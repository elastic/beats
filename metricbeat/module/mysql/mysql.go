package mysql

import (
	"os"

	"github.com/elastic/beats/metricbeat/helper"

	"database/sql"
	_ "github.com/go-sql-driver/mysql"
)

func init() {
	helper.Registry.AddModuler("mysql", Moduler{})
}

type Moduler struct{}

func (b Moduler) Setup() error {
	// TODO: Ping available servers to check if available
	return nil
}

// Connect expects a full mysql dsn
// Example: [username[:password]@][protocol[(address)]]/
func Connect(dsn string) (*sql.DB, error) {
	return sql.Open("mysql", dsn)
}

///*** Testing helpers ***///

func GetMySQLEnvDSN() string {
	dsn := os.Getenv("MYSQL_DSN")

	if len(dsn) == 0 {
		dsn = "root@tcp(127.0.0.1:3306)/"
	}
	return dsn
}
