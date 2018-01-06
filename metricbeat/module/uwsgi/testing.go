package uwsgi

import "os"

// GetEnvTCPServer returns uwsgi stat server host with tcp mode
func GetEnvTCPServer() string {
	env := os.Getenv("UWSGI_STAT_TCP_SERVER")
	if len(env) == 0 {
		env = "tcp://127.0.0.1:9191"
	}
	return env
}

// GetEnvHTTPServer returns uwsgi stat server host with http mode
func GetEnvHTTPServer() string {
	env := os.Getenv("UWSGI_STAT_HTTP_SERVER")
	if len(env) == 0 {
		env = "http://127.0.0.1:9192"
	}
	return env
}
