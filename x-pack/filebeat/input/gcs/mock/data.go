// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mock

const (
	bucketGcsTestNew    = "gcs-test-new"
	bucketGcsTestLatest = "gcs-test-latest"
)

var buckets = map[string]bool{
	bucketGcsTestNew:    true,
	bucketGcsTestLatest: true,
}

var availableObjects = map[string]map[string]bool{
	bucketGcsTestNew: {
		"ata.json":      true,
		"data_3.json":   true,
		"docs/ata.json": true,
	},
	bucketGcsTestLatest: {
		"ata.json":    true,
		"data_3.json": true,
	},
}

var objects = map[string]map[string]string{
	bucketGcsTestNew: {
		"ata.json":      Gcs_test_new_object_ata_json,
		"data_3.json":   Gcs_test_new_object_data3_json,
		"docs/ata.json": Gcs_test_new_object_docs_ata_json,
	},
	bucketGcsTestLatest: {
		"ata.json":    Gcs_test_latest_object_ata_json,
		"data_3.json": Gcs_test_latest_object_data3_json,
	},
}

var fetchBucket = map[string]string{
	bucketGcsTestNew: `{
		"kind": "storage#bucket",
		"selfLink": "https://www.googleapis.com/storage/v1/b/gcs-test-new",
		"id": "gcs-test-new",
		"name": "gcs-test-new",
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
	bucketGcsTestLatest: `{
		"kind": "storage#bucket",
		"selfLink": "https://www.googleapis.com/storage/v1/b/gcs-test-latest",
		"id": "gcs-test-latest",
		"name": "gcs-test-latest",
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

var objectList = map[string]string{
	bucketGcsTestNew: `{
		"kind": "storage#objects",
		"items": [
		  {
			"kind": "storage#object",
			"id": "gcs-test-new/ata.json/1661343619910503",
			"selfLink": "https://www.googleapis.com/storage/v1/b/gcs-test-new/o/ata.json",
			"mediaLink": "https://content-storage.googleapis.com/download/storage/v1/b/gcs-test-new/o/ata.json?generation=1661343619910503&alt=media",
			"name": "ata.json",
			"bucket": "gcs-test-new",
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
			"id": "gcs-test-new/data_3.json/1661343636712270",
			"selfLink": "https://www.googleapis.com/storage/v1/b/gcs-test-new/o/data_3.json",
			"mediaLink": "https://content-storage.googleapis.com/download/storage/v1/b/gcs-test-new/o/data_3.json?generation=1661343636712270&alt=media",
			"name": "data_3.json",
			"bucket": "gcs-test-new",
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
			"id": "gcs-test-new/docs/ata.json/1661424694341949",
			"selfLink": "https://www.googleapis.com/storage/v1/b/gcs-test-new/o/docs%2Fata.json",
			"mediaLink": "https://content-storage.googleapis.com/download/storage/v1/b/gcs-test-new/o/docs%2Fata.json?generation=1661424694341949&alt=media",
			"name": "docs/ata.json",
			"bucket": "gcs-test-new",
			"generation": "1661424694341949",
			"metageneration": "1",
			"contentType": "application/json",
			"storageClass": "STANDARD",
			"size": "643",
			"md5Hash": "UjQX73kQRTHx+UyXZDvVkg==",
			"crc32c": "ZI5qFw==",
			"etag": "CL3i6KXp4fkCEAE=",
			"timeCreated": "2022-08-25T10:51:34.343Z",
			"updated": "2022-08-25T10:51:34.343Z",
			"timeStorageClassUpdated": "2022-08-25T10:51:34.343Z"
		  }
		]
	  }`,
	bucketGcsTestLatest: `{
		"kind": "storage#objects",
		"items": [
		  {
			"kind": "storage#object",
			"id": "gcs-test-latest/ata.json/1661343619910503",
			"selfLink": "https://www.googleapis.com/storage/v1/b/gcs-test-latest/o/ata.json",
			"mediaLink": "https://content-storage.googleapis.com/download/storage/v1/b/gcs-test-latest/o/ata.json?generation=1661343619910503&alt=media",
			"name": "ata.json",
			"bucket": "gcs-test-latest",
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
			"id": "gcs-test-latest/data_3.json/1661343636712270",
			"selfLink": "https://www.googleapis.com/storage/v1/b/gcs-test-latest/o/data_3.json",
			"mediaLink": "https://content-storage.googleapis.com/download/storage/v1/b/gcs-test-latest/o/data_3.json?generation=1661343636712270&alt=media",
			"name": "data_3.json",
			"bucket": "gcs-test-latest",
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

var Gcs_test_new_object_ata_json = `{
    "id": 1,
    "title": "iPhone 9",
    "description": "An apple mobile which is nothing like apple",
    "price": 549,
    "discountPercentage": 12.96,
    "rating": 4.69,
    "stock": 94,
    "brand": "Apple",
    "category": "smartphones",
    "thumbnail": "https://dummyjson.com/image/i/products/1/thumbnail.jpg",
    "images": [
        "https://dummyjson.com/image/i/products/1/1.jpg",
        "https://dummyjson.com/image/i/products/1/2.jpg",
        "https://dummyjson.com/image/i/products/1/3.jpg",
        "https://dummyjson.com/image/i/products/1/4.jpg",
        "https://dummyjson.com/image/i/products/1/thumbnail.jpg"
    ]
}`

var Gcs_test_new_object_data3_json = `{
    "id": 3,
    "title": "Samsung Universe 9",
    "description": "Samsung's new variant which goes beyond Galaxy to the Universe",
    "price": 1249,
    "discountPercentage": 15.46,
    "rating": 4.09,
    "stock": 36,
    "brand": "Samsung",
    "category": "smartphones",
    "thumbnail": "https://dummyjson.com/image/i/products/3/thumbnail.jpg",
    "images": [
        "https://dummyjson.com/image/i/products/3/1.jpg"
    ]
}`

var Gcs_test_new_object_docs_ata_json = `{
    "id": 1,
    "title": "iPhone 9",
    "description": "An apple mobile which is nothing like apple",
    "price": 549,
    "discountPercentage": 15.46,
    "rating": 4.09,
    "stock": 36,
    "brand": "Samsung",
    "category": "smartphones",
    "thumbnail": "https://dummyjson.com/image/i/products/3/thumbnail.jpg",
    "images": [
        "https://dummyjson.com/image/i/products/3/1.jpg"
    ]
}`

var Gcs_test_latest_object_ata_json = `{
    "id": 1,
    "title": "iPhone 9",
    "description": "An apple mobile which is nothing like apple",
    "price": 549,
    "discountPercentage": 12.96,
    "rating": 4.69,
    "stock": 94,
    "brand": "Apple",
    "category": "smartphones",
    "thumbnail": "https://dummyjson.com/image/i/products/1/thumbnail.jpg",
    "images": [
        "https://dummyjson.com/image/i/products/1/1.jpg",
        "https://dummyjson.com/image/i/products/1/2.jpg",
        "https://dummyjson.com/image/i/products/1/3.jpg",
        "https://dummyjson.com/image/i/products/1/4.jpg",
        "https://dummyjson.com/image/i/products/1/thumbnail.jpg"
    ]
}`

var Gcs_test_latest_object_data3_json = `{
    "id": 3,
    "title": "Samsung Universe 9",
    "description": "Samsung's new variant which goes beyond Galaxy to the Universe",
    "price": 1249,
    "discountPercentage": 15.46,
    "rating": 4.09,
    "stock": 36,
    "brand": "Samsung",
    "category": "smartphones",
    "thumbnail": "https://dummyjson.com/image/i/products/3/thumbnail.jpg",
    "images": [
        "https://dummyjson.com/image/i/products/3/1.jpg"
    ]
}`

// These 2 variables are intentionally indented like this to match the output of certain tests
//
//nolint:stylecheck // required for edge case test scenario
var Gcs_test_latest_object_ata_json_parsed = `[{"brand":"Apple","category":"smartphones","description":"An apple mobile which is nothing like apple","discountPercentage":12.96,"id":1,"images":["https://dummyjson.com/image/i/products/1/1.jpg","https://dummyjson.com/image/i/products/1/2.jpg","https://dummyjson.com/image/i/products/1/3.jpg","https://dummyjson.com/image/i/products/1/4.jpg","https://dummyjson.com/image/i/products/1/thumbnail.jpg"],"price":549,"rating":4.69,"stock":94,"thumbnail":"https://dummyjson.com/image/i/products/1/thumbnail.jpg","title":"iPhone 9"}]`
var Gcs_test_latest_object_data3_json_parsed = `[{"brand":"Samsung","category":"smartphones","description":"Samsung's new variant which goes beyond Galaxy to the Universe","discountPercentage":15.46,"id":3,"images":["https://dummyjson.com/image/i/products/3/1.jpg"],"price":1249,"rating":4.09,"stock":36,"thumbnail":"https://dummyjson.com/image/i/products/3/thumbnail.jpg","title":"Samsung Universe 9"}]`
