package opa

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	"github.com/elastic/beats/v7/kubebeat/beater/bundle"
	"github.com/open-policy-agent/opa/logging"
	"github.com/open-policy-agent/opa/sdk"
	"github.com/sirupsen/logrus"
)

type Evaluator struct {
	bundleServer *http.Server
	opa          *sdk.OPA
}

func NewEvaluator() (*Evaluator, error) {
	server, err := bundle.CreateServer()
	if err != nil {
		return nil, err
	}

	// provide the OPA configuration which specifies
	// fetching policy bundles from the mock bundleServer
	// and logging decisions locally to the console
	config := []byte(fmt.Sprintf(bundle.Config, server.Addr))

	// create an instance of the OPA object
	opaLogger := newEvaluatorLogger()
	opa, err := sdk.New(context.Background(), sdk.Options{
		Config: bytes.NewReader(config),
		Logger: opaLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("fail to init opa: %s", err.Error())
	}

	return &Evaluator{
		opa:          opa,
		bundleServer: server,
	}, nil
}

func (e *Evaluator) Decision(input interface{}) (interface{}, error) {
	// get the named policy decision for the specified input
	result, err := e.opa.Decision(context.Background(), sdk.DecisionOptions{
		Path:  "main",
		Input: input,
	})
	if err != nil {
		return nil, err
	}

	return result.Result, nil
}

<<<<<<< HEAD:kubebeat/beater/evaluator.go
func (e *evaluator) Stop() {
	ctx := context.Background()
	e.opa.Stop(ctx)
	e.bundleServer.Shutdown(ctx)
}

func newEvaluatorLogger() logging.Logger {
	opaLogger := logging.New()
	opaLogger.SetFormatter(&logrus.JSONFormatter{})
	return opaLogger.WithFields(map[string]interface{}{"goroutine": "opa"})
=======
func (e *Evaluator) Stop() {
	e.opa.Stop(context.Background())
	e.bundleServer.Stop()
>>>>>>> master:kubebeat/opa/evaluator.go
}
