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

	"github.com/elastic/beats/libbeat/common"
)

type Server struct {
	Httpd               Httpd               `json:"httpd"`
	HttpdRequestMethods HttpdRequestMethods `json:"httpd_request_methods"`
	HttpdStatusCodes    HttpdStatusCodes    `json:"httpd_status_codes"`
	Couchdb             Couchdb             `json:"couchdb"`
}

type Httpd struct {
	ViewReads                General `json:"view_reads"`
	BulkRequests             General `json:"bulk_requests"`
	ClientsRequestingChanges General `json:"clients_requesting_changes"`
	TemporaryViewReads       General `json:"temporary_view_reads"`
	Requests                 General `json:"requests"`
}

type HttpdRequestMethods struct {
	Copy   General `json:"COPY"`
	Head   General `json:"HEAD"`
	Post   General `json:"POST"`
	Delete General `json:"DELETE"`
	Get    General `json:"GET"`
	Put    General `json:"PUT"`
}

type HttpdStatusCodes struct {
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

type Couchdb struct {
	OpenOsFiles     General `json:"open_os_files"`
	OpenDatabases   General `json:"open_databases"`
	AuthCacheHits   General `json:"auth_cache_hits"`
	RequestTime     General `json:"request_time"`
	DatabaseReads   General `json:"database_reads"`
	DatabaseWrites  General `json:"database_writes"`
	AuthCacheMisses General `json:"auth_cache_misses"`
}

type General struct {
	Description string  `json:"description"`
	Current     float64 `json:"current"`
	Sum         float64 `json:"sum"`
	Mean        float64 `json:"mean"`
	Stddev      float64 `json:"stddev"`
	Min         float64 `json:"min"`
	Max         float64 `json:"max"`
}

func eventMapping(content []byte) common.MapStr {
	var data Server
	json.Unmarshal(content, &data)
	event := common.MapStr{
		"httpd": common.MapStr{
			"viewReads": common.MapStr{
				"description": data.Httpd.ViewReads.Description,
				"current":     data.Httpd.ViewReads.Current,
				"sum":         data.Httpd.ViewReads.Sum,
				"mean":        data.Httpd.ViewReads.Mean,
				"stddev":      data.Httpd.ViewReads.Stddev,
				"min":         data.Httpd.ViewReads.Min,
				"max":         data.Httpd.ViewReads.Max,
			},
			"bulk_requests": common.MapStr{
				"description": data.Httpd.BulkRequests.Description,
				"current":     data.Httpd.BulkRequests.Current,
				"sum":         data.Httpd.BulkRequests.Sum,
				"mean":        data.Httpd.BulkRequests.Mean,
				"stddev":      data.Httpd.BulkRequests.Stddev,
				"min":         data.Httpd.BulkRequests.Min,
				"max":         data.Httpd.BulkRequests.Max,
			},
			"clients_requesting_changes": common.MapStr{
				"description": data.Httpd.ClientsRequestingChanges.Description,
				"current":     data.Httpd.ClientsRequestingChanges.Current,
				"sum":         data.Httpd.ClientsRequestingChanges.Sum,
				"mean":        data.Httpd.ClientsRequestingChanges.Mean,
				"stddev":      data.Httpd.ClientsRequestingChanges.Stddev,
				"min":         data.Httpd.ClientsRequestingChanges.Min,
				"max":         data.Httpd.ClientsRequestingChanges.Max,
			},
			"temporary_view_reads": common.MapStr{
				"description": data.Httpd.TemporaryViewReads.Description,
				"current":     data.Httpd.TemporaryViewReads.Current,
				"sum":         data.Httpd.TemporaryViewReads.Sum,
				"mean":        data.Httpd.TemporaryViewReads.Mean,
				"stddev":      data.Httpd.TemporaryViewReads.Stddev,
				"min":         data.Httpd.TemporaryViewReads.Min,
				"max":         data.Httpd.TemporaryViewReads.Max,
			},
			"requests": common.MapStr{
				"description": data.Httpd.Requests.Description,
				"current":     data.Httpd.Requests.Current,
				"sum":         data.Httpd.Requests.Sum,
				"mean":        data.Httpd.Requests.Mean,
				"stddev":      data.Httpd.Requests.Stddev,
				"min":         data.Httpd.Requests.Min,
				"max":         data.Httpd.Requests.Max,
			},
		},
		"httpd_request_methods": common.MapStr{
			"COPY": common.MapStr{
				"description": data.HttpdRequestMethods.Copy.Description,
				"current":     data.HttpdRequestMethods.Copy.Current,
				"sum":         data.HttpdRequestMethods.Copy.Sum,
				"mean":        data.HttpdRequestMethods.Copy.Mean,
				"stddev":      data.HttpdRequestMethods.Copy.Stddev,
				"min":         data.HttpdRequestMethods.Copy.Min,
				"max":         data.HttpdRequestMethods.Copy.Max,
			},
			"HEAD": common.MapStr{
				"description": data.HttpdRequestMethods.Head.Description,
				"current":     data.HttpdRequestMethods.Head.Current,
				"sum":         data.HttpdRequestMethods.Head.Sum,
				"mean":        data.HttpdRequestMethods.Head.Mean,
				"stddev":      data.HttpdRequestMethods.Head.Stddev,
				"min":         data.HttpdRequestMethods.Head.Min,
				"max":         data.HttpdRequestMethods.Head.Max,
			},
			"POST": common.MapStr{
				"description": data.HttpdRequestMethods.Post.Description,
				"current":     data.HttpdRequestMethods.Post.Current,
				"sum":         data.HttpdRequestMethods.Post.Sum,
				"mean":        data.HttpdRequestMethods.Post.Mean,
				"stddev":      data.HttpdRequestMethods.Post.Stddev,
				"min":         data.HttpdRequestMethods.Post.Min,
				"max":         data.HttpdRequestMethods.Post.Max,
			},
			"DELETE": common.MapStr{
				"description": data.HttpdRequestMethods.Delete.Description,
				"current":     data.HttpdRequestMethods.Delete.Current,
				"sum":         data.HttpdRequestMethods.Delete.Sum,
				"mean":        data.HttpdRequestMethods.Delete.Mean,
				"stddev":      data.HttpdRequestMethods.Delete.Stddev,
				"min":         data.HttpdRequestMethods.Delete.Min,
				"max":         data.HttpdRequestMethods.Delete.Max,
			},
			"GET": common.MapStr{
				"description": data.HttpdRequestMethods.Get.Description,
				"current":     data.HttpdRequestMethods.Get.Current,
				"sum":         data.HttpdRequestMethods.Get.Sum,
				"mean":        data.HttpdRequestMethods.Get.Mean,
				"stddev":      data.HttpdRequestMethods.Get.Stddev,
				"min":         data.HttpdRequestMethods.Get.Min,
				"max":         data.HttpdRequestMethods.Get.Max,
			},
			"PUT": common.MapStr{
				"description": data.HttpdRequestMethods.Put.Description,
				"current":     data.HttpdRequestMethods.Put.Current,
				"sum":         data.HttpdRequestMethods.Put.Sum,
				"mean":        data.HttpdRequestMethods.Put.Mean,
				"stddev":      data.HttpdRequestMethods.Put.Stddev,
				"min":         data.HttpdRequestMethods.Put.Min,
				"max":         data.HttpdRequestMethods.Put.Max,
			},
		},
		"httpd_status_codes": common.MapStr{
			"200": common.MapStr{
				"description": data.HttpdStatusCodes.Num200.Description,
				"current":     data.HttpdStatusCodes.Num200.Current,
				"sum":         data.HttpdStatusCodes.Num200.Sum,
				"mean":        data.HttpdStatusCodes.Num200.Mean,
				"stddev":      data.HttpdStatusCodes.Num200.Stddev,
				"min":         data.HttpdStatusCodes.Num200.Min,
				"max":         data.HttpdStatusCodes.Num200.Max,
			},
			"201": common.MapStr{
				"description": data.HttpdStatusCodes.Num201.Description,
				"current":     data.HttpdStatusCodes.Num201.Current,
				"sum":         data.HttpdStatusCodes.Num201.Sum,
				"mean":        data.HttpdStatusCodes.Num201.Mean,
				"stddev":      data.HttpdStatusCodes.Num201.Stddev,
				"min":         data.HttpdStatusCodes.Num201.Min,
				"max":         data.HttpdStatusCodes.Num201.Max,
			},
			"202": common.MapStr{
				"description": data.HttpdStatusCodes.Num202.Description,
				"current":     data.HttpdStatusCodes.Num202.Current,
				"sum":         data.HttpdStatusCodes.Num202.Sum,
				"mean":        data.HttpdStatusCodes.Num202.Mean,
				"stddev":      data.HttpdStatusCodes.Num202.Stddev,
				"min":         data.HttpdStatusCodes.Num202.Min,
				"max":         data.HttpdStatusCodes.Num202.Max,
			},
			"301": common.MapStr{
				"description": data.HttpdStatusCodes.Num301.Description,
				"current":     data.HttpdStatusCodes.Num301.Current,
				"sum":         data.HttpdStatusCodes.Num301.Sum,
				"mean":        data.HttpdStatusCodes.Num301.Mean,
				"stddev":      data.HttpdStatusCodes.Num301.Stddev,
				"min":         data.HttpdStatusCodes.Num301.Min,
				"max":         data.HttpdStatusCodes.Num301.Max,
			},
			"304": common.MapStr{
				"description": data.HttpdStatusCodes.Num304.Description,
				"current":     data.HttpdStatusCodes.Num304.Current,
				"sum":         data.HttpdStatusCodes.Num304.Sum,
				"mean":        data.HttpdStatusCodes.Num304.Mean,
				"stddev":      data.HttpdStatusCodes.Num304.Stddev,
				"min":         data.HttpdStatusCodes.Num304.Min,
				"max":         data.HttpdStatusCodes.Num304.Max,
			},
			"400": common.MapStr{
				"description": data.HttpdStatusCodes.Num400.Description,
				"current":     data.HttpdStatusCodes.Num400.Current,
				"sum":         data.HttpdStatusCodes.Num400.Sum,
				"mean":        data.HttpdStatusCodes.Num400.Mean,
				"stddev":      data.HttpdStatusCodes.Num400.Stddev,
				"min":         data.HttpdStatusCodes.Num400.Min,
				"max":         data.HttpdStatusCodes.Num400.Max,
			},
			"401": common.MapStr{
				"description": data.HttpdStatusCodes.Num401.Description,
				"current":     data.HttpdStatusCodes.Num401.Current,
				"sum":         data.HttpdStatusCodes.Num401.Sum,
				"mean":        data.HttpdStatusCodes.Num401.Mean,
				"stddev":      data.HttpdStatusCodes.Num401.Stddev,
				"min":         data.HttpdStatusCodes.Num401.Min,
				"max":         data.HttpdStatusCodes.Num401.Max,
			},
			"403": common.MapStr{
				"description": data.HttpdStatusCodes.Num403.Description,
				"current":     data.HttpdStatusCodes.Num403.Current,
				"sum":         data.HttpdStatusCodes.Num403.Sum,
				"mean":        data.HttpdStatusCodes.Num403.Mean,
				"stddev":      data.HttpdStatusCodes.Num403.Stddev,
				"min":         data.HttpdStatusCodes.Num403.Min,
				"max":         data.HttpdStatusCodes.Num403.Max,
			},
			"404": common.MapStr{
				"description": data.HttpdStatusCodes.Num404.Description,
				"current":     data.HttpdStatusCodes.Num404.Current,
				"sum":         data.HttpdStatusCodes.Num404.Sum,
				"mean":        data.HttpdStatusCodes.Num404.Mean,
				"stddev":      data.HttpdStatusCodes.Num404.Stddev,
				"min":         data.HttpdStatusCodes.Num404.Min,
				"max":         data.HttpdStatusCodes.Num404.Max,
			},
			"405": common.MapStr{
				"description": data.HttpdStatusCodes.Num405.Description,
				"current":     data.HttpdStatusCodes.Num405.Current,
				"sum":         data.HttpdStatusCodes.Num405.Sum,
				"mean":        data.HttpdStatusCodes.Num405.Mean,
				"stddev":      data.HttpdStatusCodes.Num405.Stddev,
				"min":         data.HttpdStatusCodes.Num405.Min,
				"max":         data.HttpdStatusCodes.Num405.Max,
			},
			"409": common.MapStr{
				"description": data.HttpdStatusCodes.Num409.Description,
				"current":     data.HttpdStatusCodes.Num409.Current,
				"sum":         data.HttpdStatusCodes.Num409.Sum,
				"mean":        data.HttpdStatusCodes.Num409.Mean,
				"stddev":      data.HttpdStatusCodes.Num409.Stddev,
				"min":         data.HttpdStatusCodes.Num409.Min,
				"max":         data.HttpdStatusCodes.Num409.Max,
			},
			"412": common.MapStr{
				"description": data.HttpdStatusCodes.Num412.Description,
				"current":     data.HttpdStatusCodes.Num412.Current,
				"sum":         data.HttpdStatusCodes.Num412.Sum,
				"mean":        data.HttpdStatusCodes.Num412.Mean,
				"stddev":      data.HttpdStatusCodes.Num412.Stddev,
				"min":         data.HttpdStatusCodes.Num412.Min,
				"max":         data.HttpdStatusCodes.Num412.Max,
			},
			"500": common.MapStr{
				"description": data.HttpdStatusCodes.Num500.Description,
				"current":     data.HttpdStatusCodes.Num500.Current,
				"sum":         data.HttpdStatusCodes.Num500.Sum,
				"mean":        data.HttpdStatusCodes.Num500.Mean,
				"stddev":      data.HttpdStatusCodes.Num500.Stddev,
				"min":         data.HttpdStatusCodes.Num500.Min,
				"max":         data.HttpdStatusCodes.Num500.Max,
			},
		},
		"couchdb": common.MapStr{
			"database_writes": common.MapStr{
				"description": data.Couchdb.DatabaseWrites.Description,
				"current":     data.Couchdb.DatabaseWrites.Current,
				"sum":         data.Couchdb.DatabaseWrites.Sum,
				"mean":        data.Couchdb.DatabaseWrites.Mean,
				"stddev":      data.Couchdb.DatabaseWrites.Stddev,
				"min":         data.Couchdb.DatabaseWrites.Min,
				"max":         data.Couchdb.DatabaseWrites.Max,
			},
			"open_databases": common.MapStr{
				"description": data.Couchdb.OpenDatabases.Description,
				"current":     data.Couchdb.OpenDatabases.Current,
				"sum":         data.Couchdb.OpenDatabases.Sum,
				"mean":        data.Couchdb.OpenDatabases.Mean,
				"stddev":      data.Couchdb.OpenDatabases.Stddev,
				"min":         data.Couchdb.OpenDatabases.Min,
				"max":         data.Couchdb.OpenDatabases.Max,
			},
			"auth_cache_misses": common.MapStr{
				"description": data.Couchdb.AuthCacheMisses.Description,
				"current":     data.Couchdb.AuthCacheMisses.Current,
				"sum":         data.Couchdb.AuthCacheMisses.Sum,
				"mean":        data.Couchdb.AuthCacheMisses.Mean,
				"stddev":      data.Couchdb.AuthCacheMisses.Stddev,
				"min":         data.Couchdb.AuthCacheMisses.Min,
				"max":         data.Couchdb.AuthCacheMisses.Max,
			},
			"request_time": common.MapStr{
				"description": data.Couchdb.RequestTime.Description,
				"current":     data.Couchdb.RequestTime.Current,
				"sum":         data.Couchdb.RequestTime.Sum,
				"mean":        data.Couchdb.RequestTime.Mean,
				"stddev":      data.Couchdb.RequestTime.Stddev,
				"min":         data.Couchdb.RequestTime.Min,
				"max":         data.Couchdb.RequestTime.Max,
			},
			"database_reads": common.MapStr{
				"description": data.Couchdb.DatabaseReads.Description,
				"current":     data.Couchdb.DatabaseReads.Current,
				"sum":         data.Couchdb.DatabaseReads.Sum,
				"mean":        data.Couchdb.DatabaseReads.Mean,
				"stddev":      data.Couchdb.DatabaseReads.Stddev,
				"min":         data.Couchdb.DatabaseReads.Min,
				"max":         data.Couchdb.DatabaseReads.Max,
			},
			"auth_cache_hits": common.MapStr{
				"description": data.Couchdb.AuthCacheMisses.Description,
				"current":     data.Couchdb.AuthCacheMisses.Current,
				"sum":         data.Couchdb.AuthCacheMisses.Sum,
				"mean":        data.Couchdb.AuthCacheMisses.Mean,
				"stddev":      data.Couchdb.AuthCacheMisses.Stddev,
				"min":         data.Couchdb.AuthCacheMisses.Min,
				"max":         data.Couchdb.AuthCacheMisses.Max,
			},
			"open_os_files": common.MapStr{
				"description": data.Couchdb.OpenOsFiles.Description,
				"current":     data.Couchdb.OpenOsFiles.Current,
				"sum":         data.Couchdb.OpenOsFiles.Sum,
				"mean":        data.Couchdb.OpenOsFiles.Mean,
				"stddev":      data.Couchdb.OpenOsFiles.Stddev,
				"min":         data.Couchdb.OpenOsFiles.Min,
				"max":         data.Couchdb.OpenOsFiles.Max,
			},
		},
	}
	return event
}
