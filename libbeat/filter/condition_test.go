package filter

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/stretchr/testify/assert"
)

func TestEqualsCondition(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"*"})
	}

	config1 := ConditionConfig{
		Equals: map[string]string{
			"type": "process",
		},
	}
	cond1, err := NewCondition(config1)
	assert.True(t, err == nil)

	config2 := ConditionConfig{
		Equals: map[string]string{
			"proc.pid": "305",
		},
	}
	cond2, err := NewCondition(config2)
	assert.True(t, err == nil)

	config3 := ConditionConfig{
		Equals: map[string]string{
			"proc.pid": "0.08",
		},
	}
	cond3, err := NewCondition(config3)
	assert.True(t, err == nil)

	config4 := ConditionConfig{
		Equals: map[string]string{
			"proc.cpu.total_p": "0.08",
		},
	}
	cond4, err := NewCondition(config4)
	assert.True(t, err == nil)

	event := common.MapStr{
		"@timestamp": "2016-04-14T20:41:06.258Z",
		"proc": common.MapStr{
			"cmdline": "/usr/libexec/secd",
			"cpu": common.MapStr{
				"start_time": "Apr10",
				"system":     1988,
				"total":      6029,
				"total_p":    0.08,
				"user":       4041,
			},
			"name":     "secd",
			"pid":      305,
			"ppid":     1,
			"state":    "running",
			"username": "monica",
		},
		"type": "process",
	}

	assert.True(t, cond1.Check(event))
	assert.True(t, cond2.Check(event))
	assert.False(t, cond3.Check(event))
	assert.False(t, cond4.Check(event))
}

func TestContainsCondition(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"*"})
	}

	config1 := ConditionConfig{
		Contains: map[string]string{
			"proc.name": "sec",
		},
	}
	cond1, err := NewCondition(config1)
	assert.True(t, err == nil)

	config2 := ConditionConfig{
		Contains: map[string]string{
			"proc.name": "secddd",
		},
	}
	cond2, err := NewCondition(config2)
	assert.True(t, err == nil)

	event := common.MapStr{
		"@timestamp": "2016-04-14T20:41:06.258Z",
		"proc": common.MapStr{
			"cmdline": "/usr/libexec/secd",
			"cpu": common.MapStr{
				"start_time": "Apr10",
				"system":     1988,
				"total":      6029,
				"total_p":    0.08,
				"user":       4041,
			},
			"name":     "secd",
			"pid":      305,
			"ppid":     1,
			"state":    "running",
			"username": "monica",
		},
		"type": "process",
	}

	assert.True(t, cond1.Check(event))
	assert.False(t, cond2.Check(event))
}

func TestRegexpCondition(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"*"})
	}

	// first simple condition
	config1 := ConditionConfig{
		Regexp: map[string]string{
			"source": "apache2/error.*",
		},
	}
	cond1, err := NewCondition(config1)
	assert.True(t, err == nil)

	// second simple condition
	config2 := ConditionConfig{
		Regexp: map[string]string{
			"source": "apache2/access.*",
		},
	}
	cond2, err := NewCondition(config2)
	assert.True(t, err == nil)

	// third complex condition
	config3 := ConditionConfig{
		Regexp: map[string]string{
			"source":  "apache2/error.*",
			"message": "[client 1.2.3.4]",
		},
	}
	cond3, err := NewCondition(config3)
	assert.True(t, err == nil)

	event := common.MapStr{
		"@timestamp": "2016-04-14T20:41:06.258Z",
		"message":    `[Fri Dec 16 01:46:23 2005] [error] [client 1.2.3.4] Directory index forbidden by rule: /home/test/`,
		"source":     "/var/log/apache2/error.log",
		"type":       "log",
		"input_type": "log",
		"offset":     30,
	}

	event1 := common.MapStr{
		"@timestamp": "2016-04-14T20:41:06.258Z",
		"message":    `127.0.0.1 - - [28/Jul/2006:10:27:32 -0300] "GET /hidden/ HTTP/1.0" 404 7218`,
		"source":     "/var/log/apache2/access.log",
		"type":       "log",
		"input_type": "log",
		"offset":     30,
	}

	assert.True(t, cond1.Check(event))
	assert.False(t, cond2.Check(event))
	assert.True(t, cond3.Check(event))

	assert.False(t, cond1.Check(event1))
	assert.True(t, cond2.Check(event1))
}

func TestRangeCondition(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"*"})
	}

	// first condition
	var v400 float64 = 400
	var v500 float64 = 500
	config1 := ConditionConfig{
		Range: map[string]RangeValue{
			"http.code": RangeValue{Gte: &v400, Lt: &v500},
		},
	}
	cond1, err := NewCondition(config1)
	assert.True(t, err == nil)

	// second condition
	var v2800 float64 = 28000
	config2 := ConditionConfig{
		Range: map[string]RangeValue{
			"bytes_out": RangeValue{Gte: &v2800},
		},
	}
	cond2, err := NewCondition(config2)
	assert.True(t, err == nil)

	// complex condition
	var v30 float64 = 30
	config3 := ConditionConfig{
		Range: map[string]RangeValue{
			"bytes_out":    RangeValue{Gte: &v2800},
			"responsetime": RangeValue{Gt: &v30},
		},
	}
	cond3, err := NewCondition(config3)
	assert.True(t, err == nil)

	// float condition
	var v05 float64 = 0.5
	config4 := ConditionConfig{
		Range: map[string]RangeValue{
			"proc.cpu.total_p": RangeValue{Gte: &v05},
		},
	}
	cond4, err := NewCondition(config4)
	assert.True(t, err == nil)

	event := common.MapStr{
		"@timestamp":    "2015-06-11T09:51:23.642Z",
		"bytes_in":      126,
		"bytes_out":     28033,
		"client_ip":     "127.0.0.1",
		"client_port":   42840,
		"client_proc":   "",
		"client_server": "mar.local",
		"http": common.MapStr{
			"code":           404,
			"content_length": 76985,
			"phrase":         "Not found",
		},
		"ip":           "127.0.0.1",
		"method":       "GET",
		"params":       "",
		"path":         "/jszip.min.js",
		"port":         8000,
		"proc":         "",
		"query":        "GET /jszip.min.js",
		"responsetime": 30,
		"server":       "mar.local",
		"status":       "OK",
		"type":         "http",
	}

	event1 := common.MapStr{
		"@timestamp": "2016-04-20T07:46:44.633Z",
		"proc": common.MapStr{
			"cmdline": "/System/Library/Frameworks/CoreServices.framework/Frameworks/Metadata.framework/Versions/A/Support/mdworker -s mdworker -c MDSImporterWorker -m com.apple.mdworker.single",
			"cpu": common.MapStr{
				"start_time": "09:19",
				"system":     22,
				"total":      66,
				"total_p":    0.6,
				"user":       44,
			},
			"name":     "mdworker",
			"pid":      44978,
			"ppid":     1,
			"state":    "running",
			"username": "test",
		},
		"type": "process",
	}

	assert.True(t, cond1.Check(event))
	assert.True(t, cond2.Check(event))
	assert.False(t, cond3.Check(event))
	assert.True(t, cond4.Check(event1))
	assert.False(t, cond4.Check(event))
}
