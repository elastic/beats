package kafka

import (
	"errors"
	"time"

	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/packetbeat/protos/kafka/internal/kafka"
	"github.com/elastic/beats/packetbeat/protos/kafka/internal/kafka/parse"
)

type transactions struct {
	config *transactionConfig

	messages [2]messageList

	requests  uint8
	responses uint8

	onTransaction transactionHandler
}

const unknownList uint8 = 0xff

type transactionConfig struct {
	transactionTimeout time.Duration
}

type transactionHandler func(
	requMsg *requestMessage,
	respMsg *responseMessage,
) error

// List of messages available for correlation
type messageList struct {
	head, tail *rawMessage
}

var errNoSync = errors.New("streams not in sync")

func (trans *transactions) init(c *transactionConfig, cb transactionHandler) {
	trans.config = c
	trans.requests = unknownList
	trans.responses = unknownList
	trans.onTransaction = cb
}

func (trans *transactions) reset() {
	trans.messages[0].clear()
	trans.messages[1].clear()
}

func (trans *transactions) onMessage(
	dir uint8,
	msg *rawMessage,
) error {
	trans.messages[dir].append(msg)
	if !trans.isSynced() {
		if synced, err := trans.trySync(); !synced {
			debugf("not synced with error: %v", err)
			return err
		}
	}

	// correlate requests and responses
	err := trans.correlate()
	if err != nil {
		// potential syncing issue => reset sync state:
		trans.requests = unknownList
		trans.responses = unknownList
	}
	return err
}

func (trans *transactions) isSynced() bool {
	return trans.requests != unknownList
}

func isValidRequest(header *kafka.RequestHeader) bool {
	return header.APIKey.Valid()
}

func allValidRequests(lst *messageList) bool {
	for msg := lst.head; msg != nil; msg = msg.next {
		header, _, ok := parse.RequestHeader(msg.payload)
		if !ok || !isValidRequest(&header) {
			return false
		}
	}

	return true
}

func (trans *transactions) trySync() (bool, error) {
	if trans.messages[0].empty() || trans.messages[1].empty() {
		return false, nil
	}

	debugf("try to synchronize kafka streams")

	var requStreams [2]bool
	requStreams[0] = allValidRequests(&trans.messages[0])
	requStreams[1] = allValidRequests(&trans.messages[1])
	hasRequestStream := requStreams[0] || requStreams[1]
	if !hasRequestStream {
		debugf("no valid request stream found")

		// if no valid request stream can be found, parse is not in sync
		// with network stream => enforce drop
		return false, errNoSync
	}

	if isDebug {
		debugf("valid request streams: %v", requStreams)
	}

	// try to find first potential request for some response
	// + validate suffix lists for requests and responses to match
	var dropRequ [2]int
	var dropResp [2]int
	var valid [2]bool
	if requStreams[0] {
		debugf("findSyncPoint 0->1")

		dropRequ[0], dropResp[0], valid[0] = findSyncPoint(
			&trans.messages[0],
			&trans.messages[1],
		)
	}
	if requStreams[1] {
		debugf("findSyncPoint 1->0")

		dropRequ[1], dropResp[1], valid[1] = findSyncPoint(
			&trans.messages[1],
			&trans.messages[0],
		)
	}

	debugf("valid sync points: %v", valid)

	// choose and apply sync solution
	solution := -1
	clearResponses := false
	switch {
	case valid[0] && !valid[1]:
		solution = 0

	case !valid[0] && valid[1]:
		solution = 1

	case !valid[0] && !valid[1]:
		clearResponses = true
		switch {
		case requStreams[0] && !requStreams[1]:
			solution = 0

		case requStreams[0] && !requStreams[1]:
			solution = 1
		}

	case valid[0] && valid[1]:
		// TODO: both directions seem valid? => parse actual content to
		//       differentiate between requests and responses
	}

	debugf("favor sync solution: %v", solution)

	if solution == -1 {
		// For now continue syncing if we can not make up our mind
		return false, nil
	}

	trans.requests = uint8(solution)
	trans.responses = uint8(1 - solution)
	trans.messages[trans.requests].drop(dropRequ[solution])

	if clearResponses {
		trans.messages[trans.responses].clear()
	} else {
		trans.messages[trans.responses].drop(dropResp[solution])
	}

	return true, nil
}

// findSyncPoint tries to find the first response pontentially matching a request
func findSyncPoint(requests, responses *messageList) (int, int, bool) {
	dropResp := 0

	for resp := responses.head; resp != nil; resp = resp.next {
		respHeader, _, _ := parse.ResponseHeader(resp.payload)
		debugf("try response with correlation ID: %v", respHeader.CorrelationID)

		// find request potentially matching the response
		dropRequ := 0
		requ := requests.head
		for ; requ != nil; requ = requ.next {
			requHeader, _, _ := parse.RequestHeader(requ.payload)
			debugf("  with request correlation ID: %v", requHeader.CorrelationId)

			if requHeader.CorrelationId == respHeader.CorrelationID {
				debugf("  found match")
				break
			}

			dropRequ++
		}
		if requ == nil {
			debugf("  no matching request -> try next response")

			// no matching request found
			// -> drop old response and continue
			dropResp++
			continue
		}

		// check suffix of request and response list if messages can be correlated
		matches := true
		tstRequ, tstResp := requ, resp
		for tstRequ != nil && tstResp != nil {
			requHeader, _, _ := parse.RequestHeader(tstRequ.payload)
			respHeader, _, _ := parse.ResponseHeader(tstResp.payload)
			if requHeader.CorrelationId != respHeader.CorrelationID {
				matches = false
				break
			}

			tstRequ, tstResp = tstRequ.next, tstResp.next
		}

		if !matches {
			dropResp++
			continue
		}

		return dropRequ, dropResp, true
	}

	return 0, dropResp, false
}

// correlate tries to correlate requests and responses + initiate parsing of actual
// kafka messages, as message type is only available in requests.
// if parsing or correlation ID mismatch is encountered correlate returns an error
// to indicate some potential syncing issues.
func (trans *transactions) correlate() error {
	requests := &trans.messages[trans.requests]
	responses := &trans.messages[trans.responses]

	// drop responses with missing requests
	if requests.empty() {
		for !responses.empty() {
			logp.Warn("Response from unknown transaction. Ignoring.")
			responses.pop()
		}
		return nil
	}

	debugf("correlate requests and responses")

	// merge requests with responses into transactions
	for !responses.empty() && !requests.empty() {
		resp := responses.pop()
		requ := requests.pop()

		// check we're really in sync
		requHeader, requPayload, requOK := parse.RequestHeader(requ.payload)
		respHeader, respPayload, respOK := parse.ResponseHeader(resp.payload)

		debugf("request: header= %#v, payload=%v, ok=%v",
			requHeader,
			len(requPayload),
			requOK,
		)
		debugf("response: header= %#v, payload=%v, ok=%v",
			respHeader,
			len(respPayload),
			respOK,
		)

		validTransaction := (requOK && respOK) &&
			isValidRequest(&requHeader) &&
			requHeader.CorrelationId == respHeader.CorrelationID
		if !validTransaction {
			debugf("no valid transaction")
			return errNoSync
		}

		requMsg := requestMessage{
			ts:       requ.TS,
			endpoint: requ.endpoint,
			header:   requHeader,
			payload:  requPayload,
			size:     len(requ.payload) + 4,
		}
		respMsg := responseMessage{
			ts:       requ.TS,
			endpoint: resp.endpoint,
			header:   respHeader,
			payload:  respPayload,
			size:     len(resp.payload) + 4,
		}

		if err := trans.onTransaction(&requMsg, &respMsg); err != nil {
			return err
		}
	}

	return nil
}

func (ml *messageList) append(msg *rawMessage) {
	if ml.tail == nil {
		ml.head = msg
	} else {
		ml.tail.next = msg
	}
	msg.next = nil
	ml.tail = msg
}

func (ml *messageList) empty() bool {
	return ml.head == nil
}

func (ml *messageList) clear() {
	ml.head = nil
	ml.tail = nil
}

func (ml *messageList) drop(count int) {
	for ; count > 0; count-- {
		ml.pop()
	}
}

func (ml *messageList) pop() *rawMessage {
	if ml.head == nil {
		return nil
	}

	msg := ml.head
	ml.head = ml.head.next
	if ml.head == nil {
		ml.tail = nil
	}
	return msg
}
