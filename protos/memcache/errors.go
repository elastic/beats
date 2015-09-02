package memcache

// All memcache protocol errors are defined here.

import (
	"errors"
)

var (
	ErrNotImplemented = errors.New("not implemented")
)

// memcache text parser errors
var (
	ErrParserCaughtInError  = errors.New("memcache parser caught in error loop")
	ErrParserUnknownCommand = errors.New("unknown memcache command found")
	ErrNoMoreArgument       = errors.New("no more command arguments")
	ErrExpectedNumber       = errors.New("expected number value")
	ErrExpectedNoReply      = errors.New("expected noreply in argument list")
	ErrExpectedCRLF         = errors.New("expected CRLF")
	ErrExpectedKeys         = errors.New("message has no keys")
)

// memcache binary parser errors
var (
	ErrInvalidMemcacheMagic = errors.New("invalid memcache magic number")
	ErrLen                  = errors.New("length field error")
)

// memcache UDP errors
var (
	ErrUdpIncompleteMessage = errors.New("attempt to parse incomplete message failed")
)

// memcache transaction/message errors
var (
	ErrInvalidMessage             = errors.New("message is invalid")
	ErrMixOfBinaryAndText         = errors.New("mix of binary and text in one connection")
	ErrResponseUnknownTransaction = errors.New("response from unknown transaction")
	ErrExpectedValueForMerge      = errors.New("expected value to merge with")
	ErrExpectedStatsForMerge      = errors.New("expected stat respose to merge with")
)

// internal notes to be published with transaction
var (
	NoteRequestPacketLoss    = "Packet loss while capturing the request"
	NoteResponsePacketLoss   = "Packet loss while capturing the response"
	NoteTransactionNoRequ    = "Invalid transaction. Request missing"
	NoteNonQuietResponseOnly = "Missing response for non-quiet request"
	NoteTransUnfinished      = "Unfinished transaction"
)
