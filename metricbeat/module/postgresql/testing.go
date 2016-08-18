package postgresql

import "os"

func GetEnvDSN() string {
	return os.Getenv("POSTGRESQL_DSN")
}
