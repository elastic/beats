// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"

	"github.com/elastic/beats/v7/libbeat/common"
)

type pagination struct {
	extraBodyContent common.MapStr
	header           *Header
	idField          string
	requestField     string
	urlField         string
	url              string
}

func newPaginationFromConfig(config config) *pagination {
	if !config.Pagination.IsEnabled() {
		return nil
	}
	return &pagination{
		extraBodyContent: config.Pagination.ExtraBodyContent.Clone(),
		header:           config.Pagination.Header,
		idField:          config.Pagination.IDField,
		requestField:     config.Pagination.RequestField,
		urlField:         config.Pagination.URLField,
		url:              config.Pagination.URL,
	}
}

func (p *pagination) nextRequestInfo(ri *requestInfo, response response, lastObj common.MapStr) (*requestInfo, bool, error) {
	if p == nil {
		return ri, false, nil
	}

	if p.header == nil {
		var err error
		// Pagination control using HTTP Body fields
		if err = p.setRequestInfoFromBody(response.body, lastObj, ri); err != nil {
			// if the field is not found, there is no next page
			if errors.Is(err, common.ErrKeyNotFound) {
				return ri, false, nil
			}
			return ri, false, err
		}

		return ri, true, nil
	}

	// Pagination control using HTTP Header
	url, err := getNextLinkFromHeader(response.header, p.header.FieldName, p.header.RegexPattern)
	if err != nil {
		return ri, false, fmt.Errorf("failed to retrieve the next URL for pagination: %w", err)
	}
	if ri.url == url || url == "" {
		return ri, false, nil
	}

	ri.url = url

	return ri, true, nil
}

// getNextLinkFromHeader retrieves the next URL for pagination from the HTTP Header of the response
func getNextLinkFromHeader(header http.Header, fieldName string, re *regexp.Regexp) (string, error) {
	links, ok := header[fieldName]
	if !ok {
		return "", fmt.Errorf("field %s does not exist in the HTTP Header", fieldName)
	}
	for _, link := range links {
		matchArray := re.FindAllStringSubmatch(link, -1)
		if len(matchArray) == 1 {
			return matchArray[0][1], nil
		}
	}
	return "", nil
}

// createRequestInfoFromBody creates a new RequestInfo for a new HTTP request in pagination based on HTTP response body
func (p *pagination) setRequestInfoFromBody(response, last common.MapStr, ri *requestInfo) error {
	// we try to get it from last element, if not found, from the original response
	v, err := last.GetValue(p.idField)
	if err == common.ErrKeyNotFound {
		v, err = response.GetValue(p.idField)
	}

	if err != nil {
		return fmt.Errorf("failed to retrieve id_field for pagination: %w", err)
	}

	if p.requestField != "" {
		_, _ = ri.contentMap.Put(p.requestField, v)
		if p.url != "" {
			ri.url = p.url
		}
	} else if p.urlField != "" {
		url, err := url.Parse(ri.url)
		if err == nil {
			q := url.Query()
			q.Set(p.urlField, fmt.Sprint(v))
			url.RawQuery = q.Encode()
			ri.url = url.String()
		}
	} else {
		switch vt := v.(type) {
		case string:
			ri.url = vt
		default:
			return errors.New("pagination ID is not of string type")
		}
	}
	if len(p.extraBodyContent) > 0 {
		ri.contentMap.Update(common.MapStr(p.extraBodyContent))
	}
	return nil
}
