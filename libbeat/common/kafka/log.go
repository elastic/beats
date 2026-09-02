// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package kafka

import (
	"sync"
	"sync/atomic"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/sarama"
)

const (
	logSelector = "kafka"
)

var (
	saramaAdapter     kafkaLogger
	saramaAdapterOnce sync.Once
)

// SetSaramaLogger installs a process-wide adapter on sarama.Logger.
//
// Sarama exposes a single global logger, so the adapter is installed once and
// later calls only swap the inner logp logger. That avoids races when the
// Kafka input and output both configure it.
func SetSaramaLogger(log *logp.Logger) {
	if log == nil {
		log = logp.NewLogger(logSelector)
	}
	saramaAdapter.log.Store(log)
	saramaAdapterOnce.Do(func() {
		sarama.Logger = &saramaAdapter
	})
}

type kafkaLogger struct {
	log atomic.Pointer[logp.Logger]
}

func (kl *kafkaLogger) Print(v ...any) {
	kl.Log("kafka message: %v", v...)
}

func (kl *kafkaLogger) Printf(format string, v ...any) {
	kl.Log(format, v...)
}

func (kl *kafkaLogger) Println(v ...any) {
	kl.Log("kafka message: %v", v...)
}

func (kl *kafkaLogger) Log(format string, v ...any) {
	warn := false
	for _, val := range v {
		if err, ok := val.(sarama.KError); ok {
			if err != sarama.ErrNoError {
				warn = true
				break
			}
		}
	}
	log := kl.log.Load()
	if log == nil {
		log = logp.NewLogger(logSelector)
		kl.log.Store(log)
	}
	if warn {
		log.Warnf(format, v...)
	} else {
		log.Infof(format, v...)
	}
}
