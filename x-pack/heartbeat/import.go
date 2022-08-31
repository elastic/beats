//go:build linux || darwin
// +build linux darwin

package main

import (
	_ "github.com/elastic/beats/v7/x-pack/heartbeat/monitors/browser"
)
