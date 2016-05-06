package filter

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/stretchr/testify/assert"
)

func TestIncludeFields(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"*"})
	}

	Filters := FilterList{}

	rule, err := NewIncludeFields(IncludeFieldsConfig{
		Fields: []string{"proc.cpu.total_p", "proc.mem", "dd"},
		ConditionConfig: ConditionConfig{Contains: map[string]string{
			"proc.name": "test",
		}},
	})
	assert.True(t, err == nil)

	Filters.Register(rule)

	event := common.MapStr{
		"@timestamp": "2016-01-24T18:35:19.308Z",
		"beat": common.MapStr{
			"hostname": "mar",
			"name":     "my-shipper-1",
		},
		"proc": common.MapStr{
			"cpu": common.MapStr{
				"start_time": "Jan14",
				"system":     26027,
				"total":      79390,
				"total_p":    0,
				"user":       53363,
			},
			"name":    "test-1",
			"cmdline": "/sbin/launchd",
			"mem": common.MapStr{
				"rss":   11194368,
				"rss_p": 0,
				"share": 0,
				"size":  2555572224,
			},
		},
		"type": "process",
	}

	filteredEvent := Filters.Filter(event)

	expectedEvent := common.MapStr{
		"@timestamp": "2016-01-24T18:35:19.308Z",
		"proc": common.MapStr{
			"cpu": common.MapStr{
				"total_p": 0,
			},
			"mem": common.MapStr{
				"rss":   11194368,
				"rss_p": 0,
				"share": 0,
				"size":  2555572224,
			},
		},
		"type": "process",
	}

	assert.Equal(t, expectedEvent, filteredEvent)
}

func TestIncludeFields1(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"*"})
	}

	Filters := FilterList{}

	rule, err := NewIncludeFields(IncludeFieldsConfig{
		Fields: []string{"proc.cpu.total_ddd"},
		ConditionConfig: ConditionConfig{Regexp: map[string]string{
			"proc.cmdline": "launchd",
		}},
	})
	assert.True(t, err == nil)

	Filters.Register(rule)

	event := common.MapStr{
		"@timestamp": "2016-01-24T18:35:19.308Z",
		"beat": common.MapStr{
			"hostname": "mar",
			"name":     "my-shipper-1",
		},

		"proc": common.MapStr{
			"cpu": common.MapStr{
				"start_time": "Jan14",
				"system":     26027,
				"total":      79390,
				"total_p":    0,
				"user":       53363,
			},
			"cmdline": "/sbin/launchd",
			"mem": common.MapStr{
				"rss":   11194368,
				"rss_p": 0,
				"share": 0,
				"size":  2555572224,
			},
		},
		"type": "process",
	}

	filteredEvent := Filters.Filter(event)

	expectedEvent := common.MapStr{
		"@timestamp": "2016-01-24T18:35:19.308Z",
		"type":       "process",
	}

	assert.Equal(t, expectedEvent, filteredEvent)
}

func TestDropFields(t *testing.T) {

	Filters := FilterList{}

	rule, err := NewDropFields(DropFieldsConfig{
		Fields: []string{"proc.cpu.start_time", "mem", "proc.cmdline", "beat", "dd"},
		ConditionConfig: ConditionConfig{Equals: map[string]string{
			"beat.hostname": "mar",
		}},
	})
	assert.True(t, err == nil)

	Filters.Register(rule)

	event := common.MapStr{
		"@timestamp": "2016-01-24T18:35:19.308Z",
		"beat": common.MapStr{
			"hostname": "mar",
			"name":     "my-shipper-1",
		},

		"proc": common.MapStr{
			"cpu": common.MapStr{
				"start_time": "Jan14",
				"system":     26027,
				"total":      79390,
				"total_p":    0,
				"user":       53363,
			},
			"cmdline": "/sbin/launchd",
		},
		"mem": common.MapStr{
			"rss":   11194368,
			"rss_p": 0,
			"share": 0,
			"size":  2555572224,
		},
		"type": "process",
	}

	filteredEvent := Filters.Filter(event)

	expectedEvent := common.MapStr{
		"@timestamp": "2016-01-24T18:35:19.308Z",
		"proc": common.MapStr{
			"cpu": common.MapStr{
				"system":  26027,
				"total":   79390,
				"total_p": 0,
				"user":    53363,
			},
		},
		"type": "process",
	}

	assert.Equal(t, expectedEvent, filteredEvent)
}

func TestMultipleIncludeFields(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"*"})
	}
	Filters := FilterList{}

	rule, err := NewIncludeFields(IncludeFieldsConfig{
		Fields: []string{"proc"},
		ConditionConfig: ConditionConfig{Contains: map[string]string{
			"beat.name": "my-shipper",
		}},
	})
	assert.True(t, err == nil)

	Filters.Register(rule)

	rule, err = NewIncludeFields(IncludeFieldsConfig{
		Fields: []string{"proc.cpu.start_time", "proc.cpu.total_p",
			"proc.mem.rss_p", "proc.cmdline"},
	})
	assert.True(t, err == nil)

	Filters.Register(rule)

	event1 := common.MapStr{
		"@timestamp": "2016-01-24T18:35:19.308Z",
		"beat": common.MapStr{
			"hostname": "mar",
			"name":     "my-shipper-1",
		},

		"proc": common.MapStr{
			"cpu": common.MapStr{
				"start_time": "Jan14",
				"system":     26027,
				"total":      79390,
				"total_p":    0,
				"user":       53363,
			},
			"cmdline": "/sbin/launchd",
		},
		"mem": common.MapStr{
			"rss":   11194368,
			"rss_p": 0,
			"share": 0,
			"size":  2555572224,
		},
		"type": "process",
	}

	event2 := common.MapStr{
		"@timestamp": "2016-01-24T18:35:19.308Z",
		"beat": common.MapStr{
			"hostname": "mar",
			"name":     "my-shipper-1",
		},
		"fs": common.MapStr{
			"device_name": "devfs",
			"total":       198656,
			"used":        198656,
			"used_p":      1,
			"free":        0,
			"avail":       0,
			"files":       677,
			"free_files":  0,
			"mount_point": "/dev",
		},
		"type": "process",
	}

	expected1 := common.MapStr{
		"@timestamp": "2016-01-24T18:35:19.308Z",
		"proc": common.MapStr{
			"cpu": common.MapStr{
				"start_time": "Jan14",
				"total_p":    0,
			},
			"cmdline": "/sbin/launchd",
		},

		"type": "process",
	}

	expected2 := common.MapStr{
		"@timestamp": "2016-01-24T18:35:19.308Z",
		"type":       "process",
	}

	actual1 := Filters.Filter(event1)
	actual2 := Filters.Filter(event2)

	assert.Equal(t, expected1, actual1)
	assert.Equal(t, expected2, actual2)
}
