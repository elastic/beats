// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package processdb

import (
	"testing"

	"github.com/elastic/beats/v7/x-pack/auditbeat/processors/add_session_metadata/pkg/procfs"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/logp"

	"golang.org/x/sys/unix"
)

var logger = logp.NewLogger("processdb")
var reader = procfs.NewMockReader()

// glue function to fit the return type required by these tests
func newSimpleDBIntf(reader procfs.Reader) DB {
	ret := NewSimpleDB(reader, *logger)
	_ = ret.ScrapeProcfs()
	return ret
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
