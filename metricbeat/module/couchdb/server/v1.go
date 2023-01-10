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

package server

import (
	"encoding/json"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/pkg/errors"
)

type V1 struct{}

func (v *V1) MapEvent(info *CommonInfo, in []byte) (mb.Event, error) {
	var data ServerV1
	err := json.Unmarshal(in, &data)
	if err != nil {
		return mb.Event{}, errors.Wrap(err, "error parsing v1 server JSON")
	}

	event := mapstr.M{
		"httpd": mapstr.M{
			"view_reads":                 data.Httpd.ViewReads.Current,
			"bulk_requests":              data.Httpd.BulkRequests.Current,
			"clients_requesting_changes": data.Httpd.ClientsRequestingChanges.Current,
			"temporary_view_reads":       data.Httpd.TemporaryViewReads.Current,
			"requests":                   data.Httpd.Requests.Current,
		},
		"httpd_request_methods": mapstr.M{
			"COPY":   data.HttpdRequestMethods.Copy.Current,
			"HEAD":   data.HttpdRequestMethods.Head.Current,
			"POST":   data.HttpdRequestMethods.Post.Current,
			"DELETE": data.HttpdRequestMethods.Delete.Current,
			"GET":    data.HttpdRequestMethods.Get.Current,
			"PUT":    data.HttpdRequestMethods.Put.Current,
		},
		"httpd_status_codes": mapstr.M{
			"200": data.HttpdStatusCodes.Num200.Current,
			"201": data.HttpdStatusCodes.Num201.Current,
			"202": data.HttpdStatusCodes.Num202.Current,
			"301": data.HttpdStatusCodes.Num301.Current,
			"304": data.HttpdStatusCodes.Num304.Current,
			"400": data.HttpdStatusCodes.Num400.Current,
			"401": data.HttpdStatusCodes.Num401.Current,
			"403": data.HttpdStatusCodes.Num403.Current,
			"404": data.HttpdStatusCodes.Num404.Current,
			"405": data.HttpdStatusCodes.Num405.Current,
			"409": data.HttpdStatusCodes.Num409.Current,
			"412": data.HttpdStatusCodes.Num412.Current,
			"500": data.HttpdStatusCodes.Num500.Current,
		},
		"couchdb": mapstr.M{
			"database_writes":   data.Couchdb.DatabaseWrites.Current,
			"open_databases":    data.Couchdb.OpenDatabases.Current,
			"auth_cache_misses": data.Couchdb.AuthCacheMisses.Current,
			"request_time":      data.Couchdb.RequestTime.Current,
			"database_reads":    data.Couchdb.DatabaseReads.Current,
			"auth_cache_hits":   data.Couchdb.AuthCacheMisses.Current,
			"open_os_files":     data.Couchdb.OpenOsFiles.Current,
		},
	}

	ecs := mapstr.M{}
	ecs.Put("service.id", info.UUID)
	ecs.Put("service.version", info.Version)

	return mb.Event{
		RootFields:      ecs,
		MetricSetFields: event,
	}, nil
}

// Server type defines all fields of the Server Metricset
type ServerV1 struct {
	Httpd               HttpdV1               `json:"httpd"`
	HttpdRequestMethods HttpdRequestMethodsV1 `json:"httpd_request_methods"`
	HttpdStatusCodes    HttpdStatusCodesV1    `json:"httpd_status_codes"`
	Couchdb             CouchdbV1             `json:"couchdb"`
}

// HttpdV1 type defines httpd fields of the Server Metricset
type HttpdV1 struct {
	ViewReads                General `json:"view_reads"`
	BulkRequests             General `json:"bulk_requests"`
	ClientsRequestingChanges General `json:"clients_requesting_changes"`
	TemporaryViewReads       General `json:"temporary_view_reads"`
	Requests                 General `json:"requests"`
}

// HttpdRequestMethodsV1 type defines httpd requests methods fields of the Server Metricset
type HttpdRequestMethodsV1 struct {
	Copy   General `json:"COPY"`
	Head   General `json:"HEAD"`
	Post   General `json:"POST"`
	Delete General `json:"DELETE"`
	Get    General `json:"GET"`
	Put    General `json:"PUT"`
}

// HttpdStatusCodesV1 type defines httpd status codes fields of the Server Metricset
type HttpdStatusCodesV1 struct {
	Num200 General `json:"200"`
	Num201 General `json:"201"`
	Num202 General `json:"202"`
	Num301 General `json:"301"`
	Num304 General `json:"304"`
	Num400 General `json:"400"`
	Num401 General `json:"401"`
	Num403 General `json:"403"`
	Num404 General `json:"404"`
	Num405 General `json:"405"`
	Num409 General `json:"409"`
	Num412 General `json:"412"`
	Num500 General `json:"500"`
}

// CouchdbV1 type defines couchdb fields of the Server Metricset
type CouchdbV1 struct {
	OpenOsFiles     General `json:"open_os_files"`
	OpenDatabases   General `json:"open_databases"`
	AuthCacheHits   General `json:"auth_cache_hits"`
	RequestTime     General `json:"request_time"`
	DatabaseReads   General `json:"database_reads"`
	DatabaseWrites  General `json:"database_writes"`
	AuthCacheMisses General `json:"auth_cache_misses"`
}

// General type defines common fields of the Server Metricset
type General struct {
	Current float64 `json:"current"`
}
