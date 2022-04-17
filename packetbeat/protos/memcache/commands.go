// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package memcache

// Generic memcache command types and helper functions for defining
// binary/text protocol based commands with setters and serializers.

import (
	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/common/streambuf"
)

type commandType struct {
	typ   commandTypeCode
	code  commandCode
	parse parserStateFn
	event eventFn
}

type eventFn func(msg *message, event common.MapStr) error

type argDef struct {
	parse     argParser
	serialize eventFn
}

func argOptional(arg argDef) argDef {
	parse := func(parser *parser, hdr, buf *streambuf.Buffer) error {
		err := arg.parse(parser, hdr, buf)
		if err == errNoMoreArgument {
			return nil
		}
		if err != nil {
			debug("optional err: %s", err)
		}
		return err
	}

	return argDef{
		parse:     parse,
		serialize: arg.serialize,
	}
}

func setValue(msg *message, v uint64) {
	msg.value = v
}

func setValue2(msg *message, v uint64) {
	msg.value2 = v
}

func setFlags(msg *message, flags uint32) {
	msg.flags = flags
}

func setExpTime(msg *message, exptime uint32) {
	msg.exptime = exptime
}

func setCasUnique(msg *message, cas uint64) {
	msg.isCas = true
	msg.casUnique = cas
}

func setByteCount(msg *message, count uint32) {
	msg.bytes = uint(count)
}

func serializeNop(msg *message, event common.MapStr) error {
	return nil
}

func serializeArgs(msg *message, event common.MapStr, args []argDef) error {
	for _, arg := range args {
		if err := arg.serialize(msg, event); err != nil {
			return err
		}
	}
	return nil
}

func serializeValue(name string) eventFn {
	return func(msg *message, event common.MapStr) error {
		event[name] = msg.value
		return nil
	}
}

func serializeValue2(name string) eventFn {
	return func(msg *message, event common.MapStr) error {
		event[name] = msg.value2
		return nil
	}
}

func serializeFlags(msg *message, event common.MapStr) error {
	event["flags"] = msg.flags
	return nil
}

func serializeKeys(msg *message, event common.MapStr) error {
	event["keys"] = msg.keys
	return nil
}

func serializeExpTime(msg *message, event common.MapStr) error {
	event["exptime"] = msg.exptime
	return nil
}

func serializeByteCount(msg *message, event common.MapStr) error {
	event["bytes"] = msg.bytes
	return nil
}

func serializeStats(msg *message, event common.MapStr) error {
	event["stats"] = msg.stats
	return nil
}

func serializeCas(msg *message, event common.MapStr) error {
	event["cas_unique"] = msg.casUnique
	return nil
}
