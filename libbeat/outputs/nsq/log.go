package nsq

import (
	"github.com/Shopify/sarama"
	"github.com/elastic/beats/v7/libbeat/logp"
)

type nsqLogger struct {
	log *logp.Logger
}

func (nsqL nsqLogger) Print(v ...interface{}) {
	nsqL.Log("nsq message: %v", v...)
}

func (nsqL nsqLogger) Printf(format string, v ...interface{}) {
	nsqL.Log(format, v...)
}

func (nsqL nsqLogger) Println(v ...interface{}) {
	nsqL.Log("nsq mssage: %v", v...)
}

func (nsqL nsqLogger) Log(format string, v ...interface{}) {
	warn := false
	for _, val := range v {
		if err, ok := val.(sarama.KError); ok {
			if err != sarama.ErrNoError {
				warn = true
				break
			}
		}
	}

	if nsqL.log == nil {
		nsqL.log = logp.NewLogger(logSelector)
	}

	if warn {
		nsqL.log.Warnf(format, v...)
	} else {
		nsqL.log.Infof(format, v...)
	}
}
