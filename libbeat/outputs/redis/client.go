package redis

import (
	"errors"
	"regexp"
	"strconv"
	"time"

	"github.com/garyburd/redigo/redis"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/codec"
	"github.com/elastic/beats/libbeat/outputs/outil"
	"github.com/elastic/beats/libbeat/outputs/transport"
	"github.com/elastic/beats/libbeat/publisher"
	"github.com/elastic/beats/libbeat/publisher/beat"
)

var (
	versionRegex = regexp.MustCompile(`redis_version:(\d+).(\d+)`)
)

type publishFn func(
	keys outil.Selector,
	data []publisher.Event,
) ([]publisher.Event, error)

type client struct {
	*transport.Client
	index    string
	dataType redisDataType
	db       int
	key      outil.Selector
	password string
	publish  publishFn
	codec    codec.Codec
	timeout  time.Duration
}

type redisDataType uint16

const (
	redisListType redisDataType = iota
	redisChannelType
)

func newClient(tc *transport.Client, timeout time.Duration, pass string, db int, key outil.Selector, dt redisDataType, index string, codec codec.Codec) *client {
	return &client{
		Client:   tc,
		timeout:  timeout,
		password: pass,
		index:    index,
		db:       db,
		dataType: dt,
		key:      key,
		codec:    codec,
	}
}

func (c *client) Connect() error {
	debugf("connect")
	err := c.Client.Connect()
	if err != nil {
		return err
	}

	to := c.timeout
	conn := redis.NewConn(c.Client, to, to)
	defer func() {
		if err != nil {
			conn.Close()
		}
	}()

	if err = initRedisConn(conn, c.password, c.db); err == nil {
		c.publish, err = makePublish(conn, c.key, c.dataType, c.index, c.codec)
	}
	return err
}

func initRedisConn(c redis.Conn, pwd string, db int) error {
	if pwd != "" {
		if _, err := c.Do("AUTH", pwd); err != nil {
			return err
		}
	}

	if _, err := c.Do("PING"); err != nil {
		return err
	}

	if db != 0 {
		if _, err := c.Do("SELECT", db); err != nil {
			return err
		}
	}

	return nil
}

func (c *client) Close() error {
	debugf("close connection")
	return c.Client.Close()
}

func (c *client) Publish(batch publisher.Batch) error {
	if c == nil {
		panic("no client")
	}
	if batch == nil {
		panic("no batch")
	}

	events := batch.Events()
	rest, err := c.publish(c.key, events)
	if rest != nil {
		batch.RetryEvents(rest)
	}
	return err
}

func makePublish(
	conn redis.Conn,
	key outil.Selector,
	dt redisDataType,
	index string,
	codec codec.Codec,
) (publishFn, error) {
	if dt == redisChannelType {
		return makePublishPUBLISH(conn, index, codec)
	}
	return makePublishRPUSH(conn, key, index, codec)
}

func makePublishRPUSH(conn redis.Conn, key outil.Selector, index string, codec codec.Codec) (publishFn, error) {
	if !key.IsConst() {
		// TODO: more clever bulk handling batching events with same key
		return publishEventsPipeline(conn, "RPUSH", index, codec), nil
	}

	var major, minor int
	var versionRaw [][]byte

	respRaw, err := conn.Do("INFO")
	resp, err := redis.Bytes(respRaw, err)
	if err != nil {
		return nil, err
	}

	versionRaw = versionRegex.FindSubmatch(resp)
	if versionRaw == nil {
		err = errors.New("unable to read redis_version")
		return nil, err
	}

	major, err = strconv.Atoi(string(versionRaw[1]))
	if err != nil {
		return nil, err
	}

	minor, err = strconv.Atoi(string(versionRaw[2]))
	if err != nil {
		return nil, err
	}

	// Check Redis version number choosing the method
	// how RPUSH shall be used. With version 2.4 RPUSH
	// can accept multiple values at once turning RPUSH
	// into batch like call instead of relying on pipelining.
	//
	// Versions 1.0 to 2.3 only accept one value being send with
	// RPUSH requiring pipelining.
	//
	// See: http://redis.io/commands/rpush
	multiValue := major > 2 || (major == 2 && minor >= 4)
	if multiValue {
		return publishEventsBulk(conn, key, "RPUSH", index, codec), nil
	}
	return publishEventsPipeline(conn, "RPUSH", index, codec), nil
}

func makePublishPUBLISH(conn redis.Conn, index string, codec codec.Codec) (publishFn, error) {
	return publishEventsPipeline(conn, "PUBLISH", index, codec), nil
}

func publishEventsBulk(conn redis.Conn, key outil.Selector, command string, index string, codec codec.Codec) publishFn {
	// XXX: requires key.IsConst() == true
	dest, _ := key.Select(&beat.Event{Fields: common.MapStr{}})
	return func(_ outil.Selector, data []publisher.Event) ([]publisher.Event, error) {
		args := make([]interface{}, 1, len(data)+1)
		args[0] = dest

		data, args = serializeEvents(args, 1, data, index, codec)
		if (len(args) - 1) == 0 {
			return nil, nil
		}

		// RPUSH returns total length of list -> fail and retry all on error
		_, err := conn.Do(command, args...)
		if err != nil {
			logp.Err("Failed to %v to redis list with %v", command, err)
			return data, err

		}
		ackedEvents.Add(int64(len(data)))
		outputs.AckedEvents.Add(int64(len(data)))

		return nil, nil
	}
}

func publishEventsPipeline(conn redis.Conn, command string, index string, codec codec.Codec) publishFn {
	return func(key outil.Selector, data []publisher.Event) ([]publisher.Event, error) {
		var okEvents []publisher.Event
		serialized := make([]interface{}, 0, len(data))
		okEvents, serialized = serializeEvents(serialized, 0, data, index, codec)
		if len(serialized) == 0 {
			return nil, nil
		}

		data = okEvents[:0]
		for i, serializedEvent := range serialized {
			eventKey, err := key.Select(&okEvents[i].Content)
			if err != nil {
				logp.Err("Failed to set redis key: %v", err)
				continue
			}

			data = append(data, okEvents[i])
			if err := conn.Send(command, eventKey, serializedEvent); err != nil {
				logp.Err("Failed to execute %v: %v", command, err)
				return okEvents, err
			}
		}

		if err := conn.Flush(); err != nil {
			return data, err
		}

		failed := data[:0]
		var lastErr error
		for i := range serialized {
			_, err := conn.Receive()
			if err != nil {
				if _, ok := err.(redis.Error); ok {
					logp.Err("Failed to %v event to list with %v",
						command, err)
					failed = append(failed, data[i])
					lastErr = err
				} else {
					logp.Err("Failed to %v multiple events to list with %v",
						command, err)
					failed = append(failed, data[i:]...)
					lastErr = err
					break
				}
			}
		}
		ackedEvents.Add(int64(len(okEvents) - len(failed)))
		outputs.AckedEvents.Add(int64(len(okEvents) - len(failed)))
		eventsNotAcked.Add(int64(len(failed)))
		return failed, lastErr
	}
}

func serializeEvents(
	to []interface{},
	i int,
	data []publisher.Event,
	index string,
	codec codec.Codec,
) ([]publisher.Event, []interface{}) {

	succeeded := data
	for _, d := range data {
		serializedEvent, err := codec.Encode(index, &d.Content)
		if err != nil {
			goto failLoop
		}

		buf := make([]byte, len(serializedEvent))
		copy(buf, serializedEvent)
		to = append(to, buf)
		i++
	}
	return succeeded, to

failLoop:
	succeeded = data[:i]
	rest := data[i+1:]
	for _, d := range rest {
		serializedEvent, err := codec.Encode(index, &d.Content)
		if err != nil {
			i++
			continue
		}

		buf := make([]byte, len(serializedEvent))
		copy(buf, serializedEvent)
		to = append(to, buf)
		i++
	}

	return succeeded, to
}
