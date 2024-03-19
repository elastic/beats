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

package manifest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/magefile/mage/mg"

	"github.com/elastic/beats/v7/dev-tools/mage/artifacts"
)

func doWithRetries[T any](f func() (T, error)) (T, error) {
	var err error
	var resp T
	for _, backoff := range backoffSchedule {
		resp, err = f()
		if err == nil {
			return resp, nil
		}
		if mg.Verbose() {
			log.Printf("Request error: %+v\n", err)
			log.Printf("Retrying in %v\n", backoff)
		}
		time.Sleep(backoff)
	}

	// All retries failed
	return resp, err
}

func downloadFile(ctx context.Context, url string, filepath string) (path string, err error) {
	outFile, fileErr := os.Create(filepath)
	if fileErr != nil {
		return "", fmt.Errorf("failed to create destination file %w", fileErr)
	}
	defer func() {
		err = outFile.Close()
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("error creating request for %q: %w", url, err)
	}

	resp, reqErr := http.DefaultClient.Do(req)
	if reqErr != nil {
		return filepath, fmt.Errorf("failed to download manifest [%s]\n %w", url, err)
	}
	defer func() {
		err = resp.Body.Close()
	}()

	_, errCopy := io.Copy(outFile, resp.Body)
	if errCopy != nil {
		return "", fmt.Errorf("failed to decode manifest response [%s]\n %w", url, err)
	}
	if mg.Verbose() {
		log.Printf("<<<<<<<<< Downloaded: %s to %s", url, filepath)
	}

	return outFile.Name(), nil
}

func downloadManifestData(url string) (artifacts.Build, error) {
	var response artifacts.Build
	resp, err := http.Get(url) //nolint // we should have already verified that this is a proper valid url
	if err != nil {
		return response, fmt.Errorf("failed to download manifest [%s]\n %w", url, err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			panic(err)
		}
	}()

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return response, fmt.Errorf("failed to decode manifest response [%s]\n %w", url, err)
	}
	return response, nil
}
