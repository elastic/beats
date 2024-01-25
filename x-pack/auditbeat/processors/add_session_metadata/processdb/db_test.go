// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux

package processdb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/sys/unix"

	"github.com/elastic/elastic-agent-libs/logp"
)

var logger = logp.NewLogger("processdb")

func TestGetTtyType(t *testing.T) {
	assert.Equal(t, TtyConsole, getTtyType(4, 0))
	assert.Equal(t, Pts, getTtyType(136, 0))
	assert.Equal(t, Tty, getTtyType(4, 64))
	assert.Equal(t, TtyUnknown, getTtyType(1000, 1000))
}

func TestCapsFromU64ToECS(t *testing.T) {
	expected := []string{"CAP_CHOWN"}
	assert.Equal(t, expected, ecsCapsFromU64(uint64(1<<unix.CAP_CHOWN)))

	expected = []string{"CAP_SYS_ADMIN"}
	assert.Equal(t, expected, ecsCapsFromU64(uint64(1<<unix.CAP_SYS_ADMIN)))

	expected = []string{"CAP_BPF"}
	assert.Equal(t, expected, ecsCapsFromU64(uint64(1<<39)))

	expected = []string{"CAP_CHECKPOINT_RESTORE"}
	assert.Equal(t, expected, ecsCapsFromU64(uint64(1<<40)))

	expected = []string{"41"}
	assert.Equal(t, expected, ecsCapsFromU64(uint64(1<<41)))

	expected = []string{"63"}
	assert.Equal(t, expected, ecsCapsFromU64(uint64(1<<63)))

	expected = []string{"CAP_CHOWN", "CAP_SYS_ADMIN", "CAP_BPF", "CAP_CHECKPOINT_RESTORE", "41", "63"}
	caps := uint64(1 << unix.CAP_CHOWN)
	caps |= uint64(1 << unix.CAP_SYS_ADMIN)
	caps |= uint64(1 << unix.CAP_BPF)
	caps |= uint64(1 << unix.CAP_CHECKPOINT_RESTORE)
	caps |= uint64(1 << 41)
	caps |= uint64(1 << 63)
	assert.Equal(t, expected, ecsCapsFromU64(caps))
}
