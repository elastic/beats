package opa

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	"github.com/elastic/beats/v7/cloudbeat/beater/bundle"
	"github.com/open-policy-agent/opa/logging"
	"github.com/open-policy-agent/opa/sdk"
	"github.com/sirupsen/logrus"
)

type Evaluator struct {
	bundleServer *http.Server
	opa          *sdk.OPA
}

func NewEvaluator(ctx context.Context) (*Evaluator, error) {
	server, err := bundle.StartServer()
	if err != nil {
		return nil, err
	}

	// provide the OPA configuration which specifies
	// fetching policy bundles from the mock bundleServer
	// and logging decisions locally to the console
	config := []byte(fmt.Sprintf(bundle.Config, bundle.ServerAddress))

	// create an instance of the OPA object
	opaLogger := newEvaluatorLogger()
	opa, err := sdk.New(ctx, sdk.Options{
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

func (e *Evaluator) Decision(ctx context.Context, input interface{}) (interface{}, error) {
	// get the named policy decision for the specified input
	result, err := e.opa.Decision(ctx, sdk.DecisionOptions{
		Path:  "main",
		Input: input,
	})
	if err != nil {
		return nil, err
	}

	return result.Result, nil
}

func (e *Evaluator) Stop(ctx context.Context) {
	e.opa.Stop(ctx)
	e.bundleServer.Shutdown(ctx)
}

func newEvaluatorLogger() logging.Logger {
	opaLogger := logging.New()
	opaLogger.SetFormatter(&logrus.JSONFormatter{})
	return opaLogger.WithFields(map[string]interface{}{"goroutine": "opa"})
}
