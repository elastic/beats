// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package browser

import (
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/x-pack/heartbeat/monitors/browser/source"
)

type Config struct {
	Schedule  string                 `config:"schedule"`
	Params    map[string]interface{} `config:"params"`
	RawConfig *common.Config
	Source    *source.Source `config:"source"`
}
