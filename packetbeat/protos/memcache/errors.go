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

// All memcache protocol errors are defined here.

import (
	"errors"
)

var errNotImplemented = errors.New("not implemented")

// memcache text parser errors
var (
	errParserCaughtInError  = errors.New("memcache parser caught in error loop")
	errParserUnknownCommand = errors.New("unknown memcache command found")
	errNoMoreArgument       = errors.New("no more command arguments")
	errExpectedNumber       = errors.New("expected number value")
	errExpectedNoReply      = errors.New("expected noreply in argument list")
	errExpectedCRLF         = errors.New("expected CRLF")
	errExpectedKeys         = errors.New("message has no keys")
)

// memcache binary parser errors
var (
	errInvalidMemcacheMagic = errors.New("invalid memcache magic number")
	errLen                  = errors.New("length field error")
)

// memcache UDP errors
var (
	errUDPIncompleteMessage = errors.New("attempt to parse incomplete message failed")
)

// memcache transaction/message errors
var (
	errInvalidMessage             = errors.New("message is invalid")
	errMixOfBinaryAndText         = errors.New("mix of binary and text in one connection")
	errResponseUnknownTransaction = errors.New("response from unknown transaction")
	errExpectedValueForMerge      = errors.New("expected value to merge with")
	errExpectedStatsForMerge      = errors.New("expected stat response to merge with")
)

// internal notes to be published with transaction
var (
	noteRequestPacketLoss    = "Packet loss while capturing the request"
	noteResponsePacketLoss   = "Packet loss while capturing the response"
	noteTransactionNoRequ    = "Invalid transaction. Request missing"
	noteNonQuietResponseOnly = "Missing response for non-quiet request"
	noteTransUnfinished      = "Unfinished transaction"
)
