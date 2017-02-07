package couchbase

import "os"

func GetEnvDSN() string {
	dsn := os.Getenv("COUCHBASE_DSN")

	if len(dsn) == 0 {
		dsn = "http://Administrator:password@localhost:8091"
	}
	return dsn
}
