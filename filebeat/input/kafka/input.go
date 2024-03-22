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

package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/elastic/beats/v7/libbeat/common/atomic"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/Shopify/sarama"

	input "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/acker"
	"github.com/elastic/beats/v7/libbeat/common/backoff"
	"github.com/elastic/beats/v7/libbeat/common/kafka"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/reader"
	"github.com/elastic/beats/v7/libbeat/reader/parser"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

const pluginName = "kafka"

// Plugin creates a new filestream input plugin for creating a stateful input.
func Plugin() input.Plugin {
	return input.Plugin{
		Name:       pluginName,
		Stability:  feature.Stable,
		Deprecated: false,
		Info:       "Kafka input",
		Doc:        "The Kafka input consumes events from topics by connecting to the configured kafka brokers",
		Manager:    input.ConfigureWith(configure),
	}
}

func configure(cfg *conf.C) (input.Input, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	saramaConfig, err := newSaramaConfig(config)
	if err != nil {
		return nil, fmt.Errorf("initializing Sarama config: %w", err)
	}
	return NewInput(config, saramaConfig)
}

func NewInput(config kafkaInputConfig, saramaConfig *sarama.Config) (*kafkaInput, error) {
	return &kafkaInput{config: config, saramaConfig: saramaConfig}, nil
}

type kafkaInput struct {
	config          kafkaInputConfig
	saramaConfig    *sarama.Config
	saramaWaitGroup sync.WaitGroup // indicates a sarama consumer group is active
}

func (input *kafkaInput) Name() string { return pluginName }

func (input *kafkaInput) Test(ctx input.TestContext) error {
	client, err := sarama.NewClient(input.config.Hosts, input.saramaConfig)
	if err != nil {
		ctx.Logger.Error(err)
	}
	topics, err := client.Topics()
	if err != nil {
		ctx.Logger.Error(err)
	}

	var missingTopics []string
	for _, neededTopic := range input.config.Topics {
		if !contains(topics, neededTopic) {
			missingTopics = append(missingTopics, neededTopic)
		}
	}

	if len(missingTopics) > 0 {
		return fmt.Errorf("Of configured topics %v, topics: %v are not in available topics %v", input.config.Topics, missingTopics, topics)
	}

	return nil
}

func (input *kafkaInput) Run(ctx input.Context, pipeline beat.Pipeline) error {
	log := ctx.Logger.Named("kafka input").With("hosts", input.config.Hosts)

	client, err := pipeline.ConnectWith(beat.ClientConfig{
		EventListener: acker.ConnectionOnly(
			acker.EventPrivateReporter(func(_ int, events []interface{}) {
				for _, event := range events {
					if meta, ok := event.(eventMeta); ok {
						meta.ackHandler()
					}
				}
			}),
		),
		// CloseRef:  ctx.Cancelation,
		WaitClose: input.config.WaitClose,
	})
	if err != nil {
		return err
	}
	defer client.Close()

	log.Info("Starting Kafka input")
	defer log.Info("Kafka input stopped")

	// Sarama uses standard go contexts to control cancellation, so we need
	// to wrap our input context channel in that interface.
	goContext := doneChannelContext(ctx)

	// If the consumer fails to connect, we use exponential backoff with
	// jitter up to 8 * the initial backoff interval.
	connectDelay := backoff.NewEqualJitterBackoff(
		ctx.Cancelation.Done(),
		input.config.ConnectBackoff,
		8*input.config.ConnectBackoff,
	)

	for goContext.Err() == nil {
		// Connect to Kafka with a new consumer group.
		consumerGroup, err := sarama.NewConsumerGroup(
			input.config.Hosts,
			input.config.GroupID,
			input.saramaConfig,
		)
		if err != nil {
			log.Errorw("Error initializing kafka consumer group", "error", err)
			connectDelay.Wait()
			continue
		}
		// We've successfully connected, reset the backoff timer.
		connectDelay.Reset()

		// We have a connected consumer group now, try to start the main event
		// loop by calling Consume (which starts an asynchronous consumer).
		// In an ideal run, this function never returns until shutdown; if it
		// does, it means the errors have been logged and the consumer group
		// has been closed, so we try creating a new one in the next iteration.
		input.runConsumerGroup(log, client, goContext, consumerGroup)
	}

	if errors.Is(ctx.Cancelation.Err(), context.Canceled) {
		return nil
	} else {
		return ctx.Cancelation.Err()
	}
}

// Stop doesn't need to do anything because the kafka consumer group and the
// input's outlet both have a context based on input.context.Done and will
// shut themselves down, since the done channel is already closed as part of
// the shutdown process in Runner.Stop().
func (input *kafkaInput) Stop() {
}

// Wait should shut down the input and wait for it to complete, however (see
// Stop above) we don't need to take actions to shut down as long as the
// input.config.Done channel is closed, so we just make a (currently no-op)
// call to Stop() and then wait for sarama to signal completion.
func (input *kafkaInput) Wait() {
	input.Stop()
	// Wait for sarama to shut down
	input.saramaWaitGroup.Wait()
}

func (input *kafkaInput) runConsumerGroup(log *logp.Logger, client beat.Client, context context.Context, consumerGroup sarama.ConsumerGroup) {
	handler := &groupHandler{
		version: input.config.Version,
		client:  client,
		parsers: input.config.Parsers,
		// expandEventListFromField will be assigned the configuration option expand_event_list_from_field
		expandEventListFromField: input.config.ExpandEventListFromField,
		log:                      log,
	}

	input.saramaWaitGroup.Add(1)
	defer func() {
		consumerGroup.Close()
		input.saramaWaitGroup.Done()
	}()

	// Listen asynchronously to any errors during the consume process
	go func() {
		for err := range consumerGroup.Errors() {
			log.Errorw("Error reading from kafka", "error", err)
		}
	}()

	err := consumerGroup.Consume(context, input.config.Topics, handler)
	if err != nil {
		log.Errorw("Kafka consume error", "error", err)
	}
}

// The metadata attached to incoming events, so they can be ACKed once they've
// been successfully sent.
type eventMeta struct {
	ackHandler func()
}

func arrayForKafkaHeaders(headers []*sarama.RecordHeader) []string {
	array := []string{}
	for _, header := range headers {
		// Rather than indexing headers in the same object structure Kafka does
		// (which would give maximal fidelity, but would be effectively unsearchable
		// in elasticsearch and kibana) we compromise by serializing them all as
		// strings in the form "<key>: <value>". For this we need to mask
		// occurrences of ":" in the original key, which we expect to be uncommon.
		// We may consider another approach in the future when it's more clear what
		// the most common use cases are.
		key := strings.ReplaceAll(string(header.Key), ":", "_")
		value := string(header.Value)
		array = append(array, fmt.Sprintf("%s: %s", key, value))
	}
	return array
}

// A barebones implementation of context.Context wrapped around the done
// channels that are more common in the beats codebase.
// TODO(faec): Generalize this to a common utility in a shared library
// (https://github.com/elastic/beats/issues/13125).
type channelCtx struct {
	ctx input.Context
}

func doneChannelContext(ctx input.Context) context.Context {
	return channelCtx{ctx}
}

func (c channelCtx) Deadline() (deadline time.Time, ok bool) {
	//nolint:nakedret // omitting the return gives a build error
	return
}

func (c channelCtx) Done() <-chan struct{} {
	return c.ctx.Cancelation.Done()
}

func (c channelCtx) Err() error {
	return c.ctx.Cancelation.Err()
}
func (c channelCtx) Value(_ interface{}) interface{} { return nil }

// The group handler for the sarama consumer group interface. In addition to
// providing the basic consumption callbacks needed by sarama, groupHandler is
// also currently responsible for marshalling kafka messages into beat.Event,
// and passing ACKs from the output channel back to the kafka cluster.
type groupHandler struct {
	sync.Mutex
	version kafka.Version
	session sarama.ConsumerGroupSession
	client  beat.Client
	parsers parser.Config
	// if the fileset using this input expects to receive multiple messages bundled under a specific field then this value is assigned
	// ex. in this case are the azure fielsets where the events are found under the json object "records"
	expandEventListFromField string // TODO
	log                      *logp.Logger
}

func (h *groupHandler) Setup(session sarama.ConsumerGroupSession) error {
	h.Lock()
	h.session = session
	h.Unlock()
	return nil
}

func (h *groupHandler) Cleanup(_ sarama.ConsumerGroupSession) error {
	h.Lock()
	h.session = nil
	h.Unlock()
	return nil
}

// ack informs the kafka cluster that this message has been consumed. Called
// from the input's ACKEvents handler.
func (h *groupHandler) ack(message *sarama.ConsumerMessage) {
	h.Lock()
	defer h.Unlock()
	if h.session != nil {
		h.session.MarkMessage(message, "")
	}
}

func (h *groupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	reader := h.createReader(claim)
	parser := h.parsers.Create(reader)
	for h.session.Context().Err() == nil {
		message, err := parser.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		h.client.Publish(beat.Event{
			Timestamp: message.Ts,
			Meta:      message.Meta,
			Fields:    message.Fields,
			Private:   message.Private,
		})
	}
	return nil
}

func (h *groupHandler) createReader(claim sarama.ConsumerGroupClaim) reader.Reader {
	if h.expandEventListFromField != "" {
		return &listFromFieldReader{
			claim:        claim,
			groupHandler: h,
			field:        h.expandEventListFromField,
			log:          h.log,
		}
	}
	return &recordReader{
		claim:        claim,
		groupHandler: h,
		log:          h.log,
	}
}

type recordReader struct {
	claim        sarama.ConsumerGroupClaim
	groupHandler *groupHandler
	log          *logp.Logger
}

func (m *recordReader) Close() error {
	return nil
}

func (m *recordReader) Next() (reader.Message, error) {
	msg, ok := <-m.claim.Messages()
	if !ok {
		return reader.Message{}, io.EOF
	}

	timestamp, kafkaFields := composeEventMetadata(m.claim, m.groupHandler, msg)
	ackHandler := func() {
		m.groupHandler.ack(msg)
	}
	return composeMessage(timestamp, msg.Value, kafkaFields, ackHandler), nil
}

type listFromFieldReader struct {
	claim        sarama.ConsumerGroupClaim
	groupHandler *groupHandler
	buffer       []reader.Message
	field        string
	log          *logp.Logger
}

func (l *listFromFieldReader) Close() error {
	return nil
}

func (l *listFromFieldReader) Next() (reader.Message, error) {
	if len(l.buffer) != 0 {
		return l.returnFromBuffer()
	}

	msg, ok := <-l.claim.Messages()
	if !ok {
		return reader.Message{}, io.EOF
	}

	timestamp, kafkaFields := composeEventMetadata(l.claim, l.groupHandler, msg)
	messages := l.parseMultipleMessages(msg.Value)

	neededAcks := atomic.MakeInt(len(messages))
	ackHandler := func() {
		if neededAcks.Dec() == 0 {
			l.groupHandler.ack(msg)
		}
	}
	for _, message := range messages {
		newBuffer := append(l.buffer, composeMessage(timestamp, []byte(message), kafkaFields, ackHandler))
		l.buffer = newBuffer
	}

	return l.returnFromBuffer()
}

func (l *listFromFieldReader) returnFromBuffer() (reader.Message, error) {
	next := l.buffer[0]
	newBuffer := l.buffer[1:]
	l.buffer = newBuffer
	return next, nil
}

// parseMultipleMessages will try to split the message into multiple ones based on the group field provided by the configuration
func (l *listFromFieldReader) parseMultipleMessages(bMessage []byte) []string {
	var obj map[string][]interface{}
	err := json.Unmarshal(bMessage, &obj)
	if err != nil {
		l.log.Errorw(fmt.Sprintf("Kafka desirializing multiple messages using the group object %s", l.field), "error", err)
		return []string{}
	}
	var messages []string
	for _, ms := range obj[l.field] {
		js, err := json.Marshal(ms)
		if err == nil {
			messages = append(messages, string(js))
		} else {
			l.log.Errorw(fmt.Sprintf("Kafka serializing message %s", ms), "error", err)
		}
	}
	return messages
}

func composeEventMetadata(claim sarama.ConsumerGroupClaim, handler *groupHandler, msg *sarama.ConsumerMessage) (time.Time, mapstr.M) {
	timestamp := time.Now()
	kafkaFields := mapstr.M{
		"topic":     claim.Topic(),
		"partition": claim.Partition(),
		"offset":    msg.Offset,
		"key":       string(msg.Key),
	}

	version, versionOk := handler.version.Get()
	if versionOk && version.IsAtLeast(sarama.V0_10_0_0) {
		timestamp = msg.Timestamp
		if !msg.BlockTimestamp.IsZero() {
			kafkaFields["block_timestamp"] = msg.BlockTimestamp
		}
	}
	if versionOk && version.IsAtLeast(sarama.V0_11_0_0) {
		kafkaFields["headers"] = arrayForKafkaHeaders(msg.Headers)
	}
	return timestamp, kafkaFields
}

func composeMessage(timestamp time.Time, content []byte, kafkaFields mapstr.M, ackHandler func()) reader.Message {
	return reader.Message{
		Ts:      timestamp,
		Content: content,
		Fields: mapstr.M{
			"kafka":   kafkaFields,
			"message": string(content),
		},
		Private: eventMeta{
			ackHandler: ackHandler,
		},
	}
}

func contains(elements []string, element string) bool {
	for _, e := range elements {
		if e == element {
			return true
		}
	}
	return false
}
