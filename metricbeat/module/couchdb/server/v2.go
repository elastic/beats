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

	"github.com/pkg/errors"

	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/metricbeat/mb"
)

type V2 struct{}

func (v *V2) MapEvent(info *CommonInfo, in []byte) (mb.Event, error) {
	var data ServerV2
	err := json.Unmarshal(in, &data)
	if err != nil {
		return mb.Event{}, errors.Wrap(err, "error parsing v2 server JSON")
	}

	event := common.MapStr{
		"httpd": common.MapStr{
			"view_reads":                 data.Couchdb.Httpd.ViewReads.Value,
			"bulk_requests":              data.Couchdb.Httpd.BulkRequests.Value,
			"clients_requesting_changes": data.Couchdb.Httpd.ClientsRequestingChanges.Value,
			"temporary_view_reads":       data.Couchdb.Httpd.TemporaryViewReads.Value,
			"requests":                   data.Couchdb.Httpd.Requests.Value,
		},
		"httpd_request_methods": common.MapStr{
			"COPY":   data.Couchdb.HttpdRequestMethods.COPY.Value,
			"HEAD":   data.Couchdb.HttpdRequestMethods.HEAD.Value,
			"POST":   data.Couchdb.HttpdRequestMethods.POST.Value,
			"DELETE": data.Couchdb.HttpdRequestMethods.DELETE.Value,
			"GET":    data.Couchdb.HttpdRequestMethods.GET.Value,
			"PUT":    data.Couchdb.HttpdRequestMethods.PUT.Value,
		},
		"httpd_status_codes": common.MapStr{
			"200": data.Couchdb.HttpdStatusCodes.Num200.Value,
			"201": data.Couchdb.HttpdStatusCodes.Num201.Value,
			"202": data.Couchdb.HttpdStatusCodes.Num202.Value,
			"301": data.Couchdb.HttpdStatusCodes.Num301.Value,
			"304": data.Couchdb.HttpdStatusCodes.Num304.Value,
			"400": data.Couchdb.HttpdStatusCodes.Num400.Value,
			"401": data.Couchdb.HttpdStatusCodes.Num401.Value,
			"403": data.Couchdb.HttpdStatusCodes.Num403.Value,
			"404": data.Couchdb.HttpdStatusCodes.Num404.Value,
			"405": data.Couchdb.HttpdStatusCodes.Num405.Value,
			"409": data.Couchdb.HttpdStatusCodes.Num409.Value,
			"412": data.Couchdb.HttpdStatusCodes.Num412.Value,
			"500": data.Couchdb.HttpdStatusCodes.Num500.Value,
		},
		"couchdb": common.MapStr{
			"database_writes":   data.Couchdb.DatabaseWrites.Value,
			"open_databases":    data.Couchdb.OpenDatabases.Value,
			"auth_cache_misses": data.Couchdb.AuthCacheMisses.Value,
			"request_time":      data.Couchdb.RequestTime.Value.ArithmeticMean,
			"database_reads":    data.Couchdb.DatabaseReads.Value,
			"auth_cache_hits":   data.Couchdb.AuthCacheMisses.Value,
			"open_os_files":     data.Couchdb.OpenOsFiles.Value,
		},
	}

	ecs := common.MapStr{}
	ecs.Put("service.id", info.UUID)
	ecs.Put("service.version", info.Version)

	return mb.Event{
		RootFields:      ecs,
		MetricSetFields: event,
	}, nil
}

type ServerV2 struct {
	GlobalChanges struct {
		DbWrites               ValueTypeDesc `json:"db_writes"`
		EventDocConflict       ValueTypeDesc `json:"event_doc_conflict"`
		ListenerPendingUpdates ValueTypeDesc `json:"listener_pending_updates"`
		Rpcs                   ValueTypeDesc `json:"rpcs"`
		ServerPendingUpdates   ValueTypeDesc `json:"server_pending_updates"`
	} `json:"global_changes"`
	Mem3 struct {
		ShardCache struct {
			Eviction ValueTypeDesc `json:"eviction"`
			Hit      ValueTypeDesc `json:"hit"`
			Miss     ValueTypeDesc `json:"miss"`
		} `json:"shard_cache"`
	} `json:"mem3"`
	CouchLog struct {
		Level struct {
			Alert     ValueTypeDesc `json:"alert"`
			Critical  ValueTypeDesc `json:"critical"`
			Debug     ValueTypeDesc `json:"debug"`
			Emergency ValueTypeDesc `json:"emergency"`
			Error     ValueTypeDesc `json:"error"`
			Info      ValueTypeDesc `json:"info"`
			Notice    ValueTypeDesc `json:"notice"`
			Warning   ValueTypeDesc `json:"warning"`
		} `json:"level"`
	} `json:"couch_log"`
	DdocCache struct {
		Hit      ValueTypeDesc `json:"hit"`
		Miss     ValueTypeDesc `json:"miss"`
		Recovery ValueTypeDesc `json:"recovery"`
	} `json:"ddoc_cache"`
	Fabric struct {
		Worker struct {
			Timeouts ValueTypeDesc `json:"timeouts"`
		} `json:"worker"`
		OpenShard struct {
			Timeouts ValueTypeDesc `json:"timeouts"`
		} `json:"open_shard"`
		ReadRepairs struct {
			Success ValueTypeDesc `json:"success"`
			Failure ValueTypeDesc `json:"failure"`
		} `json:"read_repairs"`
		DocUpdate struct {
			Errors            ValueTypeDesc `json:"errors"`
			MismatchedErrors  ValueTypeDesc `json:"mismatched_errors"`
			WriteQuorumErrors ValueTypeDesc `json:"write_quorum_errors"`
		} `json:"doc_update"`
	} `json:"fabric"`
	Couchdb struct {
		Mrview struct {
			MapDoc ValueTypeDesc `json:"map_doc"`
			Emits  ValueTypeDesc `json:"emits"`
		} `json:"mrview"`
		AuthCacheHits      ValueTypeDesc `json:"auth_cache_hits"`
		AuthCacheMisses    ValueTypeDesc `json:"auth_cache_misses"`
		CollectResultsTime struct {
			Value AggValue `json:"value"`
			Type  string   `json:"type"`
			Desc  string   `json:"desc"`
		} `json:"collect_results_time"`
		DatabaseWrites ValueTypeDesc `json:"database_writes"`
		DatabaseReads  ValueTypeDesc `json:"database_reads"`
		DatabasePurges ValueTypeDesc `json:"database_purges"`
		DbOpenTime     struct {
			Value AggValue `json:"value"`
			Type  string   `json:"type"`
			Desc  string   `json:"desc"`
		} `json:"db_open_time"`
		DocumentInserts ValueTypeDesc `json:"document_inserts"`
		DocumentWrites  ValueTypeDesc `json:"document_writes"`
		DocumentPurges  struct {
			Total   ValueTypeDesc `json:"total"`
			Success ValueTypeDesc `json:"success"`
			Failure ValueTypeDesc `json:"failure"`
		} `json:"document_purges"`
		LocalDocumentWrites ValueTypeDesc `json:"local_document_writes"`
		Httpd               struct {
			BulkDocs struct {
				Value AggValue `json:"value"`
				Type  string   `json:"type"`
				Desc  string   `json:"desc"`
			} `json:"bulk_docs"`
			BulkRequests             ValueTypeDesc `json:"bulk_requests"`
			Requests                 ValueTypeDesc `json:"requests"`
			TemporaryViewReads       ValueTypeDesc `json:"temporary_view_reads"`
			ViewReads                ValueTypeDesc `json:"view_reads"`
			ClientsRequestingChanges ValueTypeDesc `json:"clients_requesting_changes"`
			PurgeRequests            ValueTypeDesc `json:"purge_requests"`
			AbortedRequests          ValueTypeDesc `json:"aborted_requests"`
		} `json:"httpd"`
		HttpdRequestMethods struct {
			COPY    ValueTypeDesc `json:"COPY"`
			DELETE  ValueTypeDesc `json:"DELETE"`
			GET     ValueTypeDesc `json:"GET"`
			HEAD    ValueTypeDesc `json:"HEAD"`
			OPTIONS ValueTypeDesc `json:"OPTIONS"`
			POST    ValueTypeDesc `json:"POST"`
			PUT     ValueTypeDesc `json:"PUT"`
		} `json:"httpd_request_methods"`
		HttpdStatusCodes struct {
			Num200 ValueTypeDesc `json:"200"`
			Num201 ValueTypeDesc `json:"201"`
			Num202 ValueTypeDesc `json:"202"`
			Num204 ValueTypeDesc `json:"204"`
			Num206 ValueTypeDesc `json:"206"`
			Num301 ValueTypeDesc `json:"301"`
			Num302 ValueTypeDesc `json:"302"`
			Num304 ValueTypeDesc `json:"304"`
			Num400 ValueTypeDesc `json:"400"`
			Num401 ValueTypeDesc `json:"401"`
			Num403 ValueTypeDesc `json:"403"`
			Num404 ValueTypeDesc `json:"404"`
			Num405 ValueTypeDesc `json:"405"`
			Num406 ValueTypeDesc `json:"406"`
			Num409 ValueTypeDesc `json:"409"`
			Num412 ValueTypeDesc `json:"412"`
			Num413 ValueTypeDesc `json:"413"`
			Num414 ValueTypeDesc `json:"414"`
			Num415 ValueTypeDesc `json:"415"`
			Num416 ValueTypeDesc `json:"416"`
			Num417 ValueTypeDesc `json:"417"`
			Num500 ValueTypeDesc `json:"500"`
			Num501 ValueTypeDesc `json:"501"`
			Num503 ValueTypeDesc `json:"503"`
		} `json:"httpd_status_codes"`
		OpenDatabases ValueTypeDesc `json:"open_databases"`
		OpenOsFiles   ValueTypeDesc `json:"open_os_files"`
		RequestTime   struct {
			Value AggValue `json:"value"`
			Type  string   `json:"type"`
			Desc  string   `json:"desc"`
		} `json:"request_time"`
		CouchServer struct {
			LruSkip ValueTypeDesc `json:"lru_skip"`
		} `json:"couch_server"`
		QueryServer struct {
			VduRejects     ValueTypeDesc `json:"vdu_rejects"`
			VduProcessTime struct {
				Value AggValue `json:"value"`
				Type  string   `json:"type"`
				Desc  string   `json:"desc"`
			} `json:"vdu_process_time"`
		} `json:"query_server"`
		Dbinfo struct {
			Value AggValue `json:"value"`
			Type  string   `json:"type"`
			Desc  string   `json:"desc"`
		} `json:"dbinfo"`
	} `json:"couchdb"`
	Rexi struct {
		Buffered ValueTypeDesc `json:"buffered"`
		Down     ValueTypeDesc `json:"down"`
		Dropped  ValueTypeDesc `json:"dropped"`
		Streams  struct {
			Timeout struct {
				InitStream ValueTypeDesc `json:"init_stream"`
				Stream     ValueTypeDesc `json:"stream"`
				WaitForAck ValueTypeDesc `json:"wait_for_ack"`
			} `json:"timeout"`
		} `json:"streams"`
	} `json:"rexi"`
	Pread struct {
		ExceedEOF   ValueTypeDesc `json:"exceed_eof"`
		ExceedLimit ValueTypeDesc `json:"exceed_limit"`
	} `json:"pread"`
	CouchReplicator struct {
		ChangesReadFailures  ValueTypeDesc `json:"changes_read_failures"`
		ChangesReaderDeaths  ValueTypeDesc `json:"changes_reader_deaths"`
		ChangesManagerDeaths ValueTypeDesc `json:"changes_manager_deaths"`
		ChangesQueueDeaths   ValueTypeDesc `json:"changes_queue_deaths"`
		Checkpoints          struct {
			Success ValueTypeDesc `json:"success"`
			Failure ValueTypeDesc `json:"failure"`
		} `json:"checkpoints"`
		FailedStarts ValueTypeDesc `json:"failed_starts"`
		Requests     ValueTypeDesc `json:"requests"`
		Responses    struct {
			Failure ValueTypeDesc `json:"failure"`
			Success ValueTypeDesc `json:"success"`
		} `json:"responses"`
		StreamResponses struct {
			Failure ValueTypeDesc `json:"failure"`
			Success ValueTypeDesc `json:"success"`
		} `json:"stream_responses"`
		WorkerDeaths    ValueTypeDesc `json:"worker_deaths"`
		WorkersStarted  ValueTypeDesc `json:"workers_started"`
		ClusterIsStable ValueTypeDesc `json:"cluster_is_stable"`
		DbScans         ValueTypeDesc `json:"db_scans"`
		Docs            struct {
			DbsCreated            ValueTypeDesc `json:"dbs_created"`
			DbsDeleted            ValueTypeDesc `json:"dbs_deleted"`
			DbsFound              ValueTypeDesc `json:"dbs_found"`
			DbChanges             ValueTypeDesc `json:"db_changes"`
			FailedStateUpdates    ValueTypeDesc `json:"failed_state_updates"`
			CompletedStateUpdates ValueTypeDesc `json:"completed_state_updates"`
		} `json:"docs"`
		Jobs struct {
			Adds          ValueTypeDesc `json:"adds"`
			DuplicateAdds ValueTypeDesc `json:"duplicate_adds"`
			Removes       ValueTypeDesc `json:"removes"`
			Starts        ValueTypeDesc `json:"starts"`
			Stops         ValueTypeDesc `json:"stops"`
			Crashes       ValueTypeDesc `json:"crashes"`
			Running       ValueTypeDesc `json:"running"`
			Pending       ValueTypeDesc `json:"pending"`
			Crashed       ValueTypeDesc `json:"crashed"`
			Total         ValueTypeDesc `json:"total"`
		} `json:"jobs"`
		Connection struct {
			Acquires      ValueTypeDesc `json:"acquires"`
			Creates       ValueTypeDesc `json:"creates"`
			Releases      ValueTypeDesc `json:"releases"`
			OwnerCrashes  ValueTypeDesc `json:"owner_crashes"`
			WorkerCrashes ValueTypeDesc `json:"worker_crashes"`
			Closes        ValueTypeDesc `json:"closes"`
		} `json:"connection"`
	} `json:"couch_replicator"`
}

type ValueTypeDesc struct {
	Value float64 `json:"value"`
	Type  string  `json:"type"`
	Desc  string  `json:"desc"`
}

type AggValue struct {
	Min               float64     `json:"min"`
	Max               float64     `json:"max"`
	ArithmeticMean    float64     `json:"arithmetic_mean"`
	GeometricMean     float64     `json:"geometric_mean"`
	HarmonicMean      float64     `json:"harmonic_mean"`
	Median            float64     `json:"median"`
	Variance          float64     `json:"variance"`
	StandardDeviation float64     `json:"standard_deviation"`
	Skewness          float64     `json:"skewness"`
	Kurtosis          float64     `json:"kurtosis"`
	Percentile        [][]float64 `json:"percentile"`
	Histogram         [][]float64 `json:"histogram"`
	N                 float64     `json:"n"`
}
