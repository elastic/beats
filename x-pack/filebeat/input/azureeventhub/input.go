package azureeventhub

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Azure/azure-event-hubs-go/v2/persist"
	"sync"
	"time"

	"github.com/elastic/beats/filebeat/channel"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/pkg/errors"

	"github.com/Azure/azure-event-hubs-go/v2"
	"github.com/Azure/azure-event-hubs-go/v2/eph"
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
	processor    *eph.EventProcessorHost
}

const (
	inputName = "azure-eventhub"
)

func init() {
	err := input.Register(inputName, NewInput)
	if err != nil {
		panic(errors.Wrapf(err, "failed to register %v input", inputName))
	}
}

// NewInput creates a new kafka input
func NewInput(
	cfg *common.Config,
	connector channel.Connector,
	inputContext input.Context,
) (input.Input, error) {
	var config azureInputConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, errors.Wrap(err, "reading azure eventhub input config")
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
		log:          logp.NewLogger("azure eventhub input").With("connection string", config.ConnectionString),
		outlet:       out,
		context:      inputContext,
		workerCtx:    workerCtx,
		workerCancel: workerCancel,
	}

	input.log.Info("Initialized azure eventhub input.")
	return input, nil
}

// Run starts the input worker then returns. Only the first invocation
// will ever start the worker.
func (a *azureInput) Run() {
	a.workerOnce.Do(func() {
		a.workerWg.Add(1)
		go func() {
			a.log.Info("azure eventhub input worker has started.")
			defer a.log.Info("azure eventhub input worker has stopped.")
			defer a.workerWg.Done()
			defer a.workerCancel()
			var err error
			if a.config.EPHEnabled {
				err = a.runWithEPH()
			} else {
				err = a.run()
			}
			if err != nil {
				a.log.Error(err)
				return
			}
		}()
	})
}

// run runs the input
func (a *azureInput) run() error {
	persister, err := persist.NewFilePersister("C:\\Users\\Mariana\\Downloads\\New folder (2)\\Folder nou")
	if err != nil {
		// handle err
	}
	hub, err := eventhub.NewHubFromConnectionString(a.config.ConnectionString+eventHubConnector+a.config.EventHubName, eventhub.HubWithOffsetPersistence(persister))
	ctx, cancel := context.WithCancel(a.workerCtx)
	defer cancel()
	if err != nil {
		return err
	}

	// listen to each partition of the Event Hub
	runtimeInfo, err := hub.GetRuntimeInformation(ctx)
	if err != nil {
		return err
	}

	var receiveOption eventhub.ReceiveOption
	if a.config.ConsumerGroup != "" {
		receiveOption = eventhub.ReceiveWithConsumerGroup(a.config.ConsumerGroup)
	}
	for _, partitionID := range runtimeInfo.PartitionIDs {
		// Start receiving messages
		//
		// Receive blocks while attempting to connect to hub, then runs until listenerHandle.Close() is called
		// <- listenerHandle.Done() signals listener has stopped
		// listenerHandle.Err() provides the last error the receiver encountered
		listenerHandle, err := hub.Receive(
			ctx,
			partitionID,
			func(c context.Context, event *eventhub.Event) error {
				return a.processEvents(event.Data)
			},
			receiveOption)
		_ = listenerHandle
		if err != nil {
			return err
		}
	}

	// Wait for a signal to quit:

	//err = hub.Close(context.Background())
	//if err != nil {
	//	fmt.Println(err)
	//}

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
	// timestamp temp disabled as the event date is applied for now, will be replaced
	//timestamp := time.Now()
	var events []beat.Event
	messages := a.parseMultipleMessages(message)
	for _, msg := range messages {
		for key, value := range msg {
			event := beat.Event{
				Timestamp: key,
				Fields: common.MapStr{
					"message": value,
				},
			}
			events = append(events, event)
		}
	}
	return events
}

// parseMultipleMessages will try to split the message into multiple ones based on the group field provided by the configuration
func (a *azureInput) parseMultipleMessages(bMessage []byte) []map[time.Time]string {
	var obj map[string][]interface{}
	err := json.Unmarshal(bMessage, &obj)
	if err != nil {
		a.log.Errorw(fmt.Sprintf("deserializing multiple messages using the group object `records`"), "error", err)
		return []map[time.Time]string{}
	}
	var messages []map[time.Time]string
	//var messages []string
	if len(obj[expandEventListFromField]) > 0 {
		for _, ms := range obj[expandEventListFromField] {
			js, err := json.Marshal(ms)
			if err == nil {
				// temporary implementation, retrieving the date in order to verify events are matching
				timeInter := ms.(map[string]interface{})["time"]
				date, _ := time.Parse(time.RFC3339, timeInter.(string))
				item := make(map[time.Time]string)
				item[date] = string(js)
				messages = append(messages, item)
			} else {
				a.log.Errorw(fmt.Sprintf("serializing message %s", ms), "error", err)
			}
		}
	}
	return messages
}
