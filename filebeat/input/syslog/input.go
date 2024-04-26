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

package syslog

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/filebeat/channel"
	"github.com/elastic/beats/filebeat/harvester"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/filebeat/inputsource"
	"github.com/elastic/beats/filebeat/util"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
)

// Parser is generated from a ragel state machine using the following command:
//go:generate ragel -Z -G2 parser.rl -o parser.go

// Severity and Facility are derived from the priority, theses are the human readable terms
// defined in https://tools.ietf.org/html/rfc3164#section-4.1.1.
//
// Example:
// 2 => "Critical"
type mapper []string

var (
	severityLabels = mapper{
		"Emergency",
		"Alert",
		"Critical",
		"Error",
		"Warning",
		"Notice",
		"Informational",
		"Debug",
	}

	facilityLabels = mapper{
		"kernel",
		"user-level",
		"mail",
		"system",
		"security/authorization",
		"syslogd",
		"line printer",
		"network news",
		"UUCP",
		"clock",
		"security/authorization",
		"FTP",
		"NTP",
		"log audit",
		"log alert",
		"clock",
		"local0",
		"local1",
		"local2",
		"local3",
		"local4",
		"local5",
		"local6",
		"local7",
	}
)

func init() {
	err := input.Register("syslog", NewInput)
	if err != nil {
		panic(err)
	}
}

// syslogFormatter 兼容syslog数据格式
func syslogFormatter(event beat.Event) beat.Event {
	message, _ := event.Fields.GetValue("message")
	event.Fields.Put("data", message)

	if _, ok := event.Fields["log"]; ok {
		address := event.Fields["log"].(common.MapStr)["source"].(common.MapStr)["address"].(string)
		arr := strings.Split(address, ":")
		if len(arr) > 1 {
			port, err := strconv.Atoi(arr[1])
			if err != nil {
				port = 0
			}
			event.Fields["log"].(common.MapStr)["source"].(common.MapStr)["address"] = arr[0]
			event.Fields["log"].(common.MapStr)["source"].(common.MapStr)["port"] = port
		} else {
			event.Fields["log"].(common.MapStr)["source"].(common.MapStr)["port"] = 0
		}
	} else {
		event.Fields["event"] = common.MapStr{}
		event.Fields["process"] = common.MapStr{}
		event.Fields["log"] = common.MapStr{}
		event.Fields["syslog"] = common.MapStr{}
	}

	err := event.Fields.Delete("message")
	log := logp.NewLogger("syslog")
	if err != nil {
		log.Errorw("key not found: %v", err)
	}

	return event
}

type SyslogFields struct {
	Data          interface{}
	Address       interface{}
	Port          interface{}
	Facility      interface{}
	FacilityLabel interface{}
	Priority      interface{}
	Severity      interface{}
	SeverityLabel interface{}
	Program       interface{}
	Pid           interface{}
}

func (s *SyslogFields) GetSyslogFieldValue(key string) interface{} {
	key = strings.ToLower(key)
	switch key {
	case "data", "":
		return s.Data
	case "address":
		return s.Address
	case "port":
		return s.Port
	case "facility":
		return s.Facility
	case "facility_label":
		return s.FacilityLabel
	case "priority":
		return s.Priority
	case "severity":
		return s.Severity
	case "severity_label":
		return s.SeverityLabel
	case "program":
		return s.Program
	case "pid":
		return s.Pid
	default:
		return ""
	}
}

func flattenMap(original common.MapStr, flatMap common.MapStr) {
	for key, value := range original {
		fullKey := key
		if subMap, ok := value.(common.MapStr); ok {
			flattenMap(subMap, flatMap)
		} else {
			flatMap[fullKey] = value
		}
	}
}

func parseSyslogField(data *util.Data) *SyslogFields {

	flatMap := make(common.MapStr)

	flattenMap(data.Event.Fields, flatMap)

	// 解析 syslog 字段值
	syslogFields := &SyslogFields{
		Data:          flatMap["data"],
		Address:       flatMap["address"],
		Port:          flatMap["port"],
		Facility:      flatMap["facility"],
		FacilityLabel: flatMap["facility_label"],
		Priority:      flatMap["priority"],
		Severity:      flatMap["severity"],
		SeverityLabel: flatMap["severity_label"],
		Program:       flatMap["program"],
		Pid:           flatMap["pid"],
	}
	return syslogFields
}

// Filter Syslog filter
func Filter(data *util.Data, config *config) bool {

	var text string
	var ok bool

	event := &data.Event
	text, ok = event.Fields["data"].(string)

	if !ok {
		return false
	}

	syslogFields := parseSyslogField(data)

	for _, filterConfig := range config.SyslogFilters {
		access := true
		for _, condition := range filterConfig.Conditions {

			matcher := condition.GetMatcher()
			if matcher == nil {
				access = false
				break
			}

			// syslog field value match
			fieldValue := syslogFields.GetSyslogFieldValue(condition.Key)
			switch v := fieldValue.(type) {
			case string:
				text = v
			case int:
				text = strconv.Itoa(v)
			default:
				text = ""
			}

			if !matcher(text) {
				access = false
				break
			} else {
				continue
			}
		}
		if access {
			return true
		}
	}
	return false
}

func InitFilterMatcher(config *config) (*config, error) {
	if config.SyslogFilters == nil {
		return config, nil
	}
	for _, f := range config.SyslogFilters {
		for i, condition := range f.Conditions {
			// 去除字符串首尾空白字符
			condition.Key = strings.TrimSpace(condition.Key)
			condition.Value = strings.TrimSpace(condition.Value)

			// 初始化条件匹配方法 Matcher
			matcher, err := getOperationFunc(condition.Op, condition.Value)

			if err != nil {
				return nil, err
			}

			condition.matcher = matcher

			// 重新赋值 condition
			f.Conditions[i] = condition
		}
	}
	return config, nil
}

// Input define a syslog input
type Input struct {
	sync.Mutex
	started bool
	outlet  channel.Outleter
	server  inputsource.Network
	config  *config
	log     *logp.Logger
}

// NewInput creates a new syslog input
func NewInput(
	cfg *common.Config,
	outlet channel.Connector,
	context input.Context,
) (input.Input, error) {
	cfgwarn.Experimental("Syslog input type is used")

	log := logp.NewLogger("syslog")

	out, err := outlet(cfg, context.DynamicFields)
	if err != nil {
		return nil, err
	}

	config := defaultConfig
	if err = cfg.Unpack(&config); err != nil {
		return nil, err
	}

	conf, err := InitFilterMatcher(&config)
	if err != nil {
		return nil, err
	}
	config = *conf

	forwarder := harvester.NewForwarder(out)
	cb := func(data []byte, metadata inputsource.NetworkMetadata) {
		var ev beat.Event

		if IsRFC5424Format(data) {
			ev = parseAndCreateEvent5424(data, metadata, time.UTC, log)
		} else {
			ev = parseAndCreateEvent3164(data, metadata, time.UTC, log)
		}
		log.Error(ev)

		var d = &util.Data{Event: syslogFormatter(ev)}
		filterAccess := true
		if config.SyslogFilters != nil {
			filterAccess = Filter(d, &config)
		}
		if filterAccess {
			forwarder.Send(d)
		}
	}

	server, err := factory(cb, config.Protocol)
	if err != nil {
		return nil, err
	}

	return &Input{
		outlet:  out,
		started: false,
		server:  server,
		config:  &config,
		log:     log,
	}, nil
}

// Run starts listening for Syslog events over the network.
func (p *Input) Run() {
	p.Lock()
	defer p.Unlock()

	if !p.started {
		p.log.Infow("Starting Syslog input", "protocol", p.config.Protocol.Name())
		err := p.server.Start()
		if err != nil {
			p.log.Error("Error starting the server", "error", err)
			return
		}
		p.started = true
	}
}

// Reload runs the input
func (p *Input) Reload() {
	return
}

// Stop stops the syslog input.
func (p *Input) Stop() {
	defer p.outlet.Close()
	p.Lock()
	defer p.Unlock()

	if !p.started {
		return
	}

	p.log.Info("Stopping Syslog input")
	p.server.Stop()
	p.started = false
}

// Wait stops the syslog input.
func (p *Input) Wait() {
	p.Stop()
}

func createEvent(ev *event, metadata inputsource.NetworkMetadata, timezone *time.Location, log *logp.Logger) beat.Event {
	f := common.MapStr{
		"message": strings.TrimRight(ev.Message(), "\n"),
		"log": common.MapStr{
			"source": common.MapStr{
				"address": metadata.RemoteAddr.String(),
			},
		},
	}

	syslog := common.MapStr{}
	event := common.MapStr{}
	process := common.MapStr{}

	if ev.Hostname() != "" {
		f["hostname"] = ev.Hostname()
	}

	if ev.HasPid() {
		process["pid"] = ev.Pid()
	}

	if ev.Program() != "" {
		process["program"] = ev.Program()
	}

	if ev.HasPriority() {
		syslog["priority"] = ev.Priority()

		event["severity"] = ev.Severity()
		v, err := mapValueToName(ev.Severity(), severityLabels)
		if err != nil {
			log.Debugw("could not find severity label", "error", err)
		} else {
			syslog["severity_label"] = v
		}

		syslog["facility"] = ev.Facility()
		v, err = mapValueToName(ev.Facility(), facilityLabels)
		if err != nil {
			log.Debugw("could not find facility label", "error", err)
		} else {
			syslog["facility_label"] = v
		}
	}

	f["syslog"] = syslog
	f["event"] = event
	f["process"] = process

	return beat.Event{
		Timestamp: ev.Timestamp(timezone),
		Meta: common.MapStr{
			"truncated": metadata.Truncated,
		},
		Fields: f,
	}
}

func mapValueToName(v int, m mapper) (string, error) {
	if v < 0 || v >= len(m) {
		return "", errors.Errorf("value out of bound: %d", v)
	}
	return m[v], nil
}

func parseAndCreateEvent3164(data []byte, metadata inputsource.NetworkMetadata, timezone *time.Location, log *logp.Logger) beat.Event {
	ev := newEvent()
	ParserRFC3164(data, ev)
	if !ev.IsValid() {
		log.Errorw("can't parse event as syslog rfc3164", "message", string(data))
		return newBeatEvent(time.Now(), metadata, common.MapStr{
			"message": string(data),
		})
	}
	return createEvent(ev, metadata, timezone, log)
}

func parseAndCreateEvent5424(data []byte, metadata inputsource.NetworkMetadata, timezone *time.Location, log *logp.Logger) beat.Event {
	ev := newEvent()
	ParserRFC5424(data, ev)
	if !ev.IsValid() {
		log.Errorw("can't parse event as syslog rfc5424", "message", string(data))
		return newBeatEvent(time.Now(), metadata, common.MapStr{
			"message": string(data),
		})
	}
	return createEvent(ev, metadata, timezone, log)
}

func newBeatEvent(timestamp time.Time, metadata inputsource.NetworkMetadata, fields common.MapStr) beat.Event {
	event := beat.Event{
		Timestamp: timestamp,
		Meta: common.MapStr{
			"truncated": metadata.Truncated,
		},
		Fields: fields,
	}
	if metadata.RemoteAddr != nil {
		event.Fields.Put("log.source.address", metadata.RemoteAddr.String())
	}
	return event
}
