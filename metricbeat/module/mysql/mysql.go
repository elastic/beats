package mysql

import (
	"database/sql"

	"github.com/elastic/beats/metricbeat/helper"

	_ "github.com/go-sql-driver/mysql"
)

func init() {
	if err := helper.Registry.AddModuler("mysql", New); err != nil {
		panic(err)
	}
}

// New creates new instance of Moduler
func New() helper.Moduler {
	return &Moduler{}
}

type Moduler struct{}

func (m *Moduler) Setup(mo *helper.Module) error {
	return nil
}

// CreateDSN creates a dsn string out of hostname, username and password
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

// Connect expects a full mysql dsn
// Example: [username[:password]@][protocol[(address)]]/
func Connect(dsn string) (*sql.DB, error) {
	return sql.Open("mysql", dsn)
}
