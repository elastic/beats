package logstash

import "github.com/elastic/beats/libbeat/logp"

type logstashLogger struct{}

func (logstashLogger) Print(v ...interface{}) {
	logp.Info("logstash message: %v", v...)
}

func (logstashLogger) Printf(format string, v ...interface{}) {
	logp.Info(format, v...)
}

func (logstashLogger) Println(v ...interface{}) {
	logp.Info("logstash message: %v", v...)
}
