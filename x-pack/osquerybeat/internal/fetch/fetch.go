// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fetch

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/hash"
)

func Download(url, fp string) (hashout string, err error) {
	log.Printf("Download %s to %s", url, fp)

	cli := http.Client{}

	res, err := cli.Get(url)
	if err != nil {
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		// Read body for extended error message
		b, err := ioutil.ReadAll(res.Body)
		var s string
		if err != nil {
			log.Printf("Failed to read the error response body: %v", err)
		} else {
			s = string(b)
		}
		return hashout, fmt.Errorf("failed fetch %s, status: %d, message: %s", url, res.StatusCode, s)
	}

	out, err := os.OpenFile(fp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return
	}
	defer out.Close()

	// Calculate hash and write file
	return hash.Calculate(res.Body, out)
}
