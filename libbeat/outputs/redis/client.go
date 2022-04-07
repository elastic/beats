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

package redis

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gomodule/redigo/redis"

	"github.com/elastic/beats/v8/libbeat/beat"
	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/common/transport"
	"github.com/elastic/beats/v8/libbeat/logp"
	"github.com/elastic/beats/v8/libbeat/outputs"
	"github.com/elastic/beats/v8/libbeat/outputs/codec"
	"github.com/elastic/beats/v8/libbeat/outputs/outil"
	"github.com/elastic/beats/v8/libbeat/publisher"
)

var (
	versionRegex = regexp.MustCompile(`redis_version:(\d+).(\d+)`)
)

type publishFn func(
	keys outil.Selector,
	data []publisher.Event,
) ([]publisher.Event, error)

type client struct {
	log *logp.Logger
	*transport.Client
	observer outputs.Observer
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

func newClient(
	tc *transport.Client,
	observer outputs.Observer,
	timeout time.Duration,
	pass string,
	db int, key outil.Selector, dt redisDataType,
	index string, codec codec.Codec,
) *client {
	return &client{
		log:      logp.NewLogger("redis"),
		Client:   tc,
		observer: observer,
		timeout:  timeout,
		password: pass,
		index:    strings.ToLower(index),
		db:       db,
		dataType: dt,
		key:      key,
		codec:    codec,
	}
}

func (c *client) Connect() error {
	c.log.Debug("connect")
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
		c.publish, err = c.makePublish(conn)
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
	c.log.Debug("close connection")
	return c.Client.Close()
}

func (c *client) Publish(_ context.Context, batch publisher.Batch) error {
	if c == nil {
		panic("no client")
	}
	if batch == nil {
		panic("no batch")
	}

	events := batch.Events()
	c.observer.NewBatch(len(events))
	rest, err := c.publish(c.key, events)
	if rest != nil {
		c.observer.Failed(len(rest))
		batch.RetryEvents(rest)
		return err
	}

	batch.ACK()
	return err
}

func (c *client) String() string {
	return "redis(" + c.Client.String() + ")"
}

func (c *client) makePublish(
	conn redis.Conn,
) (publishFn, error) {
	if c.dataType == redisChannelType {
		return c.makePublishPUBLISH(conn)
	}
	return c.makePublishRPUSH(conn)
}

func (c *client) makePublishRPUSH(conn redis.Conn) (publishFn, error) {
	if !c.key.IsConst() {
		// TODO: more clever bulk handling batching events with same key
		return c.publishEventsPipeline(conn, "RPUSH"), nil
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
		return c.publishEventsBulk(conn, "RPUSH"), nil
	}
	return c.publishEventsPipeline(conn, "RPUSH"), nil
}

func (c *client) makePublishPUBLISH(conn redis.Conn) (publishFn, error) {
	return c.publishEventsPipeline(conn, "PUBLISH"), nil
}

func (c *client) publishEventsBulk(conn redis.Conn, command string) publishFn {
	// XXX: requires key.IsConst() == true
	dest, _ := c.key.Select(&beat.Event{Fields: common.MapStr{}})
	return func(_ outil.Selector, data []publisher.Event) ([]publisher.Event, error) {
		args := make([]interface{}, 1, len(data)+1)
		args[0] = dest

		okEvents, args := serializeEvents(c.log, args, 1, data, c.index, c.codec)
		c.observer.Dropped(len(data) - len(okEvents))
		if (len(args) - 1) == 0 {
			return nil, nil
		}

		// RPUSH returns total length of list -> fail and retry all on error
		_, err := conn.Do(command, args...)
		if err != nil {
			c.log.Errorf("Failed to %v to redis list with: %+v", command, err)
			return okEvents, err

		}

		c.observer.Acked(len(okEvents))
		return nil, nil
	}
}

func (c *client) publishEventsPipeline(conn redis.Conn, command string) publishFn {
	return func(key outil.Selector, data []publisher.Event) ([]publisher.Event, error) {
		var okEvents []publisher.Event
		serialized := make([]interface{}, 0, len(data))
		okEvents, serialized = serializeEvents(c.log, serialized, 0, data, c.index, c.codec)
		c.observer.Dropped(len(data) - len(okEvents))
		if len(serialized) == 0 {
			return nil, nil
		}

		data = okEvents[:0]
		dropped := 0
		for i, serializedEvent := range serialized {
			eventKey, err := key.Select(&okEvents[i].Content)
			if err != nil {
				c.log.Errorf("Failed to set redis key: %+v", err)
				dropped++
				continue
			}

			data = append(data, okEvents[i])
			if err := conn.Send(command, eventKey, serializedEvent); err != nil {
				c.log.Errorf("Failed to execute %v: %+v", command, err)
				return okEvents, err
			}
		}
		c.observer.Dropped(dropped)

		if err := conn.Flush(); err != nil {
			return data, err
		}

		failed := data[:0]
		var lastErr error
		for i := range serialized {
			_, err := conn.Receive()
			if err != nil {
				if _, ok := err.(redis.Error); ok {
					c.log.Errorf("Failed to %v event to list with %+v",
						command, err)
					failed = append(failed, data[i])
					lastErr = err
				} else {
					c.log.Errorf("Failed to %v multiple events to list with %+v",
						command, err)
					failed = append(failed, data[i:]...)
					lastErr = err
					break
				}
			}
		}

		c.observer.Acked(len(okEvents) - len(failed))
		return failed, lastErr
	}
}

func serializeEvents(
	log *logp.Logger,
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
			log.Errorf("Encoding event failed with error: %+v", err)
			log.Debugf("Failed event: %v", d.Content)
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
			log.Errorf("Encoding event failed with error: %+v", err)
			log.Debugf("Failed event: %v", d.Content)
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
