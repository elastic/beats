package spool

import (
	"github.com/elastic/beats/libbeat/publisher"
)

// producer -> broker API
type (
	pushRequest struct {
		event publisher.Event
		seq   uint32
		state *produceState
	}

	producerCancelRequest struct {
		state *produceState
		resp  chan producerCancelResponse
	}

	producerCancelResponse struct {
		removed int
	}
)

// consumer -> broker API

type (
	getRequest struct {
		sz   int              // request sz events from the broker
		resp chan getResponse // channel to send response to
	}

	getResponse struct {
		ack chan batchAckMsg
		err error
		buf []publisher.Event
	}

	batchAckMsg struct{}

	batchCancelRequest struct {
		// ack *ackChan
	}
)
