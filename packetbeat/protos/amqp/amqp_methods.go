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

package amqp

import (
	"encoding/binary"
	"strconv"
	"strings"

	// unsafe is used only for unsafe.Sizeof in generic helpers to calculate
	// type sizes. No pointer arithmetic or other unsafe memory operations
	// are performed.
	"unsafe"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func connectionStartMethod(m *amqpMessage, args []byte) (bool, bool) {
	if len(args) < 2 {
		logp.Debug("amqp", "Unexpected end of data")
		return false, false
	}
	major := args[0]
	minor := args[1]
	properties := make(mapstr.M)
	next, err, exists := getTable(properties, args, 2)
	if err {
		// failed to get de peer-properties, size may be wrong, let's quit
		logp.Debug("amqp", "Failed to parse server properties in connection.start method")
		return false, false
	}
	mechanisms, consumed, err := getLVString[uint32](args, next)
	next += consumed
	if err {
		logp.Debug("amqp", "Failed to get connection mechanisms")
		return false, false
	}
	locales, _, err := getLVString[uint32](args, next)
	if err {
		logp.Debug("amqp", "Failed to get connection locales")
		return false, false
	}
	m.method = "connection.start"
	m.isRequest = true
	m.fields = mapstr.M{
		"version-major": major,
		"version-minor": minor,
		"mechanisms":    mechanisms,
		"locales":       locales,
	}
	// if there is a server properties table, add it
	if exists {
		m.fields["server-properties"] = properties
	}
	return true, true
}

func connectionStartOkMethod(m *amqpMessage, args []byte) (bool, bool) {
	properties := make(mapstr.M)
	next, err, exists := getTable(properties, args, 0)
	if err {
		// failed to get de peer-properties, size may be wrong, let's quit
		logp.Debug("amqp", "Failed to parse server properties in connection.start method")
		return false, false
	}
	mechanism, consumed, err := getLVString[uint8](args, next)
	next += consumed
	if err {
		logp.Debug("amqp", "Failed to get connection mechanism from client")
		return false, false
	}
	_, consumed, err = getLVString[uint32](args, next)
	next += consumed
	if err {
		logp.Debug("amqp", "Failed to get connection response from client")
		return false, false
	}
	locale, _, err := getLVString[uint8](args, next)
	if err {
		logp.Debug("amqp", "Failed to get connection locale from client")
		return false, false
	}
	m.isRequest = false
	m.fields = mapstr.M{
		"mechanism": mechanism,
		"locale":    locale,
	}
	// if there is a client properties table, add it
	if exists {
		m.fields["client-properties"] = properties
	}
	return true, true
}

func connectionTuneMethod(m *amqpMessage, args []byte) (bool, bool) {
	m.isRequest = true
	m.method = "connection.tune"
	// parameters are not parsed here, they are further negotiated by the server
	// in the connection.tune-ok method
	return true, true
}

func connectionTuneOkMethod(m *amqpMessage, args []byte) (bool, bool) {
	channelMax, err := getIntegerAt[uint16](args, 0)
	if err {
		return false, false
	}
	frameMax, err := getIntegerAt[uint32](args, 2)
	if err {
		return false, false
	}
	heartbeat, err := getIntegerAt[uint16](args, 6)
	if err {
		return false, false
	}
	m.fields = mapstr.M{
		"channel-max": channelMax,
		"frame-max":   frameMax,
		"heartbeat":   heartbeat,
	}
	return true, true
}

func connectionOpenMethod(m *amqpMessage, args []byte) (bool, bool) {
	m.isRequest = true
	m.method = "connection.open"
	host, _, err := getLVString[uint8](args, 0)
	if err {
		logp.Debug("amqp", "Failed to get virtual host from client")
		return false, false
	}
	m.fields = mapstr.M{"virtual-host": host}
	return true, true
}

func connectionCloseMethod(m *amqpMessage, args []byte) (bool, bool) {
	err := getCloseInfo(args, m)
	if err {
		return false, false
	}
	m.method = "connection.close"
	m.isRequest = true
	return true, true
}

func channelOpenMethod(m *amqpMessage, args []byte) (bool, bool) {
	m.method = "channel.open"
	m.isRequest = true
	return true, true
}

func channelFlowMethod(m *amqpMessage, args []byte) (bool, bool) {
	m.method = "channel.flow"
	m.isRequest = true
	return true, true
}

func channelFlowOkMethod(m *amqpMessage, args []byte) (bool, bool) {
	params, err := getBitParamsAt(args, 0)
	if err {
		return false, false
	}
	m.fields = mapstr.M{"active": params[0]}
	return true, true
}

func channelCloseMethod(m *amqpMessage, args []byte) (bool, bool) {
	m.method = "channel.close"
	m.isRequest = true
	err := getCloseInfo(args, m)
	if err {
		return false, false
	}
	return true, true
}

// function to fetch fields from channel close and connection close
func getCloseInfo(args []byte, m *amqpMessage) bool {
	code, err := getIntegerAt[uint16](args, 0)
	if err {
		logp.Debug("amqp", "Failed to get close info code")
		return err
	}
	m.isRequest = true
	replyText, consumed, err := getLVString[uint8](args, 2)
	if err {
		logp.Debug("amqp", "Failed to get close info reply text")
		return err
	}
	classID, err := getIntegerAt[uint16](args, consumed+2)
	if err {
		logp.Debug("amqp", "Failed to get close info class-id")
		return err
	}
	methodID, err := getIntegerAt[uint16](args, consumed+4)
	if err {
		logp.Debug("amqp", "Failed to get close info method-id")
		return err
	}

	m.fields = mapstr.M{
		"reply-code": code,
		"reply-text": replyText,
		"class-id":   classID,
		"method-id":  methodID,
	}
	return false
}

func queueDeclareMethod(m *amqpMessage, args []byte) (bool, bool) {
	offset := uint32(2)
	name, consumed, err := getLVString[uint8](args, offset)
	offset += consumed
	if err {
		logp.Debug("amqp", "Error getting name of queue in queue declaration")
		return false, false
	}
	m.isRequest = true
	m.method = "queue.declare"
	params, err := getBitParamsAt(args, offset)
	if err {
		logp.Debug("amqp", "Error getting params in queue declaration")
		return false, false
	}
	m.request = name
	m.fields = mapstr.M{
		"queue":       name,
		"passive":     params[0],
		"durable":     params[1],
		"exclusive":   params[2],
		"auto-delete": params[3],
		"no-wait":     params[4],
	}
	if len(args) <= int(offset+1) {
		logp.Debug("amqp", "Expected end of frame or arguments in queue declaration")
		return false, false
	}
	if args[offset+1] != frameEndOctet && m.parseArguments {
		arguments := make(mapstr.M)
		_, err, exists := getTable(arguments, args, offset+1)
		if !err && exists {
			m.fields["arguments"] = arguments
		} else if err {
			m.notes = append(m.notes, "Failed to parse additional arguments")
		}
	}
	return true, true
}

func queueDeclareOkMethod(m *amqpMessage, args []byte) (bool, bool) {
	name, offset, err := getLVString[uint8](args, 0)
	if err {
		logp.Debug("amqp", "Error getting name of queue in queue confirmation")
		return false, false
	}
	messageCount, err := getIntegerAt[uint32](args, offset)
	if err {
		return false, false
	}
	consumerCount, err := getIntegerAt[uint32](args, offset+4)
	if err {
		return false, false
	}
	m.method = "queue.declare-ok"
	m.fields = mapstr.M{
		"queue":          name,
		"consumer-count": consumerCount,
		"message-count":  messageCount,
	}
	return true, true
}

func queueBindMethod(m *amqpMessage, args []byte) (bool, bool) {
	offset := uint32(2)
	queue, consumed, err := getLVString[uint8](args, offset)
	offset += consumed
	if err {
		logp.Debug("amqp", "Error getting name of queue in queue bind")
		return false, false
	}
	m.isRequest = true
	exchange, consumed, err := getLVString[uint8](args, offset)
	offset += consumed
	if err {
		logp.Debug("amqp", "Error getting name of queue in queue bind")
		return false, false
	}
	routingKey, consumed, err := getLVString[uint8](args, offset)
	offset += consumed
	if err {
		logp.Debug("amqp", "Error getting name of queue in queue bind")
		return false, false
	}
	params, err := getBitParamsAt(args, offset)
	if err {
		logp.Debug("amqp", "Error getting params in queue bind")
		return false, false
	}
	m.method = "queue.bind"
	m.request = strings.Join([]string{queue, exchange}, " ")
	m.fields = mapstr.M{
		"queue":       queue,
		"routing-key": routingKey,
		"no-wait":     params[0],
	}
	if len(exchange) > 0 {
		m.fields["exchange"] = exchange
	}
	if len(args) <= int(offset+1) {
		logp.Debug("amqp", "Expected end of frame or arguments in queue bind")
		return false, false
	}
	if args[offset+1] != frameEndOctet && m.parseArguments {
		arguments := make(mapstr.M)
		_, err, exists := getTable(arguments, args, offset+1)
		if !err && exists {
			m.fields["arguments"] = arguments
		} else if err {
			m.notes = append(m.notes, "Failed to parse additional arguments")
		}
	}
	return true, true
}

func queueUnbindMethod(m *amqpMessage, args []byte) (bool, bool) {
	offset := uint32(2)
	queue, consumed, err := getLVString[uint8](args, offset)
	offset += consumed
	if err {
		logp.Debug("amqp", "Error getting name of queue in queue unbind")
		return false, false
	}
	exchange, consumed, err := getLVString[uint8](args, offset)
	offset += consumed
	if err {
		logp.Debug("amqp", "Error getting name of queue in queue unbind")
		return false, false
	}
	routingKey, consumed, err := getLVString[uint8](args, offset)
	offset += consumed
	if err {
		logp.Debug("amqp", "Error getting name of queue in queue unbind")
		return false, false
	}
	m.isRequest = true
	m.method = "queue.unbind"
	m.request = strings.Join([]string{queue, exchange}, " ")
	m.fields = mapstr.M{
		"queue":       queue,
		"routing-key": routingKey,
	}
	if len(exchange) > 0 {
		m.fields["exchange"] = exchange
	}
	if len(args) <= int(offset+1) {
		logp.Debug("amqp", "Expected end of frame or arguments in queue unbind")
		return false, false
	}
	if args[offset+1] != frameEndOctet && m.parseArguments {
		arguments := make(mapstr.M)
		_, err, exists := getTable(arguments, args, offset+1)
		if !err && exists {
			m.fields["arguments"] = arguments
		} else if err {
			m.notes = append(m.notes, "Failed to parse additional arguments")
		}
	}
	return true, true
}

func queuePurgeMethod(m *amqpMessage, args []byte) (bool, bool) {
	offset := uint32(2)
	queue, consumed, err := getLVString[uint8](args, offset)
	offset += consumed
	if err {
		logp.Debug("amqp", "Error getting name of queue in queue purge")
		return false, false
	}
	m.isRequest = true
	params, err := getBitParamsAt(args, offset)
	if err {
		logp.Debug("amqp", "Error getting params in queue purge")
		return false, false
	}
	m.method = "queue.purge"
	m.request = queue
	m.fields = mapstr.M{
		"queue":   queue,
		"no-wait": params[0],
	}
	return true, true
}

func queuePurgeOkMethod(m *amqpMessage, args []byte) (bool, bool) {
	messageCount, err := getIntegerAt[uint32](args, 0)
	if err {
		return false, false
	}
	m.method = "queue.purge-ok"
	m.fields = mapstr.M{
		"message-count": messageCount,
	}
	return true, true
}

func queueDeleteMethod(m *amqpMessage, args []byte) (bool, bool) {
	offset := uint32(2)
	queue, consumed, err := getLVString[uint8](args, offset)
	offset += consumed
	if err {
		logp.Debug("amqp", "Error getting name of queue in queue delete")
		return false, false
	}
	m.isRequest = true
	params, err := getBitParamsAt(args, offset)
	if err {
		logp.Debug("amqp", "Error getting params in queue delete")
		return false, false
	}
	m.method = "queue.delete"
	m.request = queue
	m.fields = mapstr.M{
		"queue":     queue,
		"if-unused": params[0],
		"if-empty":  params[1],
		"no-wait":   params[2],
	}
	return true, true
}

func queueDeleteOkMethod(m *amqpMessage, args []byte) (bool, bool) {
	messageCount, err := getIntegerAt[uint32](args, 0)
	if err {
		return false, false
	}
	m.method = "queue.delete-ok"
	m.fields = mapstr.M{
		"message-count": messageCount,
	}
	return true, true
}

func exchangeDeclareMethod(m *amqpMessage, args []byte) (bool, bool) {
	offset := uint32(2)
	exchange, consumed, err := getLVString[uint8](args, offset)
	offset += consumed
	if err {
		logp.Debug("amqp", "Error getting name of exchange in exchange declare")
		return false, false
	}
	exchangeType, consumed, err := getLVString[uint8](args, offset)
	offset += consumed
	if err {
		logp.Debug("amqp", "Error getting name of routing key in exchange declare")
		return false, false
	}
	params, err := getBitParamsAt(args, offset)
	if err {
		logp.Debug("amqp", "Error getting params in exchange declare")
		return false, false
	}
	m.method = "exchange.declare"
	m.isRequest = true
	m.request = exchange
	if exchangeType == "" {
		exchangeType = "direct"
	}
	m.fields = mapstr.M{
		"exchange":      exchange,
		"exchange-type": exchangeType,
		"passive":       params[0],
		"durable":       params[1],
		"no-wait":       params[4],
	}
	if len(args) <= int(offset+1) {
		logp.Debug("amqp", "Error getting name of routing key in exchange declare")
		return false, false
	}
	if args[offset+1] != frameEndOctet && m.parseArguments {
		arguments := make(mapstr.M)
		_, err, exists := getTable(arguments, args, offset+1)
		if !err && exists {
			m.fields["arguments"] = arguments
		} else if err {
			m.notes = append(m.notes, "Failed to parse additional arguments")
		}
	}
	return true, true
}

func exchangeDeleteMethod(m *amqpMessage, args []byte) (bool, bool) {
	offset := uint32(2)
	exchange, consumed, err := getLVString[uint8](args, offset)
	offset += consumed
	if err {
		logp.Debug("amqp", "Error getting name of exchange in exchange delete")
		return false, false
	}
	m.method = "exchange.delete"
	m.isRequest = true
	params, err := getBitParamsAt(args, offset)
	if err {
		logp.Debug("amqp", "Error getting params in exchange delete")
		return false, false
	}
	m.request = exchange
	m.fields = mapstr.M{
		"exchange":  exchange,
		"if-unused": params[0],
		"no-wait":   params[1],
	}
	return true, true
}

// this is a method exclusive to RabbitMQ
func exchangeBindMethod(m *amqpMessage, args []byte) (bool, bool) {
	m.method = "exchange.bind"
	err := exchangeBindUnbindInfo(m, args)
	if err {
		return false, false
	}
	return true, true
}

// this is a method exclusive to RabbitMQ
func exchangeUnbindMethod(m *amqpMessage, args []byte) (bool, bool) {
	m.method = "exchange.unbind"
	err := exchangeBindUnbindInfo(m, args)
	if err {
		return false, false
	}
	return true, true
}

func exchangeBindUnbindInfo(m *amqpMessage, args []byte) bool {
	offset := uint32(2)
	destination, consumed, err := getLVString[uint8](args, offset)
	offset += consumed
	if err {
		logp.Debug("amqp", "Error getting name of destination in exchange bind/unbind")
		return true
	}
	source, consumed, err := getLVString[uint8](args, offset)
	offset += consumed
	if err {
		logp.Debug("amqp", "Error getting name of source in exchange bind/unbind")
		return true
	}
	routingKey, consumed, err := getLVString[uint8](args, offset)
	offset += consumed
	if err {
		logp.Debug("amqp", "Error getting name of routing-key in exchange bind/unbind")
		return true
	}
	m.isRequest = true
	params, err := getBitParamsAt(args, offset)
	if err {
		logp.Debug("amqp", "Error getting params in exchange bind/unbind")
		return true
	}
	m.request = strings.Join([]string{source, destination}, " ")
	m.fields = mapstr.M{
		"destination": destination,
		"source":      source,
		"routing-key": routingKey,
		"no-wait":     params[0],
	}

	if len(args) <= int(offset+1) {
		logp.Debug("amqp", "Error getting args in exchange bind/unbind")
		return true
	}
	if args[offset+1] != frameEndOctet && m.parseArguments {
		arguments := make(mapstr.M)
		_, err, exists := getTable(arguments, args, offset+1)
		if !err && exists {
			m.fields["arguments"] = arguments
		} else if err {
			m.notes = append(m.notes, "Failed to parse additional arguments")
		}
	}
	return false
}

func basicQosMethod(m *amqpMessage, args []byte) (bool, bool) {
	prefetchSize, err := getIntegerAt[uint32](args, 0)
	if err {
		logp.Debug("amqp", "Error getting prefetch-size in basic qos")
		return false, false
	}
	prefetchCount, err := getIntegerAt[uint16](args, 4)
	if err {
		logp.Debug("amqp", "Error getting prefetch-count in basic qos")
		return false, false
	}
	params, err := getBitParamsAt(args, 6)
	if err {
		logp.Debug("amqp", "Error getting params in basic qos")
		return false, false
	}
	m.isRequest = true
	m.method = "basic.qos"
	m.fields = mapstr.M{
		"prefetch-size":  prefetchSize,
		"prefetch-count": prefetchCount,
		"global":         params[0],
	}
	return true, true
}

func basicConsumeMethod(m *amqpMessage, args []byte) (bool, bool) {
	offset := uint32(2)
	queue, consumed, err := getLVString[uint8](args, offset)
	offset += consumed
	if err {
		logp.Debug("amqp", "Error getting name of queue in basic consume")
		return false, false
	}
	consumerTag, consumed, err := getLVString[uint8](args, offset)
	offset += consumed
	if err {
		logp.Debug("amqp", "Error getting name of consumer tag in basic consume")
		return false, false
	}
	params, err := getBitParamsAt(args, offset)
	if err {
		logp.Debug("amqp", "Error getting params in basic consume")
		return false, false
	}
	m.method = "basic.consume"
	m.isRequest = true
	m.request = queue
	m.fields = mapstr.M{
		"queue":        queue,
		"consumer-tag": consumerTag,
		"no-local":     params[0],
		"no-ack":       params[1],
		"exclusive":    params[2],
		"no-wait":      params[3],
	}
	if len(args) <= int(offset+1) {
		logp.Debug("amqp", "Expected end of frame or arguments in basic consume")
		return false, false
	}
	if args[offset+1] != frameEndOctet && m.parseArguments {
		arguments := make(mapstr.M)
		_, err, exists := getTable(arguments, args, offset+1)
		if !err && exists {
			m.fields["arguments"] = arguments
		} else if err {
			m.notes = append(m.notes, "Failed to parse additional arguments")
		}
	}
	return true, true
}

func basicConsumeOkMethod(m *amqpMessage, args []byte) (bool, bool) {
	consumerTag, _, err := getLVString[uint8](args, 0)
	if err {
		logp.Debug("amqp", "Error getting name of queue in basic consume")
		return false, false
	}
	m.method = "basic.consume-ok"
	m.fields = mapstr.M{
		"consumer-tag": consumerTag,
	}
	return true, true
}

func basicCancelMethod(m *amqpMessage, args []byte) (bool, bool) {
	consumerTag, offset, err := getLVString[uint8](args, 0)
	if err {
		logp.Debug("amqp", "Error getting consumer tag in basic cancel")
		return false, false
	}
	m.method = "basic.cancel"
	m.isRequest = true
	m.request = consumerTag
	params, err := getBitParamsAt(args, offset)
	if err {
		logp.Debug("amqp", "Error getting params in basic cancel")
		return false, false
	}
	m.fields = mapstr.M{
		"consumer-tag": consumerTag,
		"no-wait":      params[0],
	}
	return true, true
}

func basicCancelOkMethod(m *amqpMessage, args []byte) (bool, bool) {
	consumerTag, _, err := getLVString[uint8](args, 0)
	if err {
		logp.Debug("amqp", "Error getting consumer tag in basic cancel ok")
		return false, false
	}
	m.method = "basic.cancel-ok"
	m.fields = mapstr.M{
		"consumer-tag": consumerTag,
	}
	return true, true
}

func basicPublishMethod(m *amqpMessage, args []byte) (bool, bool) {
	offset := uint32(2)
	exchange, consumed, err := getLVString[uint8](args, offset)
	offset += consumed
	if err {
		logp.Debug("amqp", "Error getting exchange in basic publish")
		return false, false
	}
	routingKey, consumed, err := getLVString[uint8](args, offset)
	offset += consumed
	if err {
		logp.Debug("amqp", "Error getting routing key in basic publish")
		return false, false
	}
	params, err := getBitParamsAt(args, offset)
	if err {
		logp.Debug("amqp", "Error getting params in basic publish")
		return false, false
	}
	m.method = "basic.publish"
	m.fields = mapstr.M{
		"routing-key": routingKey,
		"mandatory":   params[0],
		"immediate":   params[1],
	}
	// is exchange not default exchange ?
	if len(exchange) > 0 {
		m.fields["exchange"] = exchange
	}
	return true, false
}

func basicReturnMethod(m *amqpMessage, args []byte) (bool, bool) {
	code, err := getIntegerAt[uint16](args, 0)
	if err {
		logp.Debug("amqp", "Error getting code in basic return")
		return false, false
	}
	if code < 300 {
		// not an error or exception ? not interesting
		return true, false
	}
	offset := uint32(2)
	replyText, consumed, err := getLVString[uint8](args, offset)
	offset += consumed
	if err {
		logp.Debug("amqp", "Error getting name of reply text in basic return")
		return false, false
	}
	exchange, consumed, err := getLVString[uint8](args, offset)
	offset += consumed
	if err {
		logp.Debug("amqp", "Error getting name of exchange in basic return")
		return false, false
	}
	routingKey, _, err := getLVString[uint8](args, offset)
	if err {
		logp.Debug("amqp", "Error getting name of routing key in basic return")
		return false, false
	}
	m.method = "basic.return"
	m.fields = mapstr.M{
		"exchange":    exchange,
		"routing-key": routingKey,
		"reply-code":  code,
		"reply-text":  replyText,
	}
	return true, false
}

func basicDeliverMethod(m *amqpMessage, args []byte) (bool, bool) {
	consumerTag, offset, err := getLVString[uint8](args, 0)
	if err {
		logp.Debug("amqp", "Failed to get consumer tag in basic deliver")
		return false, false
	}
	params, err := getBitParamsAt(args, offset+8)
	if err {
		logp.Debug("amqp", "Failed to get params in basic deliver")
		return false, false
	}
	// Saving the len check since this is before params in the args so it's guaranteed to be in bounds
	deliveryTag := binary.BigEndian.Uint64(args[offset : offset+8])
	offset = offset + 9
	exchange, consumed, err := getLVString[uint8](args, offset)
	offset += consumed
	if err {
		logp.Debug("amqp", "Failed to get exchange in basic deliver")
		return false, false
	}
	routingKey, _, err := getLVString[uint8](args, offset)
	if err {
		logp.Debug("amqp", "Failed to get routing key in basic deliver")
		return false, false
	}
	m.method = "basic.deliver"
	m.fields = mapstr.M{
		"consumer-tag": consumerTag,
		"delivery-tag": deliveryTag,
		"redelivered":  params[0],
		"routing-key":  routingKey,
	}
	// is exchange not default exchange ?
	if len(exchange) > 0 {
		m.fields["exchange"] = exchange
	}
	return true, false
}

func basicGetMethod(m *amqpMessage, args []byte) (bool, bool) {
	offset := uint32(2)
	queue, consumed, err := getLVString[uint8](args, offset)
	offset += consumed
	if err {
		logp.Debug("amqp", "Failed to get queue in basic get method")
		return false, false
	}
	m.method = "basic.get"
	params, err := getBitParamsAt(args, offset)
	if err {
		logp.Debug("amqp", "Failed to get params in basic get method")
		return false, false
	}
	m.isRequest = true
	m.request = queue
	m.fields = mapstr.M{
		"queue":  queue,
		"no-ack": params[0],
	}
	return true, true
}

func basicGetOkMethod(m *amqpMessage, args []byte) (bool, bool) {
	params, err := getBitParamsAt(args, 8)
	if err {
		logp.Debug("amqp", "Failed to get params in basic get-ok")
		return false, false

	}
	offset := uint32(9)
	exchange, consumed, err := getLVString[uint8](args, offset)
	offset += consumed
	if err {
		logp.Debug("amqp", "Failed to get queue in basic get-ok")
		return false, false
	}
	routingKey, consumed, err := getLVString[uint8](args, offset)
	offset += consumed
	if err {
		logp.Debug("amqp", "Failed to get routing key in basic get-ok")
		return false, false
	}
	m.method = "basic.get-ok"
	deliveryTag, err := getIntegerAt[uint64](args, 0)
	if err {
		return false, false
	}
	messageCount, err := getIntegerAt[uint32](args, offset)
	if err {
		return false, false
	}
	m.fields = mapstr.M{
		"delivery-tag":  deliveryTag,
		"redelivered":   params[0],
		"routing-key":   routingKey,
		"message-count": messageCount,
	}
	if len(exchange) > 0 {
		m.fields["exchange"] = exchange
	}
	return true, false
}

func basicGetEmptyMethod(m *amqpMessage, args []byte) (bool, bool) {
	m.method = "basic.get-empty"
	return true, true
}

func basicAckMethod(m *amqpMessage, args []byte) (bool, bool) {
	deliveryTag, err := getIntegerAt[uint64](args, 0)
	if err {
		logp.Debug("amqp", "Failed to get delivery-tag in basic ack")
		return false, false
	}
	params, err := getBitParamsAt(args, 8)
	if err {
		logp.Debug("amqp", "Failed to get params in basic ack")
		return false, false

	}
	m.method = "basic.ack"
	m.isRequest = true
	m.fields = mapstr.M{
		"delivery-tag": deliveryTag,
		"multiple":     params[0],
	}
	return true, true
}

// this is a rabbitMQ specific method
func basicNackMethod(m *amqpMessage, args []byte) (bool, bool) {
	deliveryTag, err := getIntegerAt[uint64](args, 0)
	if err {
		logp.Debug("amqp", "Failed to get delivery-tag in basic nack")
		return false, false
	}
	params, err := getBitParamsAt(args, 8)
	if err {
		logp.Debug("amqp", "Failed to get params in basic nack")
		return false, false

	}
	m.method = "basic.nack"
	m.isRequest = true
	m.fields = mapstr.M{
		"delivery-tag": deliveryTag,
		"multiple":     params[0],
		"requeue":      params[1],
	}
	return true, true
}

func basicRejectMethod(m *amqpMessage, args []byte) (bool, bool) {
	params, err := getBitParamsAt(args, 8)
	if err {
		logp.Debug("amqp", "Failed to get params in basic reject")
		return false, false
	}
	// Saving the len check since this is before params in the args so it's guaranteed to be in bounds
	tag := binary.BigEndian.Uint64(args[0:8])
	m.isRequest = true
	m.method = "basic.reject"
	m.fields = mapstr.M{
		"delivery-tag": tag,
		"multiple":     params[0],
	}
	m.request = strconv.FormatUint(tag, 10)
	return true, true
}

func basicRecoverMethod(m *amqpMessage, args []byte) (bool, bool) {
	params, err := getBitParamsAt(args, 0)
	if err {
		logp.Debug("amqp", "Failed to get params in basic recover")
		return false, false
	}
	m.isRequest = true
	m.method = "basic.recover"
	m.fields = mapstr.M{
		"requeue": params[0],
	}
	return true, true
}

func txSelectMethod(m *amqpMessage, args []byte) (bool, bool) {
	m.isRequest = true
	m.method = "tx.select"
	return true, true
}

func txCommitMethod(m *amqpMessage, args []byte) (bool, bool) {
	m.isRequest = true
	m.method = "tx.commit"
	return true, true
}

func txRollbackMethod(m *amqpMessage, args []byte) (bool, bool) {
	m.isRequest = true
	m.method = "tx.rollback"
	return true, true
}

// simple function used when server/client responds to a sync method with no new info
func okMethod(m *amqpMessage, args []byte) (bool, bool) {
	return true, true
}

// Function to get a Length-Value string. It is generic over the integer length
// of the length key. It sends back an error if slice is too short for declared
// length and string, or if the offset itself is out of bounds. if length == 0
// the function sends back an empty string. bytesConsumed represents the number
// of bytes of the returned string + 1 for the length itself,
func getLVString[T uint8 | uint16 | uint32](data []byte, offset uint32) (short string, bytesConsumed uint32, err bool) {
	var length T
	if len(data) == 0 {
		return "", 0, true
	}
	lengthSize := uint32(unsafe.Sizeof(length))
	offset64 := int64(offset)
	lengthSize64 := int64(lengthSize)
	dataLen64 := int64(len(data))

	// If there's not enough data to read the length of the string return err
	if offset64 > dataLen64 || lengthSize64 > dataLen64-offset64 {
		return "", 0, true
	}
	offsetInt := int(offset64)
	lengthSizeInt := int(lengthSize64)

	switch any(length).(type) {
	case uint8:
		length = T(data[offsetInt])
	case uint16:
		length = T(binary.BigEndian.Uint16(data[offsetInt : offsetInt+lengthSizeInt]))
	case uint32:
		length = T(binary.BigEndian.Uint32(data[offsetInt : offsetInt+lengthSizeInt]))
	}
	strlen := uint32(length)
	strlen64 := int64(strlen)

	if strlen == 0 {
		return "", lengthSize, false
	}

	start64 := offset64 + lengthSize64
	if strlen64 > dataLen64-start64 {
		logp.Debug("amqp", "Not enough data for string")
		return "", 0, true
	}
	if strlen > ^uint32(0)-lengthSize {
		return "", 0, true
	}

	startInt := int(start64)
	endInt := startInt + int(strlen64)
	return string(data[startInt:endInt]), strlen + lengthSize, false
}

// Attempts to get an integer from a byte slice. Returns the integer and an err boolean.
// The err is true if there were not enough bytes to successfully read the integer type.
// The returned integer is meaningless if err == true
func getIntegerAt[T uint8 | uint16 | uint32 | uint64 | int8 | int16 | int32 | int64](data []byte, offset uint32) (integer T, err bool) {
	var value T
	size := uint32(unsafe.Sizeof(value))
	offset64 := int64(offset)
	size64 := int64(size)
	dataLen64 := int64(len(data))

	// If there's not enough bytes to read the requested integer type
	if offset64 > dataLen64 || size64 > dataLen64-offset64 {
		return T(0), true
	}
	offsetInt := int(offset64)
	sizeInt := int(size64)

	switch any(value).(type) {
	case uint8:
		return T(data[offsetInt]), false
	case uint16:
		return T(binary.BigEndian.Uint16(data[offsetInt : offsetInt+sizeInt])), false
	case uint32:
		return T(binary.BigEndian.Uint32(data[offsetInt : offsetInt+sizeInt])), false
	case uint64:
		return T(binary.BigEndian.Uint64(data[offsetInt : offsetInt+sizeInt])), false
	case int8:
		return T(data[offsetInt]), false
	case int16:
		return T(binary.BigEndian.Uint16(data[offsetInt : offsetInt+sizeInt])), false
	case int32:
		return T(binary.BigEndian.Uint32(data[offsetInt : offsetInt+sizeInt])), false
	case int64:
		return T(binary.BigEndian.Uint64(data[offsetInt : offsetInt+sizeInt])), false
	}
	return T(0), true
}

// function to extract bit information in various AMQP methods at an offset in a byte slice
func getBitParamsAt(data []byte, offset uint32) (ret [5]bool, err bool) {
	if len(data) <= int(offset) {
		return [5]bool{}, true
	}
	bits := data[offset]
	if bits&16 == 16 {
		ret[4] = true
	}
	if bits&8 == 8 {
		ret[3] = true
	}
	if bits&4 == 4 {
		ret[2] = true
	}
	if bits&2 == 2 {
		ret[1] = true
	}
	if bits&1 == 1 {
		ret[0] = true
	}
	return ret, false
}
