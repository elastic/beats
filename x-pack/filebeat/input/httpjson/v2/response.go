package v2

import (
	"net/http"
	"net/url"

	"github.com/elastic/beats/v7/libbeat/common"
)

const responseNamespace = "response"

func registerResponseTransforms() {
	registerTransform(responseNamespace, appendName, newAppendResponse)
	registerTransform(responseNamespace, deleteName, newDeleteResponse)
	registerTransform(responseNamespace, setName, newSetResponse)
}

type response struct {
	body   common.MapStr
	header http.Header
	url    *url.URL
}
