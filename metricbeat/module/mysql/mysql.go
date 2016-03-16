package mysql

import (
	"database/sql"
	"os"

	"github.com/elastic/beats/metricbeat/helper"

	_ "github.com/go-sql-driver/mysql"
)

func init() {
	helper.Registry.AddModuler("mysql", New)
}

// New creates new instance of Moduler
func New() helper.Moduler {
	return &Moduler{}
}

type Moduler struct{}

func (m *Moduler) Setup(mo *helper.Module) error {
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
