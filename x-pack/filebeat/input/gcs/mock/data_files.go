// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mock

const (
	beatsMultilineJSONBucket = "beatsmultilinejsonbucket"
	beatsJSONBucket          = "beatsjsonbucket"
	beatsNdJSONBucket        = "beatsndjsonbucket"
	beatsGzJSONBucket        = "beatsgzjsonbucket"
	beatsJSONWithArrayBucket = "beatsjsonwitharraybucket"
)

var fileBuckets = map[string]bool{
	beatsMultilineJSONBucket: true,
	beatsJSONBucket:          true,
	beatsNdJSONBucket:        true,
	beatsGzJSONBucket:        true,
	beatsJSONWithArrayBucket: true,
}

var availableFileObjects = map[string]map[string]bool{
	beatsMultilineJSONBucket: {
		"multiline.json": true,
	},
	beatsJSONBucket: {
		"log.json":           true,
		"events-array.json":  true,
		"array-at-root.json": true,
	},
	beatsJSONWithArrayBucket: {
		"array-at-root.json": true,
		"nested-arrays.json": true,
	},
	beatsNdJSONBucket: {
		"log.ndjson": true,
	},
	beatsGzJSONBucket: {
		"multiline.json.gz": true,
	},
}

var fetchFileBuckets = map[string]string{
	beatsMultilineJSONBucket: `{
		"kind": "storage#bucket",
		"selfLink": "https://www.googleapis.com/storage/v1/b/beatsmultilinejsonbucket",
		"id": "beatsmultilinejsonbucket",
		"name": "beatsmultilinejsonbucket",
		"projectNumber": "1059491012611",
		"metageneration": "1",
		"location": "ASIA-SOUTH1",
		"storageClass": "STANDARD",
		"etag": "CAE=",
		"timeCreated": "2022-08-24T12:20:04.723Z",
		"updated": "2022-08-24T12:20:04.723Z",
		"iamConfiguration": {
		  "bucketPolicyOnly": {
			"enabled": true,
			"lockedTime": "2022-11-22T12:20:04.723Z"
		  },
		  "uniformBucketLevelAccess": {
			"enabled": true,
			"lockedTime": "2022-11-22T12:20:04.723Z"
		  },
		  "publicAccessPrevention": "enforced"
		},
		"locationType": "region"
	  }`,
	beatsJSONBucket: `{
		"kind": "storage#bucket",
		"selfLink": "https://www.googleapis.com/storage/v1/b/beatsjsonbucket",
		"id": "beatsjsonbucket",
		"name": "beatsjsonbucket",
		"projectNumber": "1059491012611",
		"metageneration": "1",
		"location": "ASIA-SOUTH1",
		"storageClass": "STANDARD",
		"etag": "CAD=",
		"timeCreated": "2022-08-24T12:20:04.723Z",
		"updated": "2022-08-24T12:20:04.723Z",
		"iamConfiguration": {
		  "bucketPolicyOnly": {
			"enabled": true,
			"lockedTime": "2022-11-22T12:20:04.723Z"
		  },
		  "uniformBucketLevelAccess": {
			"enabled": true,
			"lockedTime": "2022-11-22T12:20:04.723Z"
		  },
		  "publicAccessPrevention": "enforced"
		},
		"locationType": "region"
	  }`,
	beatsJSONWithArrayBucket: `{
		"kind": "storage#bucket",
		"selfLink": "https://www.googleapis.com/storage/v1/b/beatsjsonwitharraybucket",
		"id": "beatsjsonwitharraybucket",
		"name": "beatsjsonwitharraybucket",
		"projectNumber": "1059491012611",
		"metageneration": "1",
		"location": "ASIA-SOUTH1",
		"storageClass": "STANDARD",
		"etag": "CAD=",
		"timeCreated": "2022-08-24T12:20:04.723Z",
		"updated": "2022-08-24T12:20:04.723Z",
		"iamConfiguration": {
		  "bucketPolicyOnly": {
			"enabled": true,
			"lockedTime": "2022-11-22T12:20:04.723Z"
		  },
		  "uniformBucketLevelAccess": {
			"enabled": true,
			"lockedTime": "2022-11-22T12:20:04.723Z"
		  },
		  "publicAccessPrevention": "enforced"
		},
		"locationType": "region"
	  }`,
	beatsNdJSONBucket: `{
		"kind": "storage#bucket",
		"selfLink": "https://www.googleapis.com/storage/v1/b/beatsndjsonbucket",
		"id": "beatsndjsonbucket",
		"name": "beatsndjsonbucket",
		"projectNumber": "1059491012611",
		"metageneration": "1",
		"location": "ASIA-SOUTH1",
		"storageClass": "STANDARD",
		"etag": "CAD=",
		"timeCreated": "2022-08-24T12:20:04.723Z",
		"updated": "2022-08-24T12:20:04.723Z",
		"iamConfiguration": {
		  "bucketPolicyOnly": {
			"enabled": true,
			"lockedTime": "2022-11-22T12:20:04.723Z"
		  },
		  "uniformBucketLevelAccess": {
			"enabled": true,
			"lockedTime": "2022-11-22T12:20:04.723Z"
		  },
		  "publicAccessPrevention": "enforced"
		},
		"locationType": "region"
	  }`,
	beatsGzJSONBucket: `{
		"kind": "storage#bucket",
		"selfLink": "https://www.googleapis.com/storage/v1/b/beatsgzjsonbucket",
		"id": "beatsgzjsonbucket",
		"name": "beatsgzjsonbucket",
		"projectNumber": "1059491012611",
		"metageneration": "1",
		"location": "ASIA-SOUTH1",
		"storageClass": "STANDARD",
		"etag": "CAD=",
		"timeCreated": "2022-08-24T12:20:04.723Z",
		"updated": "2022-08-24T12:20:04.723Z",
		"iamConfiguration": {
		  "bucketPolicyOnly": {
			"enabled": true,
			"lockedTime": "2022-11-22T12:20:04.723Z"
		  },
		  "uniformBucketLevelAccess": {
			"enabled": true,
			"lockedTime": "2022-11-22T12:20:04.723Z"
		  },
		  "publicAccessPrevention": "enforced"
		},
		"locationType": "region"
	  }`,
}

var objectFileList = map[string]string{
	beatsMultilineJSONBucket: `{
		"kind": "storage#objects",
		"items": [
		  {
			"kind": "storage#object",
			"id": "beatsmultilinejsonbucket/multiline.json/1661343619910503",
			"selfLink": "https://www.googleapis.com/storage/v1/b/beatsmultilinejsonbucket/o/multiline.json",
			"mediaLink": "https://content-storage.googleapis.com/download/storage/v1/b/beatsmultilinejsonbucket/o/multiline.json?generation=1661343619910503&alt=media",
			"name": "multiline.json",
			"bucket": "beatsmultilinejsonbucket",
			"generation": "1661343619910503",
			"metageneration": "1",
			"contentType": "application/octet-stream",
			"storageClass": "STANDARD",
			"size": "643",
			"md5Hash": "UjQX73kQRTHx+UyXZDvVkg==",
			"crc32c": "ZI5qFw==",
			"etag": "COeWwqK73/kCEAE=",
			"timeCreated": "2022-08-24T12:20:19.911Z",
			"updated": "2022-08-24T12:20:19.911Z",
			"timeStorageClassUpdated": "2022-08-24T12:20:19.911Z"
		  }
		]
	  }`,
	beatsJSONBucket: `{
		"kind": "storage#objects",
		"items": [
		  {
			"kind": "storage#object",
			"id": "beatsjsonbucket/log.json/1661343619910503",
			"selfLink": "https://www.googleapis.com/storage/v1/b/beatsjsonbucket/o/log.json",
			"mediaLink": "https://content-storage.googleapis.com/download/storage/v1/b/beatsjsonbucket/o/log.json?generation=1661343619910503&alt=media",
			"name": "log.json",
			"bucket": "beatsjsonbucket",
			"generation": "1661343619910503",
			"metageneration": "1",
			"contentType": "application/json",
			"storageClass": "STANDARD",
			"size": "643",
			"md5Hash": "UjQX73kQRTHx+UyXZDvVkg==",
			"crc32c": "ZI5qFw==",
			"etag": "COeWwqK73/kCEAE=",
			"timeCreated": "2022-08-24T12:20:19.911Z",
			"updated": "2022-08-24T12:20:19.911Z",
			"timeStorageClassUpdated": "2022-08-24T12:20:19.911Z"
		  },
		  {
			"kind": "storage#object",
			"id": "beatsjsonbucket/events-array.json/1661343636712270",
			"selfLink": "https://www.googleapis.com/storage/v1/b/beatsjsonbucket/o/events-array.json",
			"mediaLink": "https://content-storage.googleapis.com/download/storage/v1/b/beatsjsonbucket/o/events-array.json?generation=1661343636712270&alt=media",
			"name": "events-array.json",
			"bucket": "beatsjsonbucket",
			"generation": "1661343636712270",
			"metageneration": "1",
			"contentType": "application/json",
			"storageClass": "STANDARD",
			"size": "434",
			"md5Hash": "eOXjYygu6k6687Uf3vPtKQ==",
			"crc32c": "hHW/Qw==",
			"etag": "CM7Ww6q73/kCEAE=",
			"timeCreated": "2022-08-24T12:20:36.713Z",
			"updated": "2022-08-24T12:20:36.713Z",
			"timeStorageClassUpdated": "2022-08-24T12:20:36.713Z"
		  }
		]
	  }`,
	beatsJSONWithArrayBucket: `{
		"kind": "storage#objects",
		"items": [
		  {
			"kind": "storage#object",
			"id": "beatsjsonwitharraybucket/array-at-root.json/1661343636712270",
			"selfLink": "https://www.googleapis.com/storage/v1/b/beatsjsonwitharraybucket/o/array-at-root.json",
			"mediaLink": "https://content-storage.googleapis.com/download/storage/v1/b/beatsjsonwitharraybucket/o/array-at-root.json?generation=1661343636712270&alt=media",
			"name": "array-at-root.json",
			"bucket": "beatsjsonwitharraybucket",
			"generation": "1661343636712270",
			"metageneration": "1",
			"contentType": "application/json",
			"storageClass": "STANDARD",
			"size": "434",
			"md5Hash": "eOXjYygu6k6687Uf3vPtKQ==",
			"crc32c": "hHW/Qw==",
			"etag": "CM7Ww6q73/kCEAE=",
			"timeCreated": "2022-08-24T12:20:36.713Z",
			"updated": "2022-08-24T12:20:36.713Z",
			"timeStorageClassUpdated": "2022-08-24T12:20:36.713Z"
		  },
		  {
			"kind": "storage#object",
			"id": "beatsjsonwitharraybucket/nested-arrays.json/1661343636712270",
			"selfLink": "https://www.googleapis.com/storage/v1/b/beatsjsonwitharraybucket/o/nested-arrays.json",
			"mediaLink": "https://content-storage.googleapis.com/download/storage/v1/b/beatsjsonwitharraybucket/o/nested-arrays.json?generation=1661343636712270&alt=media",
			"name": "nested-arrays.json",
			"bucket": "beatsjsonwitharraybucket",
			"generation": "1661343636712270",
			"metageneration": "1",
			"contentType": "application/json",
			"storageClass": "STANDARD",
			"size": "434",
			"md5Hash": "eOXjYygu6k6687Uf3vPtKQ==",
			"crc32c": "hHW/Qw==",
			"etag": "CM7Ww6q73/kCEAE=",
			"timeCreated": "2022-08-24T12:20:36.713Z",
			"updated": "2022-08-24T12:20:36.713Z",
			"timeStorageClassUpdated": "2022-08-24T12:20:36.713Z"
		  }
		]
	  }`,
	beatsNdJSONBucket: `{
		"kind": "storage#objects",
		"items": [
			{
				"kind": "storage#object",
				"id": "beatsndjsonbucket/log.ndjson/1672652082275368",
				"selfLink": "https://www.googleapis.com/storage/v1/b/beatsndjsonbucket/o/log.ndjson",
				"mediaLink": "https://content-storage.googleapis.com/download/storage/v1/b/beatsndjsonbucket/o/log.ndjson?generation=1672652082275368&alt=media",
				"name": "log.ndjson",
				"bucket": "beatsndjsonbucket",
				"generation": "1672652082275368",
				"metageneration": "2",
				"contentType": "application/x-ndjson",
				"storageClass": "STANDARD",
				"size": "195",
				"md5Hash": "dvu5gUK256Qw4xwVXX9esw==",
				"crc32c": "hdDB+A==",
				"etag": "CKjIycnKqPwCEAI=",
				"timeCreated": "2023-01-02T09:34:42.276Z",
				"updated": "2023-01-02T09:35:05.800Z",
				"timeStorageClassUpdated": "2023-01-02T09:34:42.276Z"
			}
		]
	  }`,
	beatsGzJSONBucket: `{
		"kind": "storage#objects",
		"items": [
			{
				"kind": "storage#object",
				"id": "beatsgzjsonbucket/multiline.json.gz/1661343636712270",
				"selfLink": "https://www.googleapis.com/storage/v1/b/beatsgzjsonbucket/o/multiline.json.gz",
				"mediaLink": "https://content-storage.googleapis.com/download/storage/v1/b/beatsgzjsonbucket/o/multiline.json.gz?generation=1661343636712270&alt=media",
				"name": "multiline.json.gz",
				"bucket": "beatsgzjsonbucket",
				"generation": "1661343636712270",
				"metageneration": "1",
				"contentType": "application/json",
				"storageClass": "STANDARD",
				"size": "434",
				"md5Hash": "eOXjYygu6k6687Uf3vPtKQ==",
				"crc32c": "hHW/Qw==",
				"etag": "CM7Ww6q73/kCEAE=",
				"timeCreated": "2022-08-24T12:20:36.713Z",
				"updated": "2022-08-24T12:20:36.713Z",
				"timeStorageClassUpdated": "2022-08-24T12:20:36.713Z"
			}
		]
	  }`,
}

// These variables are intentionally indented like this to match the output of certain tests
//
//nolint:stylecheck // required for edge case test scenario
var BeatsFilesBucket_multiline_json = []string{
	"{\n    \"@timestamp\": \"2021-05-25T17:25:42.806Z\",\n    \"log.level\": \"error\",\n    \"message\": \"error making request\"\n}",
	"{\n    \"@timestamp\": \"2021-05-25T17:25:51.391Z\",\n    \"log.level\": \"info\",\n    \"message\": \"available space 44.3gb\"\n}",
}

var BeatsFilesBucket_log_json = []string{
	`{"@timestamp":"2021-05-25T17:25:42.806Z","log.level":"error","message":"error making http request"}`,
	`{"@timestamp":"2021-05-25T17:25:51.391Z","log.level":"info","message":"available disk space 44.3gb"}`,
	"{\n    \"Events\": [\n        {\n            \"time\": \"2021-05-25 18:20:58 UTC\",\n            \"msg\": \"hello\"\n        },\n        {\n            \"time\": \"2021-05-26 22:21:40 UTC\",\n            \"msg\": \"world\"\n        }\n    ]\n}",
}
var BeatsFilesBucket_json_array = []string{
	"{\n        \"time\": \"2021-05-25 18:20:58 UTC\",\n        \"msg\": \"hello\"\n    }",
	"{\n        \"time\": \"2021-05-26 22:21:40 UTC\",\n        \"msg\": \"world\"\n    }",
	"[\n        {\n            \"time\": \"2021-05-25 18:20:58 UTC\",\n            \"msg\": \"hello\"\n        },\n        {\n            \"time\": \"2021-05-26 22:21:40 UTC\",\n            \"msg\": \"world\"\n        }\n    ]",
	"[\n        {\n            \"time\": \"2021-05-25 18:20:58 UTC\",\n            \"msg\": \"hi\"\n        },\n        {\n            \"time\": \"2021-05-26 22:21:40 UTC\",\n            \"msg\": \"seoul\"\n        }\n    ]",
}
var BeatsFilesBucket_log_ndjson = []string{
	`{"@timestamp":"2021-05-25T17:25:42.806Z","log.level":"error","message":"error in http request"}`,
	`{"@timestamp":"2021-05-25T17:25:51.391Z","log.level":"info","message":"available space is 44.3gb"}`,
}

var BeatsFilesBucket_multiline_json_gz = []string{
	"{\n    \"@timestamp\": \"2021-05-25T17:25:42.806Z\",\n    \"log.level\": \"error\",\n    \"message\": \"error making http request\"\n}",
	"{\n    \"@timestamp\": \"2021-05-25T17:25:51.391Z\",\n    \"log.level\": \"info\",\n    \"message\": \"available disk space 44.3gb\"\n}",
}
