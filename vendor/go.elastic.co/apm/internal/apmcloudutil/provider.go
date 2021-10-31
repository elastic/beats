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

package apmcloudutil

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"go.elastic.co/apm/model"
)

// defaultClient is essentially the same as http.DefaultTransport, except
// that it has a short (100ms) dial timeout to avoid delaying startup.
var defaultClient = &http.Client{
	Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   100 * time.Millisecond,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns: 100,
	},
}

// Provider identifies the cloud provider.
type Provider string

const (
	// None is a pseudo cloud provider which disables fetching of
	// cloud metadata.
	None Provider = "none"

	// Auto is a pseudo cloud provider which uses trial-and-error to
	// fetch cloud metadata from all supported clouds.
	Auto Provider = "auto"

	// AWS represents the Amazon Web Services (EC2) cloud provider.
	AWS Provider = "aws"

	// Azure represents the Microsoft Azure cloud provider.
	Azure Provider = "azure"

	// GCP represents the Google Cloud Platform cloud provider.
	GCP Provider = "gcp"
)

// ParseProvider parses the provider name "s", returning the relevant Provider.
//
// If the provider name is unknown, None will be returned with an error.
func ParseProvider(s string) (Provider, error) {
	switch Provider(s) {
	case Auto, AWS, Azure, GCP, None:
		return Provider(s), nil
	}
	return None, fmt.Errorf("unknown cloud provider %q", s)
}

// GetCloudMetadata attempts to fetch cloud metadata for cloud provider p,
// storing it into out and returning a boolean indicating that the metadata
// was successfully retrieved.
//
// It is the caller's responsibility to set a reasonable timeout, to ensure
// requests do not block normal operation in non-cloud environments.
func (p Provider) GetCloudMetadata(ctx context.Context, logger Logger, out *model.Cloud) bool {
	return p.getCloudMetadata(ctx, defaultClient, logger, out)
}

func (p Provider) getCloudMetadata(ctx context.Context, client *http.Client, logger Logger, out *model.Cloud) bool {
	if p == None {
		return false
	}
	// Rather than switching on p, we loop through all providers
	// to support "auto". If and only if p == Auto, we'll loop back
	// around on errors.
	for _, provider := range []Provider{AWS, Azure, GCP} {
		if p != Auto && p != provider {
			continue
		}
		var err error
		switch provider {
		case AWS:
			err = getAWSCloudMetadata(ctx, client, out)
		case Azure:
			err = getAzureCloudMetadata(ctx, client, out)
		case GCP:
			err = getGCPCloudMetadata(ctx, client, out)
		}
		if err == nil {
			out.Provider = string(provider)
			return true
		} else if p != Auto {
			if logger != nil {
				logger.Warningf("cloud provider %q specified, but cloud metadata could not be retrieved: %s", p, err)
			}
			return false
		}
	}
	return false
}

// Logger defines the interface for logging while fetching cloud metadata.
type Logger interface {
	Warningf(format string, args ...interface{})
}
