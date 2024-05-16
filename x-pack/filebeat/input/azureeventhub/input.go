// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix

package azureeventhub

import (
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/feature"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-concert/unison"
	"strings"
)

const (
	eventHubConnector        = ";EntityPath="
	expandEventListFromField = "records"
	inputName                = "azure-eventhub"
)

func Plugin(log *logp.Logger) v2.Plugin {
	return v2.Plugin{
		Name:       inputName,
		Stability:  feature.Stable,
		Deprecated: false,
		Info:       "Collect logs from Azure Event Hub",
		Manager: &eventHubInputManager{
			log: log,
		},
	}
}

type eventHubInputManager struct {
	log *logp.Logger
}

func (m *eventHubInputManager) Init(unison.Group) error {
	return nil
}

func (m *eventHubInputManager) Create(cfg *conf.C) (v2.Input, error) {
	var config azureInputConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, fmt.Errorf("reading %s input config: %w", inputName, err)
	}

	switch config.EngineVersion {
	case "v1":
		return newEventHubInputV1(config, m.log)
	case "v2":
		return newEventHubInputV2(config, m.log)
	default:
		return nil, fmt.Errorf("invalid azure-eventhub engine version: %s", config.EngineVersion)
	}

	//return &azureInput{
	//	config: config,
	//	log:    logp.NewLogger(fmt.Sprintf("%s input", inputName)).With("connection string", stripConnectionString(config.ConnectionString)),
	//}, nil
}

// func init() {
// 	err := input.Register(inputName, NewInput)
// 	if err != nil {
// 		panic(fmt.Errorf("failed to register %v input: %w", inputName, err))
// 	}
// }

// // configID computes a unique ID for the input configuration.
// //
// // It is used to identify the input in the registry and to detect
// // changes in the configuration.
// //
// // We will remove this function as we upgrade the input to the
// // v2 API (there is an ID in the v2 context).
// func configID(config *conf.C) (string, error) {
// 	var tmp struct {
// 		ID string `config:"id"`
// 	}
// 	if err := config.Unpack(&tmp); err != nil {
// 		return "", fmt.Errorf("error extracting ID: %w", err)
// 	}
// 	if tmp.ID != "" {
// 		return tmp.ID, nil
// 	}

// 	var h map[string]interface{}
// 	_ = config.Unpack(&h)
// 	id, err := hashstructure.Hash(h, nil)
// 	if err != nil {
// 		return "", fmt.Errorf("can not compute ID from configuration: %w", err)
// 	}

// 	return fmt.Sprintf("%16X", id), nil
// }

//// azureInput struct for the azure-eventhub input
//type azureInput struct {
//	config       azureInputConfig // azure-eventhub configuration
//	context      input.Context
//	outlet       channel.Outleter
//	log          *logp.Logger            // logging info and error messages
//	workerCtx    context.Context         // worker goroutine context. It's cancelled when the input stops or the worker exits.
//	workerCancel context.CancelFunc      // used to signal that the worker should stop.
//	workerOnce   sync.Once               // guarantees that the worker goroutine is only started once.
//	processor    *eph.EventProcessorHost // eph will be assigned if users have enabled the option
//	id           string                  // ID of the input; used to identify the input in the input metrics registry only, and will be removed once the input is migrated to v2.
//	metrics      *inputMetrics           // Metrics for the input.
//}

// // NewInput creates a new azure-eventhub input
// func NewInput(
// 	cfg *conf.C,
// 	connector channel.Connector,
// 	inputContext input.Context,
// ) (input.Input, error) {
// 	var config azureInputConfig
// 	if err := cfg.Unpack(&config); err != nil {
// 		return nil, fmt.Errorf("reading %s input config: %w", inputName, err)
// 	}

// 	// Since this is a v1 input, we need to set the ID manually.
// 	//
// 	// We need an ID to identify the input in the input metrics
// 	// registry.
// 	//
// 	// This is a temporary workaround until we migrate the input to v2.
// 	inputId, err := configID(cfg)
// 	if err != nil {
// 		return nil, err
// 	}

// 	inputCtx, cancelInputCtx := context.WithCancel(context.Background())
// 	go func() {
// 		defer cancelInputCtx()
// 		select {
// 		case <-inputContext.Done:
// 		case <-inputCtx.Done():
// 		}
// 	}()

// 	// If the input ever needs to be made restartable, then context would need
// 	// to be recreated with each restart.
// 	workerCtx, workerCancel := context.WithCancel(inputCtx)

// 	in := azureInput{
// 		id:           inputId,
// 		config:       config,
// 		log:          logp.NewLogger(fmt.Sprintf("%s input", inputName)).With("connection string", stripConnectionString(config.ConnectionString)),
// 		context:      inputContext,
// 		workerCtx:    workerCtx,
// 		workerCancel: workerCancel,
// 	}
// 	out, err := connector.Connect(cfg)
// 	if err != nil {
// 		return nil, err
// 	}
// 	in.outlet = out
// 	in.log.Infof("Initialized %s input.", inputName)

// 	return &in, nil
// }
//
//func (a *azureInput) Name() string {
//	return inputName
//}
//
//func (a *azureInput) Test(v2.TestContext) error {
//	return nil
//}
//
//// Run starts the `azure-eventhub` input and then returns.
////
//// The first invocation will start an input worker. All subsequent
//// invocations will be no-ops.
////
//// The input worker will continue fetching data from the event hub until
//// the input Runner calls the `Stop()` method.
//func (a *azureInput) Run(inputContext v2.Context, pipeline beat.Pipeline) error {
//	ctx := v2.GoContextFromCanceler(inputContext.Cancelation)
//
//	// `Run` is invoked periodically by the input Runner. The `sync.Once`
//	// guarantees that we only start the worker once during the first
//	// invocation.
//	// a.workerOnce.Do(func() {
//	a.log.Infof("%s input worker is starting.", inputName)
//
//	// We set up the metrics in the `Run()` method and tear them down
//	// in the `Stop()` method.
//	//
//	// The factory method `NewInput` is not a viable solution because
//	// the Runner invokes it during the configuration check without
//	// calling the `Stop()` function; this causes panics
//	// due to multiple metrics registrations.
//	a.metrics = newInputMetrics(inputContext.ID, nil)
//
//	err := a.runWithEPH()
//	if err != nil {
//		a.log.Errorw("error starting the input worker", "error", err)
//		return err
//	}
//	a.log.Infof("%s input worker has started.", inputName)
//	// })
//
//	for {
//		select {
//		case <-ctx.Done():
//			a.log.Infof("%s input worker is stopping.", inputName)
//			if a.processor != nil {
//				// Tells the processor to stop processing events and release all
//				// resources (like scheduler, leaser, checkpointer, and pipelineClient).
//				err := a.processor.Close(context.Background())
//				if err != nil {
//					a.log.Errorw("error while closing eventhostprocessor", "error", err)
//				}
//			}
//
//			if a.metrics != nil {
//				a.metrics.Close()
//			}
//
//			// a.workerCancel() // FIXME: is this needed?
//			a.log.Infof("%s input worker has stopped.", inputName)
//		}
//
//		break
//	}
//
//	return nil
//}
//
//// // Stop stops `azure-eventhub` input.
//// func (a *azureInput) Stop() {
//// 	a.log.Infof("%s input worker is stopping.", inputName)
//// 	if a.processor != nil {
//// 		// Tells the processor to stop processing events and release all
//// 		// resources (like scheduler, leaser, checkpointer, and pipelineClient).
//// 		err := a.processor.Close(context.Background())
//// 		if err != nil {
//// 			a.log.Errorw("error while closing eventhostprocessor", "error", err)
//// 		}
//// 	}
//
//// 	if a.metrics != nil {
//// 		a.metrics.Close()
//// 	}
//
//// 	a.workerCancel()
//// 	a.log.Infof("%s input worker has stopped.", inputName)
//// }
//
//// // Wait stop the current server
//// func (a *azureInput) Wait() {
//// 	a.Stop()
//// }

func createPipelineClient(pipeline beat.Pipeline) (beat.Client, error) {
	return pipeline.ConnectWith(beat.ClientConfig{
		Processing: beat.ProcessingConfig{
			// This input only produces events with basic types so normalization
			// is not required.
			EventNormalization: to.Ptr(false),
		},
	})
}

// Strip connection string to remove sensitive information
// A connection string should look like this:
// Endpoint=sb://dummynamespace.servicebus.windows.net/;SharedAccessKeyName=DummyAccessKeyName;SharedAccessKey=5dOntTRytoC24opYThisAsit3is2B+OGY1US/fuL3ly=
// This code will remove everything after ';' so key information is stripped
func stripConnectionString(c string) string {
	if parts := strings.SplitN(c, ";", 2); len(parts) == 2 {
		return parts[0]
	}

	// We actually expect the string to have the documented format
	// if we reach here something is wrong, so let's stay on the safe side
	return "(redacted)"
}
