// Package eph provides functionality for balancing load of Event Hub receivers through scheduling receivers across
// processes and machines.
package eph

//	MIT License
//
//	Copyright (c) Microsoft Corporation. All rights reserved.
//
//	Permission is hereby granted, free of charge, to any person obtaining a copy
//	of this software and associated documentation files (the "Software"), to deal
//	in the Software without restriction, including without limitation the rights
//	to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
//	copies of the Software, and to permit persons to whom the Software is
//	furnished to do so, subject to the following conditions:
//
//	The above copyright notice and this permission notice shall be included in all
//	copies or substantial portions of the Software.
//
//	THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
//	IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
//	FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
//	AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
//	LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
//	OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
//	SOFTWARE

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/Azure/azure-amqp-common-go/v3/auth"
	"github.com/Azure/azure-amqp-common-go/v3/conn"
	"github.com/Azure/azure-amqp-common-go/v3/sas"
	"github.com/Azure/azure-amqp-common-go/v3/uuid"
	"github.com/Azure/azure-event-hubs-go/v3"
	"github.com/Azure/azure-event-hubs-go/v3/persist"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/devigned/tab"
)

const (
	banner = `
    ______                 __  __  __      __
   / ____/   _____  ____  / /_/ / / /_  __/ /_  _____
  / __/ | | / / _ \/ __ \/ __/ /_/ / / / / __ \/ ___/
 / /___ | |/ /  __/ / / / /_/ __  / /_/ / /_/ (__  )
/_____/ |___/\___/_/ /_/\__/_/ /_/\__,_/_.___/____/

`

	exitPrompt = "=> processing events, ctrl+c to exit"
)

type (
	// EventProcessorHost provides functionality for coordinating and balancing load across multiple Event Hub partitions
	EventProcessorHost struct {
		namespace           string
		hubName             string
		name                string
		consumerGroup       string
		tokenProvider       auth.TokenProvider
		client              *eventhub.Hub
		leaser              Leaser
		checkpointer        Checkpointer
		scheduler           *scheduler
		handlers            map[string]eventhub.Handler
		hostMu              sync.Mutex
		handlersMu          sync.Mutex
		partitionIDs        []string
		noBanner            bool
		webSocketConnection bool
		env                 *azure.Environment
	}

	// EventProcessorHostOption provides configuration options for an EventProcessorHost
	EventProcessorHostOption func(host *EventProcessorHost) error

	// Receiver provides the ability to handle Event Hub events
	Receiver interface {
		Receive(ctx context.Context, handler eventhub.Handler) (close func() error, err error)
	}

	checkpointPersister struct {
		checkpointer Checkpointer
	}

	// HandlerID is a UUID in string format that identifies a registered handler
	HandlerID string
)

// WithNoBanner will configure an EventProcessorHost to not output the banner upon start
func WithNoBanner() EventProcessorHostOption {
	return func(host *EventProcessorHost) error {
		host.noBanner = true
		return nil
	}
}

// WithConsumerGroup will configure an EventProcessorHost to a specific consumer group
func WithConsumerGroup(consumerGroup string) EventProcessorHostOption {
	return func(host *EventProcessorHost) error {
		host.consumerGroup = consumerGroup
		return nil
	}
}

// WithEnvironment will configure an EventProcessorHost to use the specified Azure Environment
func WithEnvironment(env azure.Environment) EventProcessorHostOption {
	return func(host *EventProcessorHost) error {
		host.env = &env
		return nil
	}
}

// WithWebSocketConnection will configure an EventProcessorHost to use websockets
func WithWebSocketConnection() EventProcessorHostOption {
	return func(host *EventProcessorHost) error {
		host.webSocketConnection = true
		return nil
	}
}

// NewFromConnectionString builds a new Event Processor Host from an Event Hub connection string which can be found in
// the Azure portal
func NewFromConnectionString(ctx context.Context, connStr string, leaser Leaser, checkpointer Checkpointer, opts ...EventProcessorHostOption) (*EventProcessorHost, error) {
	span, ctx := startConsumerSpanFromContext(ctx, "eph.NewFromConnectionString")
	defer span.End()

	hostName, err := uuid.NewV4()
	if err != nil {
		tab.For(ctx).Error(err)
		return nil, err
	}

	parsed, err := conn.ParsedConnectionFromStr(connStr)
	if err != nil {
		tab.For(ctx).Error(err)
		return nil, err
	}

	tokenProvider, err := sas.NewTokenProvider(sas.TokenProviderWithKey(parsed.KeyName, parsed.Key))
	if err != nil {
		tab.For(ctx).Error(err)
		return nil, err
	}

	host := &EventProcessorHost{
		namespace:     parsed.Namespace,
		name:          hostName.String(),
		hubName:       parsed.HubName,
		tokenProvider: tokenProvider,
		handlers:      make(map[string]eventhub.Handler),
		leaser:        leaser,
		checkpointer:  checkpointer,
		noBanner:      false,
	}

	for _, opt := range opts {
		err := opt(host)
		if err != nil {
			return nil, err
		}
	}

	persister := checkpointPersister{checkpointer: checkpointer}
	hubOpts := []eventhub.HubOption{eventhub.HubWithOffsetPersistence(persister)}
	if host.env != nil {
		hubOpts = append(hubOpts, eventhub.HubWithEnvironment(*host.env))
	}

	if host.webSocketConnection {
		hubOpts = append(hubOpts, eventhub.HubWithWebSocketConnection())
	}

	client, err := eventhub.NewHubFromConnectionString(connStr, hubOpts...)
	if err != nil {
		tab.For(ctx).Error(err)
		return nil, err
	}

	runtimeInfo, err := client.GetRuntimeInformation(ctx)
	if err != nil {
		tab.For(ctx).Error(err)
		return nil, err
	}

	host.client = client
	host.partitionIDs = runtimeInfo.PartitionIDs

	return host, nil
}

// New constructs a new instance of an EventHostProcessor
func New(ctx context.Context, namespace, hubName string, tokenProvider auth.TokenProvider, leaser Leaser, checkpointer Checkpointer, opts ...EventProcessorHostOption) (*EventProcessorHost, error) {
	span, ctx := startConsumerSpanFromContext(ctx, "eph.New")
	defer span.End()

	hostName, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}

	host := &EventProcessorHost{
		namespace:     namespace,
		name:          hostName.String(),
		hubName:       hubName,
		tokenProvider: tokenProvider,
		handlers:      make(map[string]eventhub.Handler),
		leaser:        leaser,
		checkpointer:  checkpointer,
		noBanner:      false,
	}

	for _, opt := range opts {
		err := opt(host)
		if err != nil {
			return nil, err
		}
	}

	persister := checkpointPersister{checkpointer: checkpointer}
	hubOpts := []eventhub.HubOption{eventhub.HubWithOffsetPersistence(persister)}
	if host.env != nil {
		hubOpts = append(hubOpts, eventhub.HubWithEnvironment(*host.env))
	}

	if host.webSocketConnection {
		hubOpts = append(hubOpts, eventhub.HubWithWebSocketConnection())
	}

	client, err := eventhub.NewHub(namespace, hubName, tokenProvider, hubOpts...)
	if err != nil {
		return nil, err
	}

	runtimeInfo, err := client.GetRuntimeInformation(ctx)
	if err != nil {
		return nil, err
	}

	host.client = client
	host.partitionIDs = runtimeInfo.PartitionIDs
	return host, nil
}

// RegisteredHandlerIDs will return the registered event handler IDs
func (h *EventProcessorHost) RegisteredHandlerIDs() []HandlerID {
	h.handlersMu.Lock()
	defer h.handlersMu.Unlock()

	ids := make([]HandlerID, len(h.handlers))
	count := 0
	for key := range h.handlers {
		ids[count] = HandlerID(key)
		count++
	}
	return ids
}

// RegisterHandler will register an event handler which will receive events after Start or StartNonBlocking is called
func (h *EventProcessorHost) RegisterHandler(ctx context.Context, handler eventhub.Handler) (HandlerID, error) {
	span, _ := startConsumerSpanFromContext(ctx, "eph.EventProcessorHost.RegisterHandler")
	defer span.End()

	h.handlersMu.Lock()
	defer h.handlersMu.Unlock()

	id, err := uuid.NewV4()
	if err != nil {
		return "", err
	}

	h.handlers[id.String()] = handler
	return HandlerID(id.String()), nil
}

// UnregisterHandler will remove an event handler from receiving events, and will close the EventProcessorHost if it is
// the last handler registered.
func (h *EventProcessorHost) UnregisterHandler(ctx context.Context, id HandlerID) {
	span, ctx := startConsumerSpanFromContext(ctx, "eph.EventProcessorHost.UnregisterHandler")
	defer span.End()

	h.handlersMu.Lock()
	defer h.handlersMu.Unlock()

	delete(h.handlers, string(id))

	if len(h.handlers) == 0 {
		if err := h.Close(ctx); err != nil {
			tab.For(ctx).Error(err)
		}
	}
}

// Start begins processing of messages for registered handlers on the EventHostProcessor. The call is blocking.
func (h *EventProcessorHost) Start(ctx context.Context) error {
	span, ctx := startConsumerSpanFromContext(ctx, "eph.EventProcessorHost.Start")
	defer span.End()

	if !h.noBanner {
		fmt.Print(banner)
		fmt.Println(exitPrompt)
	}

	if len(h.handlers) == 0 {
		return errors.New("no handlers have been registered; call RegisterHandler to setup an event handler")
	}

	if err := h.setup(ctx); err != nil {
		return err
	}

	go func() {
		span := tab.FromContext(ctx)
		ctx := tab.NewContext(context.Background(), span)
		h.scheduler.Run(ctx)
	}()

	// Wait for a signal to quit:
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan
	return h.Close(ctx)
}

// StartNonBlocking begins processing of messages for registered handlers
func (h *EventProcessorHost) StartNonBlocking(ctx context.Context) error {
	span, ctx := startConsumerSpanFromContext(ctx, "eph.EventProcessorHost.StartNonBlocking")
	defer span.End()

	if !h.noBanner {
		fmt.Print(banner)
	}

	if err := h.setup(ctx); err != nil {
		return err
	}

	go func() {
		span := tab.FromContext(ctx)
		ctx := tab.NewContext(context.Background(), span)
		h.scheduler.Run(ctx)
	}()

	return nil
}

// GetName returns the name of the EventProcessorHost
func (h *EventProcessorHost) GetName() string {
	return h.name
}

// GetPartitionIDs fetches the partition IDs for the Event Hub
func (h *EventProcessorHost) GetPartitionIDs() []string {
	return h.partitionIDs
}

// PartitionIDsBeingProcessed returns the partition IDs currently receiving messages
func (h *EventProcessorHost) PartitionIDsBeingProcessed() []string {
	return h.scheduler.getPartitionIDsBeingProcessed()
}

// Close stops the EventHostProcessor from processing messages
func (h *EventProcessorHost) Close(ctx context.Context) error {
	if !h.noBanner {
		fmt.Println("shutting down...")
	}
	if h.scheduler != nil {
		if err := h.scheduler.Stop(ctx); err != nil {
			if h.client != nil {
				_ = h.client.Close(ctx)
			}
			return err
		}
	}

	if h.leaser != nil {
		_ = h.leaser.Close()
	}

	if h.checkpointer != nil {
		_ = h.checkpointer.Close()
	}

	return h.client.Close(ctx)
}

func (h *EventProcessorHost) setup(ctx context.Context) error {
	h.hostMu.Lock()
	defer h.hostMu.Unlock()
	span, ctx := startConsumerSpanFromContext(ctx, "eph.EventProcessorHost.setup")
	defer span.End()

	if h.scheduler == nil {
		h.leaser.SetEventHostProcessor(h)
		h.checkpointer.SetEventHostProcessor(h)
		if err := h.leaser.EnsureStore(ctx); err != nil {
			return err
		}

		if err := h.checkpointer.EnsureStore(ctx); err != nil {
			return err
		}

		scheduler := newScheduler(h)

		for _, partitionID := range h.partitionIDs {
			h.leaser.EnsureLease(ctx, partitionID)
			h.checkpointer.EnsureCheckpoint(ctx, partitionID)
		}

		h.scheduler = scheduler
	}
	return nil
}

func (h *EventProcessorHost) compositeHandlers() eventhub.Handler {
	return func(ctx context.Context, event *eventhub.Event) error {
		span, ctx := startConsumerSpanFromContext(ctx, "eph.EventProcessorHost.compositeHandlers")
		defer span.End()

		h.handlersMu.Lock()
		defer h.handlersMu.Unlock()

		// we accept that this will contain any of the possible len(h.handlers) errors
		// as it will be used to later decide of delivery is considered a failure
		// and NOT further inspected
		var lastError error

		wg := &sync.WaitGroup{}
		for _, handler := range h.handlers {
			wg.Add(1)
			go func(boundHandler eventhub.Handler) {
				defer wg.Done() // consider if panics should be cought here, too. Currently would crash process
				if err := boundHandler(ctx, event); err != nil {
					lastError = err
					tab.For(ctx).Error(err)
				}
			}(handler)
		}
		wg.Wait()
		return lastError
	}
}

func (c checkpointPersister) Write(namespace, name, consumerGroup, partitionID string, checkpoint persist.Checkpoint) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return c.checkpointer.UpdateCheckpoint(ctx, partitionID, checkpoint)
}

func (c checkpointPersister) Read(namespace, name, consumerGroup, partitionID string) (persist.Checkpoint, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return c.checkpointer.EnsureCheckpoint(ctx, partitionID)
}

func startConsumerSpanFromContext(ctx context.Context, operationName string) (tab.Spanner, context.Context) {
	ctx, span := tab.StartSpan(ctx, operationName)
	eventhub.ApplyComponentInfo(span)
	span.AddAttributes(tab.StringAttribute("span.kind", "consumer"))
	return span, ctx
}
