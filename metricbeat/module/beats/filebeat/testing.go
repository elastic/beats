package filebeat

import "os"

func GetEnvHost() string {
	host := os.Getenv("FILEBEAT_HOST")

	if len(host) == 0 {
		host = "127.0.0.1"
	}
	return host
}

func GetEnvPort() string {
	port := os.Getenv("FILEBEAT_PORT")

	if len(port) == 0 {
		port = "6060"
	}
	return port
}
