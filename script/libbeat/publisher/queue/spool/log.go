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

package spool

import (
	"fmt"
	"sync"

	"github.com/elastic/beats/libbeat/logp"
)

type logger interface {
	Debug(...interface{})
	Debugf(string, ...interface{})

	Info(...interface{})
	Infof(string, ...interface{})

	Error(...interface{})
	Errorf(string, ...interface{})
}

var _defaultLogger struct {
	singleton logger
	init      sync.Once
}

func defaultLogger() logger {
	_defaultLogger.init.Do(func() {
		_defaultLogger.singleton = logp.NewLogger("spool")
	})
	return _defaultLogger.singleton
}

// func defaultLogger() logger { return (*outLogger)(nil) }

type outLogger struct{}

func (l *outLogger) Debug(vs ...interface{})              { l.report("Debug", vs) }
func (l *outLogger) Debugf(fmt string, vs ...interface{}) { l.reportf("Debug: ", fmt, vs) }

func (l *outLogger) Info(vs ...interface{})              { l.report("Info", vs) }
func (l *outLogger) Infof(fmt string, vs ...interface{}) { l.reportf("Info", fmt, vs) }

func (l *outLogger) Error(vs ...interface{})              { l.report("Error", vs) }
func (l *outLogger) Errorf(fmt string, vs ...interface{}) { l.reportf("Error", fmt, vs) }

func (l *outLogger) report(level string, vs []interface{}) {
	args := append([]interface{}{level, ":"}, vs...)
	fmt.Println(args...)
}

func (*outLogger) reportf(level string, str string, vs []interface{}) {
	str = level + ": " + str
	fmt.Printf(str, vs...)
}
