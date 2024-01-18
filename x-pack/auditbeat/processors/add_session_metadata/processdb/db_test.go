// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package processdb

import (
	"testing"

	"github.com/elastic/beats/v7/x-pack/auditbeat/processors/add_session_metadata/procfs"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/logp"

	"golang.org/x/sys/unix"
)

var logger = logp.NewLogger("processdb")
var reader = procfs.NewMockReader()

// glue function to fit the return type required by these tests
func newDBIntf(reader procfs.Reader) *DB {
	ret := NewDB(reader, *logger)
	_ = ret.ScrapeProcfs()
	return ret
}


func TestGetTtyType(t *testing.T) {
	assert.Equal(t, TtyConsole, getTtyType(4, 0))
	assert.Equal(t, Pts, getTtyType(136, 0))
	assert.Equal(t, Tty, getTtyType(4, 64))
	assert.Equal(t, TtyUnknown, getTtyType(1000, 1000))
}

func TestSingleProcessSessionLeaderEntryTypeTerminal(t *testing.T) {
	testSingleProcessSessionLeaderEntryTypeTerminal(newDBIntf)(t)
}

func TestSingleProcessSessionLeaderLoginProcess(t *testing.T) {
	testSingleProcessSessionLeaderLoginProcess(newDBIntf)(t)
}

func TestSingleProcessSessionLeaderChildOfInit(t *testing.T) {
	testSingleProcessSessionLeaderChildOfInit(newDBIntf)(t)
}

func TestSingleProcessSessionLeaderChildOfSsmSessionWorker(t *testing.T) {
	testSingleProcessSessionLeaderChildOfSsmSessionWorker(newDBIntf)(t)
}

func TestSingleProcessSessionLeaderChildOfSshd(t *testing.T) {
	testSingleProcessSessionLeaderChildOfSshd(newDBIntf)(t)
}

func TestSingleProcessSessionLeaderChildOfContainerdShim(t *testing.T) {
	testSingleProcessSessionLeaderChildOfContainerdShim(newDBIntf)(t)
}

func TestSingleProcessSessionLeaderOfRunc(t *testing.T) {
	testSingleProcessSessionLeaderChildOfRunc(newDBIntf)(t)
}

func TestSingleProcessEmptyProcess(t *testing.T) {
	testSingleProcessEmptyProcess(newDBIntf)(t)
}

func TestSingleProcessOverwriteOldEntryLeader(t *testing.T) {
	testSingleProcessOverwriteOldEntryLeader(newDBIntf)(t)
}

func TestInitSshdBashLs(t *testing.T) {
	testInitSshdBashLs(newDBIntf)(t)
}

func TestInitSshdSshdBashLs(t *testing.T) {
	testInitSshdSshdBashLs(newDBIntf)(t)
}

func TestInitSshdSshdSshdBashLs(t *testing.T) {
	testInitSshdSshdSshdBashLs(newDBIntf)(t)
}

func TestInitContainerdContainerdShim(t *testing.T) {
	testInitContainerdContainerdShim(newDBIntf)(t)
}

func TestInitContainerdShimBashContainerdShimIsReparentedToInit(t *testing.T) {
	testInitContainerdShimBashContainerdShimIsReparentedToInit(newDBIntf)(t)
}

func TestInitContainerdShimPauseContainerdShimIsReparentedToInit(t *testing.T) {
	testInitContainerdShimPauseContainerdShimIsReparentedToInit(newDBIntf)(t)
}

func TestInitSshdBashLsAndGrepGrepOnlyHasGroupLeader(t *testing.T) {
	testInitSshdBashLsAndGrepGrepOnlyHasGroupLeader(newDBIntf)(t)
}

func TestInitSshdBashLsAndGrepGrepOnlyHasSessionLeader(t *testing.T) {
	testInitSshdBashLsAndGrepGrepOnlyHasSessionLeader(newDBIntf)(t)
}

func TestGrepInIsolation(t *testing.T) {
	testGrepInIsolation(newDBIntf)(t)
}

func TestKernelThreads(t *testing.T) {
	testKernelThreads(newDBIntf)(t)
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
