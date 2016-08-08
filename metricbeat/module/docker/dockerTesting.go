package docker

import (
	"os"
)

func GetDockerSocketFromEnv() string {
	socket := os.Getenv("DOCKER_SOCKET")
	if len(socket) == 0 {
		socket = "unix:///var/run/docker.sock"
	}
	return socket
}
func GetRedisEnvHost() string {
	host := os.Getenv("DOCKER_HOST")

	if len(host) == 0 {
		host = "127.0.0.1"
	}
	return host
}
