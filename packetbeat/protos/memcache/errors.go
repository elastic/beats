package memcache

// All memcache protocol errors are defined here.

import (
	"errors"
)

var (
	errNotImplemented = errors.New("not implemented")
)

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
	errExpectedStatsForMerge      = errors.New("expected stat respose to merge with")
)

// internal notes to be published with transaction
var (
	noteRequestPacketLoss    = "Packet loss while capturing the request"
	noteResponsePacketLoss   = "Packet loss while capturing the response"
	noteTransactionNoRequ    = "Invalid transaction. Request missing"
	noteNonQuietResponseOnly = "Missing response for non-quiet request"
	noteTransUnfinished      = "Unfinished transaction"
)
