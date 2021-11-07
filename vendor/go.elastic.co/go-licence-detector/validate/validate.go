// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package validate // import "go.elastic.co/go-licence-detector/validate"

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"runtime"
	"sync/atomic"
	"time"

	"go.elastic.co/go-licence-detector/dependency"
	"golang.org/x/sync/errgroup"
)

// maxErrors is the number of errors after which the validation loop will short-circuit.
// In practice, the number of errors encountered should be very low. If we are seeing a lot of errors, something is
// terribly broken (e.g. no network) and there's no point in continuing.
const maxErrors = 12

// Validate runs validation checks against the discovered dependencies.
func Validate(deps *dependency.List) error {
	return validateURLs(deps)
}

func validateURLs(deps *dependency.List) error {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	var errorCount uint32
	depsChan := make(chan dependency.Info, 64)
	numWorkers := runtime.NumCPU() + 2 // Rule of thumb for short-lived IO work. Feel free to tweak as necessary.
	client := newHTTPClient()
	group, ctx := errgroup.WithContext(ctx)

	// start workers
	for i := 0; i < numWorkers; i++ {
		group.Go(func() error {
			for dep := range depsChan {
				log.Printf("Checking %s", dep.Name)
				if err := checkDependencyURL(client, dep); err != nil {
					log.Printf("ERROR: Invalid URL [%s] for dependency [%s]: %v", dep.URL, dep.Name, err)
					if count := atomic.AddUint32(&errorCount, 1); count > maxErrors {
						return errors.New("maximum error count exceeded")
					}
				}
			}

			return nil
		})
	}

	// distribute work to workers
	group.Go(func() error {
		defer close(depsChan)
		for _, depList := range [][]dependency.Info{deps.Direct, deps.Indirect} {
			for _, dep := range depList {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case depsChan <- dep:
				}
			}
		}
		return nil
	})

	if err := group.Wait(); err != nil {
		return fmt.Errorf("validation encountered an error: %w", err)
	}

	if count := atomic.LoadUint32(&errorCount); count > 0 {
		return fmt.Errorf("encountered %d validation errors", count)
	}

	return nil
}

func newHTTPClient() *http.Client {
	defaultTransport := http.DefaultTransport.(*http.Transport)
	transport := *defaultTransport
	transport.MaxConnsPerHost = 5
	transport.IdleConnTimeout = 60 * time.Second
	transport.ResponseHeaderTimeout = 15 * time.Second

	return &http.Client{
		Transport: &transport,
		Timeout:   30 * time.Second,
	}
}

func checkDependencyURL(client *http.Client, dep dependency.Info) error {
	code, err := mkRequest(client, http.MethodHead, dep.URL)
	if err == nil && code == http.StatusMethodNotAllowed {
		code, err = mkRequest(client, http.MethodGet, dep.URL)
	}

	if err != nil {
		return err
	}

	// codes between 200 and 400 are OK.
	if code >= 200 && code < 400 {
		return nil
	}

	return fmt.Errorf("request returned status code %d", code)
}

func mkRequest(client *http.Client, method, url string) (int, error) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelFunc()

	req, err := http.NewRequestWithContext(ctx, method, url, http.NoBody)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("failed to create %s request to %s: %w", method, url, err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("%s request to %s returned an error: %w", method, url, err)
	}

	// cleanup the connection
	if resp.Body != nil {
		_, _ = io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}

	return resp.StatusCode, nil
}
