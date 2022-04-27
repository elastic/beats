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
	"net"
	"net/http"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	conf "github.com/elastic/elastic-agent-libs/config"
)

type provider struct {
	// Name contains a long name of provider and service metadata is fetched from.
	Name string

	// Local Set to true if local IP is accessed only
	Local bool

	// Create returns an actual metadataFetcher
	Create func(string, *conf.C) (metadataFetcher, error)
}

type metadataFetcher interface {
	fetchMetadata(context.Context, http.Client) result
}

// result is the result of a query for a specific hosting provider's metadata.
type result struct {
	provider string        // Hosting provider type.
	err      error         // Error that occurred while fetching (if any).
	metadata common.MapStr // A specific subset of the metadata received from the hosting provider.
}

var cloudMetaProviders = map[string]provider{
	"alibaba":       alibabaCloudMetadataFetcher,
	"ecs":           alibabaCloudMetadataFetcher,
	"azure":         azureVMMetadataFetcher,
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
	"huawei":        huaweiMetadataFetcher,
}

func selectProviders(configList providerList, providers map[string]provider) map[string]provider {
	return filterMetaProviders(providersFilter(configList, providers), providers)
}

func providersFilter(configList providerList, allProviders map[string]provider) func(string) bool {
	if len(configList) == 0 {
		return func(name string) bool {
			ff, ok := allProviders[name]
			return ok && ff.Local
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

func setupFetchers(providers map[string]provider, c *conf.C) ([]metadataFetcher, error) {
	mf := make([]metadataFetcher, 0, len(providers))
	visited := map[string]bool{}

	// Iterate over all providers and create an unique meta-data fetcher per provider type.
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
			return nil, errors.Wrapf(err, "failed to initialize the %v fetcher", name)
		}

		mf = append(mf, fetcher)
	}
	return mf, nil
}

// fetchMetadata attempts to fetch metadata in parallel from each of the
// hosting providers supported by this processor. It wait for the results to
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
			case results <- fetcher.fetchMetadata(ctx, client):
			}
		}()
	}

	for i := 0; i < len(p.initData.fetchers); i++ {
		select {
		case result := <-results:
			p.logger.Debugf("add_cloud_metadata: received disposition for %v after %v. %v",
				result.provider, time.Since(start), result)
			// Bail out on first success.
			if result.err == nil && result.metadata != nil {
				return &result
			}
		case <-ctx.Done():
			p.logger.Debugf("add_cloud_metadata: timed-out waiting for all responses")
			return nil
		}
	}

	return nil
}
