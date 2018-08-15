package mtest

import (
	"github.com/elastic/beats/libbeat/tests/compose"
)

var (
	Runner = compose.TestRunner{
		Service: "redis",
		Options: map[string][]string{
			"REDIS_VERSION": []string{
				"3.2.12",
				"4.0.11",
				"5.0-rc",
			},
			"IMAGE_OS": []string{
				"alpine",
				"stretch",
			},
		},
		Parallel: true,
	}

	DataRunner = compose.TestRunner{
		Service: "redis",
		Options: map[string][]string{
			"REDIS_VERSION": []string{
				"4.0.11",
			},
			"IMAGE_OS": []string{
				"alpine",
			},
		},
		Parallel: true,
	}
)
