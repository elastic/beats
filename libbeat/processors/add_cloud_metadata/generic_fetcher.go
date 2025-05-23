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
	"net/http"

	cfg "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type genericFetcher struct {
	provider                 string
	schema                   schemaConv
	fetchRawProviderMetadata func(context.Context, http.Client, *result)
}

func newGenericMetadataFetcher(
	c *cfg.C,
	provider string,
	conv schemaConv,
	genericFetcherMeta func(context.Context, http.Client, *result),
) (*genericFetcher, error) {

	gFetcher := &genericFetcher{provider, conv, genericFetcherMeta}
	return gFetcher, nil
}

func (g *genericFetcher) fetchMetadata(ctx context.Context, client http.Client) result {
	res := result{provider: g.provider, metadata: mapstr.M{}}
	g.fetchRawProviderMetadata(ctx, client, &res)
	if res.err != nil {
		return res
	}
	res.metadata = g.schema(res.metadata)
	_, _ = res.metadata.Put("cloud.provider", g.provider)

	return res
}
