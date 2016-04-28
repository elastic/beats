package kafka

import "github.com/elastic/beats/libbeat/logp"

type kafkaLogger struct{}

func (kafkaLogger) Print(v ...interface{}) {
	logp.Warn("kafka message: %v", v...)
}

func (kafkaLogger) Printf(format string, v ...interface{}) {
	logp.Warn(format, v...)
}

func (kafkaLogger) Println(v ...interface{}) {
	logp.Warn("kafka message: %v", v...)
}
