package mysql

import (
	"database/sql"
	"os"

	"github.com/elastic/beats/metricbeat/helper"

	_ "github.com/go-sql-driver/mysql"
	"github.com/urso/ucfg"
)

func init() {
	helper.Registry.AddModuler("mysql", Moduler{})
}

type Moduler struct{}

func (b Moduler) Setup(cfg *ucfg.Config) error {
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
