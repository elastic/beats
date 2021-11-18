package beater

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"
	"strings"

	"github.com/elastic/beats/v7/kubebeat/bundle"
	"github.com/open-policy-agent/opa/sdk"
	sdktest "github.com/open-policy-agent/opa/sdk/test"
)

type evaluator struct {
	bundleServer *sdktest.Server
	opa          *sdk.OPA
}

func NewEvaluator() (*evaluator, error) {
	policies := createCISPolicy(bundle.EmbeddedPolicy)
	// create a mock HTTP bundle bundleServer
	bundleServer, err := sdktest.NewServer(sdktest.MockBundle("/bundles/bundle.tar.gz", policies))
	if err != nil {
		return nil, fmt.Errorf("fail to init bundle server: %s", err.Error())
	}

	// provide the OPA configuration which specifies
	// fetching policy bundles from the mock bundleServer
	// and logging decisions locally to the console
	config := []byte(fmt.Sprintf(bundle.Config, bundleServer.URL()))

	// create an instance of the OPA object
	opa, err := sdk.New(context.Background(), sdk.Options{
		Config: bytes.NewReader(config),
	})
	if err != nil {
		return nil, fmt.Errorf("fail to init opa: %s", err.Error())
	}

	return &evaluator{
		opa:          opa,
		bundleServer: bundleServer,
	}, nil
}

func (e *evaluator) Decision(input interface{}) (interface{}, error) {
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

func (e *evaluator) Stop() {
	e.opa.Stop(context.Background())
	e.bundleServer.Stop()
}

func createCISPolicy(fileSystem embed.FS) map[string]string {
	policies := make(map[string]string)

	fs.WalkDir(fileSystem, ".", func(filepath string, info os.DirEntry, err error) error {
		if err != nil {
			log.Fatal(err)
		}
		if info.IsDir() == false && strings.HasSuffix(info.Name(), ".rego") && !strings.HasSuffix(info.Name(), "test.rego") {

			data, err := fs.ReadFile(fileSystem, filepath)
			if err == nil {
				policies[filepath] = string(data)
			}
		}
		return nil
	})

	return policies
}
