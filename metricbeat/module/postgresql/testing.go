package postgresql

import "os"

func GetEnvDSN() string {
	return os.Getenv("POSTGRESQL_DSN")
}

func GetEnvUsername() string {
	return os.Getenv("POSTGRESQL_USERNAME")
}

func GetEnvPassword() string {
	return os.Getenv("POSTGRESQL_PASSWORD")
}
