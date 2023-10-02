// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mock

import "errors"

const (
	beatsContainer  = "beatscontainer"
	beatsContainer2 = "beatscontainer2"
)

var containers = map[string]bool{
	beatsContainer:  true,
	beatsContainer2: true,
}

var availableBlobs = map[string]map[string]bool{
	beatsContainer: {
		"ata.json":      true,
		"data_3.json":   true,
		"docs/ata.json": true,
	},
	beatsContainer2: {
		"ata.json":    true,
		"data_3.json": true,
	},
}

var blobs = map[string]map[string]string{
	beatsContainer: {
		"ata.json":      Beatscontainer_blob_ata_json,
		"data_3.json":   Beatscontainer_blob_data3_json,
		"docs/ata.json": Beatscontainer_blob_docs_ata_json,
	},
	beatsContainer2: {
		"ata.json":    Beatscontainer_2_blob_ata_json,
		"data_3.json": Beatscontainer_2_blob_data3_json,
	},
}

var fetchContainer = map[string]string{
	beatsContainer: `<?xml version="1.0" encoding="utf-8"?>
	<EnumerationResults ServiceEndpoint="https://127.0.0.1/" ContainerName="beatscontainer">
		<Blobs>
			<Blob>
				<Name>ata.json</Name>
				<Properties>
					<Last-Modified>Wed, 12 Sep 2022 12:12:28 GMT</Last-Modified>
					<Etag>0x8DA964A64516C82</Etag>
					<Content-Length>643</Content-Length>
					<Content-Type>application/json</Content-Type>
					<Content-Encoding />
					<Content-Language />
					<Content-MD5>UjQX73kQRTHx+UyXZDvVkg==</Content-MD5>
					<Cache-Control />
					<Content-Disposition />
					<BlobType>BlockBlob</BlobType>
					<LeaseStatus>unlocked</LeaseStatus>
					<LeaseState>available</LeaseState>
				</Properties>
				<Metadata />
			</Blob>
			<Blob>
				<Name>data_3.json</Name>
				<Properties>
					<Last-Modified>Wed, 14 Sep 2022 12:12:44 GMT</Last-Modified>
					<Etag>0x8DA964A6DE60497</Etag>
					<Content-Length>434</Content-Length>
					<Content-Type>application/json</Content-Type>
					<Content-Encoding />
					<Content-Language />
					<Content-MD5>eOXjYygu6k6687Uf3vPtKQ==</Content-MD5>
					<Cache-Control />
					<Content-Disposition />
					<BlobType>BlockBlob</BlobType>
					<LeaseStatus>unlocked</LeaseStatus>
					<LeaseState>available</LeaseState>
				</Properties>
				<Metadata />
			</Blob>
			<Blob>
				<Name>docs/ata.json</Name>
				<Properties>
					<Last-Modified>Wed, 15 Sep 2022 12:13:07 GMT</Last-Modified>
					<Etag>0x8DA964A7B8D8862</Etag>
					<Content-Length>643</Content-Length>
					<Content-Type>application/json</Content-Type>
					<Content-Encoding />
					<Content-Language />
					<Content-MD5>UjQX73kQRTHx+UyXZDvVkg==</Content-MD5>
					<Cache-Control />
					<Content-Disposition />
					<BlobType>BlockBlob</BlobType>
					<LeaseStatus>unlocked</LeaseStatus>
					<LeaseState>available</LeaseState>
				</Properties>
				<Metadata />
			</Blob>
		</Blobs>
		<NextMarker />
	</EnumerationResults>`,

	beatsContainer2: `<?xml version="1.0" encoding="utf-8"?>
	<EnumerationResults ServiceEndpoint="https://127.0.0.1/" ContainerName="beatscontainer2">
		<Blobs>
			<Blob>
				<Name>ata.json</Name>
				<Properties>
					<Last-Modified>Thu, 15 Sep 2022 12:40:41 GMT</Last-Modified>
					<Etag>0x8DA9717802689DE</Etag>
					<Content-Length>643</Content-Length>
					<Content-Type>application/json</Content-Type>
					<Content-Encoding />
					<Content-Language />
					<Content-MD5>UjQX73kQRTHx+UyXZDvVkg==</Content-MD5>
					<Cache-Control />
					<Content-Disposition />
					<BlobType>BlockBlob</BlobType>
					<LeaseStatus>unlocked</LeaseStatus>
					<LeaseState>available</LeaseState>
				</Properties>
				<Metadata />
			</Blob>
			<Blob>
				<Name>data_3.json</Name>
				<Properties>
					<Last-Modified>Thu, 15 Sep 2022 12:40:41 GMT</Last-Modified>
					<Etag>0x8DA97178026B0EA</Etag>
					<Content-Length>434</Content-Length>
					<Content-Type>application/json</Content-Type>
					<Content-Encoding />
					<Content-Language />
					<Content-MD5>eOXjYygu6k6687Uf3vPtKQ==</Content-MD5>
					<Cache-Control />
					<Content-Disposition />
					<BlobType>BlockBlob</BlobType>
					<LeaseStatus>unlocked</LeaseStatus>
					<LeaseState>available</LeaseState>
				</Properties>
				<Metadata />
			</Blob>
		</Blobs>
		<NextMarker />
	</EnumerationResults>`,
}

var Beatscontainer_blob_ata_json = `{
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

var Beatscontainer_blob_data3_json = `{
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

var Beatscontainer_blob_docs_ata_json = `{
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

var Beatscontainer_2_blob_ata_json = `{
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

var Beatscontainer_2_blob_data3_json = `{
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

//nolint:stylecheck // intentionally indented like this
var NotFoundErr = errors.New("--------------------------------------------------------------------------------\nRESPONSE 404: 404 Not Found\nERROR CODE UNAVAILABLE\n--------------------------------------------------------------------------------\nresource not found\n--------------------------------------------------------------------------------\n")
