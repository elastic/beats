package couchbase

import "os"

func GetEnvDSN() string {
	return os.Getenv("COUCHBASE_DSN")
}
