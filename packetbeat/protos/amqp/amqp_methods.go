package amqp

import (
	"encoding/binary"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"strconv"
	"strings"
)

func connectionStartMethod(m *AmqpMessage, args []byte) (bool, bool) {
	major := args[0]
	minor := args[1]
	properties := make(common.MapStr)
	next, err, exists := getTable(properties, args, 2)
	if err {
		//failed to get de peer-properties, size may be wrong, let's quit
		logp.Warn("Failed to parse server properties in connection.start method")
		return false, false
	}
	mechanisms, next, err := getShortString(args, next+4, binary.BigEndian.Uint32(args[next:next+4]))
	if err {
		logp.Warn("Failed to get connection mechanisms")
		return false, false
	}
	locales, _, err := getShortString(args, next+4, binary.BigEndian.Uint32(args[next:next+4]))
	if err {
		logp.Warn("Failed to get connection locales")
		return false, false
	}
	m.Method = "connection.start"
	m.IsRequest = true
	m.Fields = common.MapStr{
		"version-major": major,
		"version-minor": minor,
		"mechanisms":    mechanisms,
		"locales":       locales,
	}
	//if there is a server properties table, add it
	if exists {
		m.Fields["server-properties"] = properties
	}
	return true, true
}

func connectionStartOkMethod(m *AmqpMessage, args []byte) (bool, bool) {
	properties := make(common.MapStr)
	next, err, exists := getTable(properties, args, 0)
	if err {
		//failed to get de peer-properties, size may be wrong, let's quit
		logp.Warn("Failed to parse server properties in connection.start method")
		return false, false
	}
	mechanism, next, err := getShortString(args, next+1, uint32(args[next]))
	if err {
		logp.Warn("Failed to get connection mechanism from client")
		return false, false
	}
	_, next, err = getShortString(args, next+4, binary.BigEndian.Uint32(args[next:next+4]))
	if err {
		logp.Warn("Failed to get connection response from client")
		return false, false
	}
	locale, _, err := getShortString(args, next+1, uint32(args[next]))
	if err {
		logp.Warn("Failed to get connection locale from client")
		return false, false
	}
	m.IsRequest = false
	m.Fields = common.MapStr{
		"mechanism": mechanism,
		"locale":    locale,
	}
	//if there is a client properties table, add it
	if exists {
		m.Fields["client-properties"] = properties
	}
	return true, true
}

func connectionTuneMethod(m *AmqpMessage, args []byte) (bool, bool) {
	m.IsRequest = true
	m.Method = "connection.tune"
	//parameters are not parsed here, they are further negociated by the server
	//in the connection.tune-ok method
	return true, true
}

func connectionTuneOkMethod(m *AmqpMessage, args []byte) (bool, bool) {
	m.Fields = common.MapStr{
		"channel-max": binary.BigEndian.Uint16(args[0:2]),
		"frame-max":   binary.BigEndian.Uint32(args[2:6]),
		"heartbeat":   binary.BigEndian.Uint16(args[6:8]),
	}
	return true, true
}

func connectionOpenMethod(m *AmqpMessage, args []byte) (bool, bool) {
	m.IsRequest = true
	m.Method = "connection.open"
	host, _, err := getShortString(args, 1, uint32(args[0]))
	if err {
		logp.Warn("Failed to get virtual host from client")
		return false, false
	}
	m.Fields = common.MapStr{"virtual-host": host}
	return true, true
}

func connectionCloseMethod(m *AmqpMessage, args []byte) (bool, bool) {
	err := getCloseInfo(args, m)
	if err {
		return false, false
	}
	m.Method = "connection.close"
	m.IsRequest = true
	return true, true
}

func channelOpenMethod(m *AmqpMessage, args []byte) (bool, bool) {
	m.Method = "channel.open"
	m.IsRequest = true
	return true, true
}

func channelFlowMethod(m *AmqpMessage, args []byte) (bool, bool) {
	m.Method = "channel.flow"
	m.IsRequest = true
	return true, true
}

func channelFlowOkMethod(m *AmqpMessage, args []byte) (bool, bool) {
	params := getBitParams(args[0])
	m.Fields = common.MapStr{"active": params[0]}
	return true, true
}

func channelCloseMethod(m *AmqpMessage, args []byte) (bool, bool) {
	m.Method = "channel.close"
	m.IsRequest = true
	err := getCloseInfo(args, m)
	if err {
		return false, false
	}
	return true, true
}

//function to fetch fields from channel close and connection close
func getCloseInfo(args []byte, m *AmqpMessage) bool {
	code := binary.BigEndian.Uint16(args[0:2])
	m.IsRequest = true
	replyText, nextOffset, err := getShortString(args, 3, uint32(args[2]))
	if err {
		logp.Warn("Failed to get error reply text")
		return true
	}
	m.Fields = common.MapStr{
		"reply-code": code,
		"reply-text": replyText,
		"class-id":   binary.BigEndian.Uint16(args[nextOffset : nextOffset+2]),
		"method-id":  binary.BigEndian.Uint16(args[nextOffset+2 : nextOffset+4]),
	}
	return false
}

func queueDeclareMethod(m *AmqpMessage, args []byte) (bool, bool) {
	name, offset, err := getShortString(args, 3, uint32(args[2]))
	if err {
		logp.Warn("Error getting name of queue in queue declaration")
		return false, false
	}
	m.IsRequest = true
	m.Method = "queue.declare"
	params := getBitParams(args[offset])
	m.Request = name
	m.Fields = common.MapStr{
		"queue":       name,
		"passive":     params[0],
		"durable":     params[1],
		"exclusive":   params[2],
		"auto-delete": params[3],
		"no-wait":     params[4],
	}
	if args[offset+1] != frameEndOctet && m.ParseArguments {
		arguments := make(common.MapStr)
		_, err, exists := getTable(arguments, args, offset+1)
		if !err && exists {
			m.Fields["arguments"] = arguments
		} else if err {
			m.Notes = append(m.Notes, "Failed to parse additional arguments")
		}
	}
	return true, true
}

func queueDeclareOkMethod(m *AmqpMessage, args []byte) (bool, bool) {
	name, nextOffset, err := getShortString(args, 1, uint32(args[0]))
	if err {
		logp.Warn("Error getting name of queue in queue confirmation")
		return false, false
	}
	m.Method = "queue.declare-ok"
	m.Fields = common.MapStr{
		"queue":          name,
		"consumer-count": binary.BigEndian.Uint32(args[nextOffset+4:]),
		"message-count":  binary.BigEndian.Uint32(args[nextOffset : nextOffset+4]),
	}
	return true, true
}

func queueBindMethod(m *AmqpMessage, args []byte) (bool, bool) {
	queue, offset, err := getShortString(args, 3, uint32(args[2]))
	if err {
		logp.Warn("Error getting name of queue in queue bind")
		return false, false
	}
	m.IsRequest = true
	exchange, offset, err := getShortString(args, offset+1, uint32(args[offset]))
	if err {
		logp.Warn("Error getting name of queue in queue bind")
		return false, false
	}
	routingKey, offset, err := getShortString(args, offset+1, uint32(args[offset]))
	if err {
		logp.Warn("Error getting name of queue in queue bind")
		return false, false
	}
	params := getBitParams(args[offset])
	m.Method = "queue.bind"
	m.Request = strings.Join([]string{queue, exchange}, " ")
	m.Fields = common.MapStr{
		"queue":       queue,
		"routing-key": routingKey,
		"no-wait":     params[0],
	}
	if len(exchange) > 0 {
		m.Fields["exchange"] = exchange
	}
	if args[offset+1] != frameEndOctet && m.ParseArguments {
		arguments := make(common.MapStr)
		_, err, exists := getTable(arguments, args, offset+1)
		if !err && exists {
			m.Fields["arguments"] = arguments
		} else if err {
			m.Notes = append(m.Notes, "Failed to parse additional arguments")
		}
	}
	return true, true
}

func queueUnbindMethod(m *AmqpMessage, args []byte) (bool, bool) {
	queue, offset, err := getShortString(args, 3, uint32(args[2]))
	if err {
		logp.Warn("Error getting name of queue in queue unbind")
		return false, false
	}
	exchange, offset, err := getShortString(args, offset+1, uint32(args[offset]))
	if err {
		logp.Warn("Error getting name of queue in queue unbind")
		return false, false
	}
	routingKey, offset, err := getShortString(args, offset+1, uint32(args[offset]))
	if err {
		logp.Warn("Error getting name of queue in queue unbind")
		return false, false
	}
	m.IsRequest = true
	m.Method = "queue.unbind"
	m.Request = strings.Join([]string{queue, exchange}, " ")
	m.Fields = common.MapStr{
		"queue":       queue,
		"routing-key": routingKey,
	}
	if len(exchange) > 0 {
		m.Fields["exchange"] = exchange
	}
	if args[offset+1] != frameEndOctet && m.ParseArguments {
		arguments := make(common.MapStr)
		_, err, exists := getTable(arguments, args, offset+1)
		if !err && exists {
			m.Fields["arguments"] = arguments
		} else if err {
			m.Notes = append(m.Notes, "Failed to parse additional arguments")
		}
	}
	return true, true
}

func queuePurgeMethod(m *AmqpMessage, args []byte) (bool, bool) {
	queue, nextOffset, err := getShortString(args, 3, uint32(args[2]))
	if err {
		logp.Warn("Error getting name of queue in queue purge")
		return false, false
	}
	m.IsRequest = true
	params := getBitParams(args[nextOffset])
	m.Method = "queue.purge"
	m.Request = queue
	m.Fields = common.MapStr{
		"queue":   queue,
		"no-wait": params[0],
	}
	return true, true
}

func queuePurgeOkMethod(m *AmqpMessage, args []byte) (bool, bool) {
	m.Method = "queue.purge-ok"
	m.Fields = common.MapStr{
		"message-count": binary.BigEndian.Uint32(args[0:4]),
	}
	return true, true
}

func queueDeleteMethod(m *AmqpMessage, args []byte) (bool, bool) {
	queue, nextOffset, err := getShortString(args, 3, uint32(args[2]))
	if err {
		logp.Warn("Error getting name of queue in queue delete")
		return false, false
	}
	m.IsRequest = true
	params := getBitParams(args[nextOffset])
	m.Method = "queue.delete"
	m.Request = queue
	m.Fields = common.MapStr{
		"queue":     queue,
		"if-unused": params[0],
		"if-empty":  params[1],
		"no-wait":   params[2],
	}
	return true, true
}

func queueDeleteOkMethod(m *AmqpMessage, args []byte) (bool, bool) {
	m.Method = "queue.delete-ok"
	m.Fields = common.MapStr{
		"message-count": binary.BigEndian.Uint32(args[0:4]),
	}
	return true, true
}

func exchangeDeclareMethod(m *AmqpMessage, args []byte) (bool, bool) {
	exchange, offset, err := getShortString(args, 3, uint32(args[2]))
	if err {
		logp.Warn("Error getting name of exchange in exchange declare")
		return false, false
	}
	exchangeType, offset, err := getShortString(args, offset+1, uint32(args[offset]))
	if err {
		logp.Warn("Error getting name of routing key in exchange declare")
		return false, false
	}
	params := getBitParams(args[offset])
	m.Method = "exchange.declare"
	m.IsRequest = true
	m.Request = exchange
	if exchangeType == "" {
		exchangeType = "direct"
	}
	m.Fields = common.MapStr{
		"exchange":      exchange,
		"exchange-type": exchangeType,
		"passive":       params[0],
		"durable":       params[1],
		"no-wait":       params[4],
	}
	if args[offset+1] != frameEndOctet && m.ParseArguments {
		arguments := make(common.MapStr)
		_, err, exists := getTable(arguments, args, offset+1)
		if !err && exists {
			m.Fields["arguments"] = arguments
		} else if err {
			m.Notes = append(m.Notes, "Failed to parse additional arguments")
		}
	}
	return true, true
}

func exchangeDeleteMethod(m *AmqpMessage, args []byte) (bool, bool) {
	exchange, nextOffset, err := getShortString(args, 3, uint32(args[2]))
	if err {
		logp.Warn("Error getting name of exchange in exchange delete")
		return false, false
	}
	m.Method = "exchange.delete"
	m.IsRequest = true
	params := getBitParams(args[nextOffset])
	m.Request = exchange
	m.Fields = common.MapStr{
		"exchange":  exchange,
		"if-unused": params[0],
		"no-wait":   params[1],
	}
	return true, true
}

//this is a method exclusive to RabbitMQ
func exchangeBindMethod(m *AmqpMessage, args []byte) (bool, bool) {
	m.Method = "exchange.bind"
	err := exchangeBindUnbindInfo(m, args)
	if err {
		return false, false
	}
	return true, true
}

//this is a method exclusive to RabbitMQ
func exchangeUnbindMethod(m *AmqpMessage, args []byte) (bool, bool) {
	m.Method = "exchange.unbind"
	err := exchangeBindUnbindInfo(m, args)
	if err {
		return false, false
	}
	return true, true
}

func exchangeBindUnbindInfo(m *AmqpMessage, args []byte) bool {
	destination, offset, err := getShortString(args, 3, uint32(args[2]))
	if err {
		logp.Warn("Error getting name of destination in exchange bind/unbind")
		return true
	}
	source, offset, err := getShortString(args, offset+1, uint32(args[offset]))
	if err {
		logp.Warn("Error getting name of source in exchange bind/unbind")
		return true
	}
	routingKey, offset, err := getShortString(args, offset+1, uint32(args[offset]))
	if err {
		logp.Warn("Error getting name of routing-key in exchange bind/unbind")
		return true
	}
	m.IsRequest = true
	params := getBitParams(args[offset])
	m.Request = strings.Join([]string{source, destination}, " ")
	m.Fields = common.MapStr{
		"destination": destination,
		"source":      source,
		"routing-key": routingKey,
		"no-wait":     params[0],
	}
	if args[offset+1] != frameEndOctet && m.ParseArguments {
		arguments := make(common.MapStr)
		_, err, exists := getTable(arguments, args, offset+1)
		if !err && exists {
			m.Fields["arguments"] = arguments
		} else if err {
			m.Notes = append(m.Notes, "Failed to parse additional arguments")
		}
	}
	return false
}

func basicQosMethod(m *AmqpMessage, args []byte) (bool, bool) {
	prefetchSize := binary.BigEndian.Uint32(args[0:4])
	prefetchCount := binary.BigEndian.Uint16(args[4:6])
	params := getBitParams(args[6])
	m.IsRequest = true
	m.Method = "basic.qos"
	m.Fields = common.MapStr{
		"prefetch-size":  prefetchSize,
		"prefetch-count": prefetchCount,
		"global":         params[0],
	}
	return true, true
}

func basicConsumeMethod(m *AmqpMessage, args []byte) (bool, bool) {
	queue, offset, err := getShortString(args, 3, uint32(args[2]))
	if err {
		logp.Warn("Error getting name of queue in basic consume")
		return false, false
	}
	consumerTag, offset, err := getShortString(args, offset+1, uint32(args[offset]))
	if err {
		logp.Warn("Error getting name of consumer tag in basic consume")
		return false, false
	}
	params := getBitParams(args[offset])
	m.Method = "basic.consume"
	m.IsRequest = true
	m.Request = queue
	m.Fields = common.MapStr{
		"queue":        queue,
		"consumer-tag": consumerTag,
		"no-local":     params[0],
		"no-ack":       params[1],
		"exclusive":    params[2],
		"no-wait":      params[3],
	}
	if args[offset+1] != frameEndOctet && m.ParseArguments {
		arguments := make(common.MapStr)
		_, err, exists := getTable(arguments, args, offset+1)
		if !err && exists {
			m.Fields["arguments"] = arguments
		} else if err {
			m.Notes = append(m.Notes, "Failed to parse additional arguments")
		}
	}
	return true, true
}

func basicConsumeOkMethod(m *AmqpMessage, args []byte) (bool, bool) {
	consumerTag, _, err := getShortString(args, 1, uint32(args[0]))
	if err {
		logp.Warn("Error getting name of queue in basic consume")
		return false, false
	}
	m.Method = "basic.consume-ok"
	m.Fields = common.MapStr{
		"consumer-tag": consumerTag,
	}
	return true, true
}

func basicCancelMethod(m *AmqpMessage, args []byte) (bool, bool) {
	consumerTag, offset, err := getShortString(args, 1, uint32(args[0]))
	if err {
		logp.Warn("Error getting consumer tag in basic cancel")
		return false, false
	}
	m.Method = "basic.cancel"
	m.IsRequest = true
	m.Request = consumerTag
	params := getBitParams(args[offset])
	m.Fields = common.MapStr{
		"consumer-tag": consumerTag,
		"no-wait":      params[0],
	}
	return true, true
}

func basicCancelOkMethod(m *AmqpMessage, args []byte) (bool, bool) {
	consumerTag, _, err := getShortString(args, 1, uint32(args[0]))
	if err {
		logp.Warn("Error getting consumer tag in basic cancel ok")
		return false, false
	}
	m.Method = "basic.cancel-ok"
	m.Fields = common.MapStr{
		"consumer-tag": consumerTag,
	}
	return true, true
}

func basicPublishMethod(m *AmqpMessage, args []byte) (bool, bool) {
	exchange, nextOffset, err := getShortString(args, 3, uint32(args[2]))
	if err {
		logp.Warn("Error getting exchange in basic publish")
		return false, false
	}
	routingKey, nextOffset, err := getShortString(args, nextOffset+1, uint32(args[nextOffset]))
	if err {
		logp.Warn("Error getting routing key in basic publish")
		return false, false
	}
	params := getBitParams(args[nextOffset])
	m.Method = "basic.publish"
	m.Fields = common.MapStr{
		"routing-key": routingKey,
		"mandatory":   params[0],
		"immediate":   params[1],
	}
	// is exchange not default exchange ?
	if len(exchange) > 0 {
		m.Fields["exchange"] = exchange
	}
	return true, false
}

func basicReturnMethod(m *AmqpMessage, args []byte) (bool, bool) {
	code := binary.BigEndian.Uint16(args[0:2])
	if code < 300 {
		//not an error or exception ? not interesting
		return true, false
	}
	replyText, nextOffset, err := getShortString(args, 3, uint32(args[2]))
	if err {
		logp.Warn("Error getting name of reply text in basic return")
		return false, false
	}
	exchange, nextOffset, err := getShortString(args, nextOffset+1, uint32(args[nextOffset]))
	if err {
		logp.Warn("Error getting name of exchange in basic return")
		return false, false
	}
	routingKey, nextOffset, err := getShortString(args, nextOffset+1, uint32(args[nextOffset]))
	if err {
		logp.Warn("Error getting name of routing key in basic return")
		return false, false
	}
	m.Method = "basic.return"
	m.Fields = common.MapStr{
		"exchange":    exchange,
		"routing-key": routingKey,
		"reply-code":  code,
		"reply-text":  replyText,
	}
	return true, false
}

func basicDeliverMethod(m *AmqpMessage, args []byte) (bool, bool) {
	consumerTag, offset, err := getShortString(args, 1, uint32(args[0]))
	if err {
		logp.Warn("Failed to get consumer tag in basic deliver")
		return false, false
	}
	deliveryTag := binary.BigEndian.Uint64(args[offset : offset+8])
	params := getBitParams(args[offset+8])
	offset = offset + 9
	exchange, offset, err := getShortString(args, offset+1, uint32(args[offset]))
	if err {
		logp.Warn("Failed to get exchange in basic deliver")
		return false, false
	}
	routingKey, _, err := getShortString(args, offset+1, uint32(args[offset]))
	if err {
		logp.Warn("Failed to get routing key in basic deliver")
		return false, false
	}
	m.Method = "basic.deliver"
	m.Fields = common.MapStr{
		"consumer-tag": consumerTag,
		"delivery-tag": deliveryTag,
		"redelivered":  params[0],
		"routing-key":  routingKey,
	}
	// is exchange not default exchange ?
	if len(exchange) > 0 {
		m.Fields["exchange"] = exchange
	}
	return true, false
}

func basicGetMethod(m *AmqpMessage, args []byte) (bool, bool) {
	queue, offset, err := getShortString(args, 3, uint32(args[2]))
	if err {
		logp.Warn("Failed to get queue in basic get method")
		return false, false
	}
	m.Method = "basic.get"
	params := getBitParams(args[offset])
	m.IsRequest = true
	m.Request = queue
	m.Fields = common.MapStr{
		"queue":  queue,
		"no-ack": params[0],
	}
	return true, true
}

func basicGetOkMethod(m *AmqpMessage, args []byte) (bool, bool) {
	params := getBitParams(args[8])
	exchange, offset, err := getShortString(args, 10, uint32(args[9]))
	if err {
		logp.Warn("Failed to get queue in basic get-ok")
		return false, false
	}
	routingKey, offset, err := getShortString(args, offset+1, uint32(args[offset]))
	if err {
		logp.Warn("Failed to get routing key in basic get-ok")
		return false, false
	}
	m.Method = "basic.get-ok"
	m.Fields = common.MapStr{
		"delivery-tag":  binary.BigEndian.Uint64(args[0:8]),
		"redelivered":   params[0],
		"routing-key":   routingKey,
		"message-count": binary.BigEndian.Uint32(args[offset : offset+4]),
	}
	if len(exchange) > 0 {
		m.Fields["exchange"] = exchange
	}
	return true, false
}

func basicGetEmptyMethod(m *AmqpMessage, args []byte) (bool, bool) {
	m.Method = "basic.get-empty"
	return true, true
}

func basicAckMethod(m *AmqpMessage, args []byte) (bool, bool) {
	params := getBitParams(args[8])
	m.Method = "basic.ack"
	m.IsRequest = true
	m.Fields = common.MapStr{
		"delivery-tag": binary.BigEndian.Uint64(args[0:8]),
		"multiple":     params[0],
	}
	return true, true
}

//this is a rabbitMQ specific method
func basicNackMethod(m *AmqpMessage, args []byte) (bool, bool) {
	params := getBitParams(args[8])
	m.Method = "basic.nack"
	m.IsRequest = true
	m.Fields = common.MapStr{
		"delivery-tag": binary.BigEndian.Uint64(args[0:8]),
		"multiple":     params[0],
		"requeue":      params[1],
	}
	return true, true
}

func basicRejectMethod(m *AmqpMessage, args []byte) (bool, bool) {
	params := getBitParams(args[8])
	tag := binary.BigEndian.Uint64(args[0:8])
	m.IsRequest = true
	m.Method = "basic.reject"
	m.Fields = common.MapStr{
		"delivery-tag": tag,
		"multiple":     params[0],
	}
	m.Request = strconv.FormatUint(tag, 10)
	return true, true
}

func basicRecoverMethod(m *AmqpMessage, args []byte) (bool, bool) {
	params := getBitParams(args[0])
	m.IsRequest = true
	m.Method = "basic.recover"
	m.Fields = common.MapStr{
		"requeue": params[0],
	}
	return true, true
}

func txSelectMethod(m *AmqpMessage, args []byte) (bool, bool) {
	m.IsRequest = true
	m.Method = "tx.select"
	return true, true
}

func txCommitMethod(m *AmqpMessage, args []byte) (bool, bool) {
	m.IsRequest = true
	m.Method = "tx.commit"
	return true, true
}

func txRollbackMethod(m *AmqpMessage, args []byte) (bool, bool) {
	m.IsRequest = true
	m.Method = "tx.rollback"
	return true, true
}

//simple function used when server/client responds to a sync method with no new info
func okMethod(m *AmqpMessage, args []byte) (bool, bool) {
	return true, true
}

// function to get a short string. It sends back an error if slice is too short
//for declared length. if length == 0, the function sends back an empty string and
//advances the offset. Otherwise, it returns the string and the new offset
func getShortString(data []byte, start uint32, length uint32) (short string, nextOffset uint32, err bool) {
	if length == 0 {
		return "", start, false
	}
	if uint32(len(data)) < start || uint32(len(data[start:])) < length {
		return "", 0, true
	}
	return string(data[start : start+length]), start + length, false
}

//function to extract bit information in various AMQP methods
func getBitParams(bits byte) (ret [5]bool) {
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
	return ret
}
