// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v2

import (
	"net/http"
	"net/url"

	"github.com/elastic/beats/v7/libbeat/common"
)

const paginationNamespace = "pagination"

func registerPaginationTransforms() {
	registerTransform(paginationNamespace, appendName, newAppendPagination)
	registerTransform(paginationNamespace, deleteName, newDeletePagination)
	registerTransform(paginationNamespace, setName, newSetPagination)
}

type pagination struct {
	body   common.MapStr
	header http.Header
	url    *url.URL
}

func (p *pagination) nextPageRequest() (*http.Request, error) {
	return nil, nil
}
