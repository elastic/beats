package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/elastic/beats/filebeat/channel"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/pkg/errors"

	"github.com/Azure/azure-event-hubs-go"
	"github.com/Azure/azure-event-hubs-go/eph"
)

var eventHubConnector = ";EntityPath="
var expandEventListFromField = "records"


type azureInput struct {
	config       azureInputConfig
	context      input.Context
	outlet       channel.Outleter
	log          *logp.Logger
	workerCtx    context.Context    // Worker goroutine context. It's cancelled when the input stops or the worker exits.
	workerCancel context.CancelFunc // Used to signal that the worker should stop.
	workerOnce   sync.Once          // Guarantees that the worker goroutine is only started once.
	workerWg     sync.WaitGroup     // Waits on worker goroutine.
}

func init() {
	err := input.Register("azure", NewInput)
	if err != nil {
		panic(err)
	}
}

// NewInput creates a new kafka input
func NewInput(
	cfg *common.Config,
	connector channel.Connector,
	inputContext input.Context,
) (input.Input, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, errors.Wrap(err, "reading kafka input config")
	}

	out, err := connector.ConnectWith(cfg, beat.ClientConfig{
		Processing: beat.ProcessingConfig{
			DynamicFields: inputContext.DynamicFields,
		},
	})
	if err != nil {
		return nil, err
	}

	inputCtx, cancelInputCtx := context.WithCancel(context.Background())
	go func() {
		defer cancelInputCtx()
		select {
		case <-inputContext.Done:
		case <-inputCtx.Done():
		}
	}()

	// If the input ever needs to be made restartable, then context would need
	// to be recreated with each restart.
	workerCtx, workerCancel := context.WithCancel(inputCtx)

	input := &azureInput{
		config:       config,
		log:          logp.NewLogger("azure input").With("connection string", config.ConnectionString),
		outlet:       out,
		context:      inputContext,
		workerCtx:    workerCtx,
		workerCancel: workerCancel,
	}

	input.log.Info("Initialized azure input.")
	return input, nil
}

// Run starts the input worker then returns. Only the first invocation
// will ever start the worker.
func (a *azureInput) Run() {
	a.workerOnce.Do(func() {
		a.workerWg.Add(1)
		go func() {
			a.log.Info("azure input worker has started.")
			defer a.log.Info("azure input worker has stopped.")
			defer a.workerWg.Done()
			defer a.workerCancel()
			if err := a.run(); err != nil {
				a.log.Error(err)
				return
			}
		}()
	})
}

// Run runs the input
func (a *azureInput) run() error {
	ctx, cancel := context.WithCancel(a.workerCtx)
	defer cancel()

	leaserCheckpointer := NewMemoryLeaserCheckpointer(eph.DefaultLeaseDuration, new(SharedStore))
	if leaserCheckpointer == nil {
		// handle err
	}

	processor, err := eph.NewFromConnectionString(ctx, a.config.ConnectionString + eventHubConnector + a.config.EventHubName, leaserCheckpointer, leaserCheckpointer)
	if err != nil {
		return err
	}

	// register a message handler -- many can be registered
	handlerID, err := processor.RegisterHandler(ctx,
		func(c context.Context, e *eventhub.Event) error {
			return a.processEvents(e.Data)
		})
	if err != nil {
		return err
	}

	a.log.Info("handler id: %q is running\n", handlerID)

	// unregister a handler to stop that handler from receiving events
	// processor.UnregisterHandler(ctx, handleID)

	// start handling messages from all of the partitions balancing across multiple consumers
	err = processor.StartNonBlocking(ctx)
	if err != nil {
		return err
	}

	// Wait for a signal to quit:
	//signalChan := make(chan os.Signal, 1)
	//signal.Notify(signalChan, os.Interrupt, os.Kill)
	//<-signalChan

	err = processor.Close(ctx)
	if err != nil {
		return err
	}
	return nil
}

// Stop stops TCP server
func (a *azureInput) Stop() {
	a.workerCancel()
	a.workerWg.Wait()
}

// Wait stop the current server
func (a *azureInput) Wait() {
	a.Stop()
}

func (a *azureInput) processEvents(raw []byte) error {
	events := a.createEvents(raw)
	for _, event := range events {
		ok := a.outlet.OnEvent(event)
		if !ok {
			return errors.New("function OnEvent returned false - ")
		}
	}
	return nil
}

func (a *azureInput) createEvents(message []byte) []beat.Event {
	timestamp := time.Now()
	var events []beat.Event
	messages := a.parseMultipleMessages(message)
	for _, msg := range messages {
		event := beat.Event{
			Timestamp: timestamp,
			Fields: common.MapStr{
				"message": msg,
			},
		}
		events = append(events, event)

	}
	return events
}

// parseMultipleMessages will try to split the message into multiple ones based on the group field provided by the configuration
func (a *azureInput) parseMultipleMessages(bMessage []byte) []string {
	var obj map[string][]interface{}
	err := json.Unmarshal(bMessage, &obj)
	if err != nil {
		a.log.Errorw(fmt.Sprintf("deserializing multiple messages using the group object `records`"), "error", err)
		return []string{}
	}
	var messages []string
	if len(obj[expandEventListFromField]) > 0 {
		for _, ms := range obj[expandEventListFromField] {
			js, err := json.Marshal(ms)
			if err == nil {
				messages = append(messages, string(js))
			} else {
				a.log.Errorw(fmt.Sprintf("serializing message %s", ms), "error", err)
			}
		}
	}
	return messages
}
