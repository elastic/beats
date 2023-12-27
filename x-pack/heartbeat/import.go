// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux || darwin

package main

import (
	_ "github.com/elastic/beats/v7/x-pack/heartbeat/monitors/browser"
)
