package uwsgi

import "os"

func GetEnvTCPServer() string {
	env := os.Getenv("UWSGI_STAT_TCP_SERVER")
	if len(env) == 0 {
		env = "tcp://127.0.0.1:9191"
	}
	return env
}

func GetEnvHTTPServer() string {
	env := os.Getenv("UWSGI_STAT_TCP_SERVER")
	if len(env) == 0 {
		env = "http://127.0.0.1:9192"
	}
	return env
}
