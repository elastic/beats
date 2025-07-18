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

package add_cloud_metadata

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type provider struct {
	// Name contains a long name of provider and service metadata is fetched from.
	Name string

	// DefaultEnabled allows to control whether metadata provider should be enabled by default
	// Set to true if metadata access is enabled by default for the provider
	DefaultEnabled bool

	// Create returns an actual metadataFetcher
	Create func(string, *conf.C) (metadataFetcher, error)
}

type metadataFetcher interface {
	fetchMetadata(context.Context, http.Client, *logp.Logger) result
}

// result is the result of a query for a specific hosting provider's metadata.
type result struct {
	provider string   // Hosting provider type.
	err      error    // Error that occurred while fetching (if any).
	metadata mapstr.M // A specific subset of the metadata received from the hosting provider.
}

var cloudMetaProviders = map[string]provider{
	"alibaba":       alibabaCloudMetadataFetcher,
	"ecs":           alibabaCloudMetadataFetcher,
	"digitalocean":  doMetadataFetcher,
	"aws":           ec2MetadataFetcher,
	"ec2":           ec2MetadataFetcher,
	"gcp":           gceMetadataFetcher,
	"openstack":     openstackNovaMetadataFetcher,
	"nova":          openstackNovaMetadataFetcher,
	"openstack-ssl": openstackNovaSSLMetadataFetcher,
	"nova-ssl":      openstackNovaSSLMetadataFetcher,
	"qcloud":        qcloudMetadataFetcher,
	"tencent":       qcloudMetadataFetcher,
	"huawei":        openstackNovaMetadataFetcher,
	"hetzner":       hetznerMetadataFetcher,
}

// priorityProviders contains providers which has priority over others.
// Metadata of these are derived using cloud provider SDKs, making them valid over metadata derived over well-known IP
// or other common endpoints. For example, Openstack supports EC2 compliant metadata endpoint. Thus adding possibility to
// conflict metadata between EC2/AWS and Openstack.
var priorityProviders = []string{
	"aws", "ec2",
}

func selectProviders(configList providerList, providers map[string]provider) map[string]provider {
	return filterMetaProviders(providersFilter(configList, providers), providers)
}

func providersFilter(configList providerList, allProviders map[string]provider) func(string) bool {
	if v, ok := os.LookupEnv("BEATS_ADD_CLOUD_METADATA_PROVIDERS"); ok {
		// We allow users to override the config and defaults with
		// this environment variable as a workaround in case the
		// configured/default providers misbehave.
		configList = nil
		for _, name := range strings.Split(v, ",") {
			configList = append(configList, strings.TrimSpace(name))
		}
		if len(configList) == 0 {
			// User explicitly disabled all providers.
			return func(string) bool {
				return false
			}
		}
	}
	if len(configList) == 0 {
		return func(name string) bool {
			ff, ok := allProviders[name]
			return ok && ff.DefaultEnabled
		}
	}
	return func(name string) (ok bool) {
		if ok = configList.Has(name); ok {
			_, ok = allProviders[name]
		}
		return ok
	}
}

func filterMetaProviders(filter func(string) bool, fetchers map[string]provider) map[string]provider {
	out := map[string]provider{}
	for name, ff := range fetchers {
		if filter(name) {
			out[name] = ff
		}
	}
	return out
}

func setupFetchers(providers map[string]provider, c *conf.C, logger *logp.Logger) ([]metadataFetcher, error) {
	mf := make([]metadataFetcher, 0, len(providers))
	visited := map[string]bool{}

	// Iterate over all providers and create a unique meta-data fetcher per provider type.
	// Some providers might appear twice in the set of providers to support aliases on provider names.
	// For example aws and ec2 both use the same provider.
	// The loop tracks already seen providers in the `visited` set, to ensure that we do not create
	// duplicate providers for aliases.
	for name, ff := range providers {
		if visited[ff.Name] {
			continue
		}
		visited[ff.Name] = true

		fetcher, err := ff.Create(name, c)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize the %v fetcher: %w", name, err)
		}

		mf = append(mf, fetcher)
	}
	return mf, nil
}

// fetchMetadata attempts to fetch metadata in parallel from each of the
// hosting providers supported by this processor. It will wait for the results to
// be returned or for a timeout to occur then returns the first result that
// completed in time.
func (p *addCloudMetadata) fetchMetadata() *result {
	p.logger.Debugf("add_cloud_metadata: starting to fetch metadata, timeout=%v", p.initData.timeout)
	start := time.Now()
	defer func() {
		p.logger.Debugf("add_cloud_metadata: fetchMetadata ran for %v", time.Since(start))
	}()

	// Create HTTP client with our timeouts and keep-alive disabled.
	client := http.Client{
		Timeout: p.initData.timeout,
		Transport: &http.Transport{
			DisableKeepAlives: true,
			DialContext: (&net.Dialer{
				Timeout:   p.initData.timeout,
				KeepAlive: 0,
			}).DialContext,
			TLSClientConfig: p.initData.tlsConfig.ToConfig(),
		},
	}

	// Create context to enable explicit cancellation of the http requests.
	ctx, cancel := context.WithTimeout(context.TODO(), p.initData.timeout)
	defer cancel()

	results := make(chan result)
	for _, fetcher := range p.initData.fetchers {
		fetcher := fetcher
		go func() {
			select {
			case <-ctx.Done():
			case results <- fetcher.fetchMetadata(ctx, client, p.logger):
			}
		}()
	}

	return acceptFirstPriorityResult(ctx, p.logger, start, results)
}

func acceptFirstPriorityResult(
	ctx context.Context,
	logger *logp.Logger,
	startTime time.Time,
	results chan result,
) *result {
	var response *result

	done := false
	for !done {
		select {
		case result := <-results:
			logger.Debugf("add_cloud_metadata: received disposition for %v after %v. %v",
				result.provider, time.Since(startTime), result)

			if result.err == nil && result.metadata != nil {
				if slices.Contains(priorityProviders, result.provider) {
					// We got a valid response from a priority provider, we don't need
					// to wait for the rest.
					response = &result
					done = true
				} else if response == nil {
					// For non-priority providers, only set the response if it's currently
					// empty.
					response = &result
				}
			}

			if result.err != nil {
				logger.Debugf("add_cloud_metadata: received error for provider %s: %v", result.provider, result.err)
			}
		case <-ctx.Done():
			done = true
		}
	}

	if response != nil {
		logger.Debugf("add_cloud_metadata: using provider %s metadata based on priority", response.provider)
	} else {
		logger.Debugf("add_cloud_metadata: timed-out waiting for responses")
	}
	return response
}
