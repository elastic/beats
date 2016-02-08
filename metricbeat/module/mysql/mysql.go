package mysql

import (
	"github.com/elastic/beats/metricbeat/helper"

	"database/sql"
	_ "github.com/go-sql-driver/mysql"

	"os"
)

func init() {
	Module.Register()
}

// Module object
var Module = helper.NewModule("mysql", Mysql{})

type Mysql struct {
}

func (b Mysql) Setup() error {
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
