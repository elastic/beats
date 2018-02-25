package kafka

import (
	"github.com/Shopify/sarama"

	"github.com/elastic/beats/libbeat/logp"
)

type kafkaLogger struct{}

func (kl kafkaLogger) Print(v ...interface{}) {
	kl.Log("kafka message: %v", v...)
}

func (kl kafkaLogger) Printf(format string, v ...interface{}) {
	kl.Log(format, v...)
}

func (kl kafkaLogger) Println(v ...interface{}) {
	kl.Log("kafka message: %v", v...)
}

func (kafkaLogger) Log(format string, v ...interface{}) {
	warn := false
	for _, val := range v {
		if err, ok := val.(sarama.KError); ok {
			if err != sarama.ErrNoError {
				warn = true
				break
			}
		}
	}
	if warn {
		logp.Warn(format, v...)
	} else {
		logp.Info(format, v...)
	}
}
