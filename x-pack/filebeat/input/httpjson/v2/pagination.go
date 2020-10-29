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
