// +build !integration

package memcache

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_TextParseQuitCommand(t *testing.T) {
	msg := textParseNoFail(t, "quit\r\n")
	assert.NotNil(t, msg)
	assert.Equal(t, MemcacheCmdQuit, msg.command.code)

	msg = textParseNoFail(t, "quit noreply\r\n")
	assert.NotNil(t, msg)
	assert.Equal(t, MemcacheCmdQuit, msg.command.code)
	assert.True(t, msg.noreply)
}

func Test_TextParseUnknownCommand(t *testing.T) {
	msg := textParseNoFail(t, "unknown command\r\n")
	assert.NotNil(t, msg)

	event := makeMessageEvent(t, msg)
	assert.Equal(t, "UNKNOWN", event["command"])
	assert.Equal(t, "unknown command", event["line"].(memcacheString).String())
}

func Test_TextSimpleGetCommand(t *testing.T) {
	msg := textParseNoFail(t, "get k\r\n")
	assert.NotNil(t, msg)
	assert.Equal(t, MemcacheCmdGet, msg.command.code)
	assert.Equal(t, 1, len(msg.keys))
	assert.Equal(t, "k", msg.keys[0].String())

	event := makeMessageEvent(t, msg)
	assert.Equal(t, "get", event["command"])
	assert.Equal(t, msg.keys, event["keys"])

	msg = textParseNoFail(t, "VALUE k 0 5 10\r\nvalue\r\n")
	assert.NotNil(t, msg)
	assert.Equal(t, uint64(10), msg.casUnique)
	assert.Equal(t, 1, len(msg.keys))
	assert.Equal(t, "k", msg.keys[0].String())

	event = makeMessageEvent(t, msg)
	assert.Equal(t, "VALUE", event["command"])
	assert.Equal(t, uint(5), event["bytes"])
}

func Test_TextParseMultiGetCommand(t *testing.T) {
	msg := textParseNoFail(t, "get a b c d\r\n")
	assert.NotNil(t, msg)
	assert.Equal(t, MemcacheCmdGet, msg.command.code)
	assert.Equal(t, 4, len(msg.keys))
	assert.Equal(t, "a", msg.keys[0].String())
	assert.Equal(t, "b", msg.keys[1].String())
	assert.Equal(t, "c", msg.keys[2].String())
	assert.Equal(t, "d", msg.keys[3].String())

	// let's see if we can parse with bumps...
	p := newTextTestParser(t)
	msg = p.textNoFail("get a b ")
	assert.Nil(t, msg)
	msg = p.textNoFail("c d\r\n")
	assert.NotNil(t, msg)
	assert.Equal(t, MemcacheCmdGet, msg.command.code)
	assert.Equal(t, 4, len(msg.keys))
	if len(msg.keys) == 4 {
		assert.Equal(t, "a", msg.keys[0].String())
		assert.Equal(t, "b", msg.keys[1].String())
		assert.Equal(t, "c", msg.keys[2].String())
		assert.Equal(t, "d", msg.keys[3].String())
	}
}

func Test_TextParseFailingGet(t *testing.T) {
	_, err := textTryParse(t, "get\r\n")
	assert.Equal(t, ErrExpectedKeys, err)
}

func Test_TextParseSet(t *testing.T) {
	msg := textParseNoFail(t, "set k 2 102 5\r\nvalue\r\n")
	assert.NotNil(t, msg)
	assert.Equal(t, MemcacheCmdSet, msg.command.code)
	assert.False(t, msg.noreply)
	assert.Equal(t, "k", msg.keys[0].String())
	assert.Equal(t, uint32(2), msg.flags)
	assert.Equal(t, uint32(102), msg.exptime)
	assert.Equal(t, uint(5), msg.bytes)
	assert.Equal(t, "value", msg.values[0].String())

	event := makeMessageEvent(t, msg)
	assert.Equal(t, "set", event["command"])
	assert.Equal(t, msg.keys, event["keys"])
	assert.Equal(t, false, event["noreply"])
	assert.Equal(t, uint32(2), event["flags"])
	assert.Equal(t, uint32(102), event["exptime"])
	assert.Equal(t, uint(5), event["bytes"])

	msg = textParseNoFail(t, "STORED\r\n")
	assert.NotNil(t, msg)
	assert.Equal(t, MemcacheResStored, msg.command.code)
	assert.Equal(t, MemcacheSuccessResp, msg.command.typ)

	event = makeMessageEvent(t, msg)
	assert.Equal(t, "STORED", event["command"])

	msg = textParseNoFail(t, "set k 2 102 5 noreply\r\nvalue\r\n")
	assert.NotNil(t, msg)
	assert.True(t, msg.noreply)
	assert.Equal(t, MemcacheCmdSet, msg.command.code)
	assert.Equal(t, "k", msg.keys[0].String())
	assert.Equal(t, uint32(2), msg.flags)
	assert.Equal(t, uint32(102), msg.exptime)
	assert.Equal(t, uint(5), msg.bytes)
	assert.Equal(t, "value", msg.values[0].String())
}

func Test_TextParseSetCont(t *testing.T) {
	p := newTextTestParser(t)

	msg := p.textNoFail("set k 2 102 10\r\nvalue")
	assert.Nil(t, msg)

	msg = p.textNoFail("value\r\n")
	assert.NotNil(t, msg)
	assert.False(t, msg.noreply)
	assert.Equal(t, MemcacheCmdSet, msg.command.code)
	assert.Equal(t, "k", msg.keys[0].String())
	assert.Equal(t, uint32(2), msg.flags)
	assert.Equal(t, uint32(102), msg.exptime)
	assert.Equal(t, uint(10), msg.bytes)
	assert.Equal(t, "valuevalue", msg.values[0].String())

	msg = p.textNoFail("BUSY ...\r\n")
	assert.NotNil(t, msg)
	assert.Equal(t, MemcacheErrBusy, msg.command.code)
	assert.Equal(t, MemcacheFailResp, msg.command.typ)
}

func Test_TextParseCasCommand(t *testing.T) {
	msg := textParseNoFail(t, "cas k 2 102 5 1234\r\nvalue\r\n")
	assert.NotNil(t, msg)
	assert.False(t, msg.noreply)
	assert.Equal(t, MemcacheCmdCas, msg.command.code)
	assert.Equal(t, "k", msg.keys[0].String())
	assert.Equal(t, uint32(2), msg.flags)
	assert.Equal(t, uint32(102), msg.exptime)
	assert.Equal(t, uint(5), msg.bytes)
	assert.Equal(t, "value", msg.values[0].String())
	assert.True(t, msg.isCas)
	assert.Equal(t, uint64(1234), msg.casUnique)
}

func Test_TextParseSlabsCommands(t *testing.T) {
	msg := textParseNoFail(t, "slabs reassign -1 2\r\n")
	assert.NotNil(t, msg)
	assert.Equal(t, MemcacheCmdSlabsReassign, msg.command.code)
	assert.Equal(t, int64(-1), msg.ivalue)
	assert.Equal(t, int64(2), msg.ivalue2)

	event := makeMessageEvent(t, msg)
	assert.Equal(t, "slabs reassign", event["command"])
	assert.Equal(t, int64(-1), event["source_class"])
	assert.Equal(t, int64(2), event["dest_class"])
}

func Test_TextParseSlabsUnknown(t *testing.T) {
	msg := textParseNoFail(t, "slabs unknown -1 2\r\n")
	assert.NotNil(t, msg)
	assert.Equal(t, MemcacheCmdUNKNOWN, msg.command.code)
}

func Test_TextSlabsAutomove(t *testing.T) {
	msg := textParseNoFail(t, "slabs automove 1\r\n")
	event := makeMessageEvent(t, msg)
	assert.Equal(t, "slabs automove", event["command"])
	assert.Equal(t, "slow", event["automove"])

	msg = textParseNoFail(t, "slabs automove 0\r\n")
	event = makeMessageEvent(t, msg)
	assert.Equal(t, "slabs automove", event["command"])
	assert.Equal(t, "standby", event["automove"])

	msg = textParseNoFail(t, "slabs automove 2\r\n")
	event = makeMessageEvent(t, msg)
	assert.Equal(t, "slabs automove", event["command"])
	assert.Equal(t, "aggressive", event["automove"])
}

func Test_TextParseCounterResponse(t *testing.T) {
	msg := textParseNoFail(t, "12\r\n")
	assert.NotNil(t, msg)
	assert.Equal(t, MemcacheResCounterOp, msg.command.code)
	assert.Equal(t, uint64(12), msg.value)

	event := makeMessageEvent(t, msg)

	assert.Equal(t, "<counter_op_res>", event["command"])
	assert.Equal(t, uint64(12), event["value"])
}

func Test_TextStatusRequest(t *testing.T) {
	msg := textParseNoFail(t, "stats abc\r\n")
	assert.NotNil(t, msg)

	event := makeMessageEvent(t, msg)
	assert.Equal(t, "stats", event["command"])
	assert.Equal(t, "abc", event["raw_args"].(memcacheString).String())
}

func Test_TextParseStatusResponse(t *testing.T) {
	msg := textParseNoFail(t, "STAT abc 5\r\n")
	assert.NotNil(t, msg)
	assert.Equal(t, MemcacheResStat, msg.command.code)
	assert.Equal(t, "abc", msg.stats[0].Name.String())
	assert.Equal(t, "5", msg.stats[0].Value.String())
}
