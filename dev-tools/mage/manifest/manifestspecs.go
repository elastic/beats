// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

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

func downloadFile(ctx context.Context, url string, filepath string) (string, error) {
	outFile, fileErr := os.Create(filepath)
	if fileErr != nil {
		return "", fmt.Errorf("failed to create destination file %w", fileErr)
	}
	defer func() {
		if err := outFile.Close(); err != nil {
			panic(err)
		}
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
		if err := resp.Body.Close(); err != nil {
			panic(err)
		}
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
