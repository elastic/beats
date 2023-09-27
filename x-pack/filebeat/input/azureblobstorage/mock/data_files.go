// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mock

const (
	beatsMultilineJSONContainer = "beatsmultilinejsoncontainer"
	beatsJSONContainer          = "beatsjsoncontainer"
	beatsNdJSONContainer        = "beatsndjsoncontainer"
	beatsGzJSONContainer        = "beatsgzjsoncontainer"
	beatsJSONWithArrayContainer = "beatsjsonwitharraycontainer"
)

var fileContainers = map[string]bool{
	beatsMultilineJSONContainer: true,
	beatsJSONContainer:          true,
	beatsNdJSONContainer:        true,
	beatsGzJSONContainer:        true,
	beatsJSONWithArrayContainer: true,
}

var availableFileBlobs = map[string]map[string]bool{
	beatsMultilineJSONContainer: {
		"multiline.json": true,
	},
	beatsJSONContainer: {
		"log.json":          true,
		"events-array.json": true,
	},
	beatsNdJSONContainer: {
		"log.ndjson": true,
	},
	beatsGzJSONContainer: {
		"multiline.json.gz": true,
	},
	beatsJSONWithArrayContainer: {
		"array-at-root.json": true,
		"nested-arrays.json": true,
	},
}

var fetchFilesContainer = map[string]string{
	beatsMultilineJSONContainer: `<?xml version="1.0" encoding="utf-8"?>
	<EnumerationResults ServiceEndpoint="https://127.0.0.1/" ContainerName="beatsmultilinejsoncontainer">
		<Blobs>
			<Blob>
				<Name>multiline.json</Name>
				<Properties>
					<Last-Modified>Wed, 14 Sep 2022 12:12:28 GMT</Last-Modified>
					<Etag>0x8DA964A64516C82</Etag>
					<Content-Length>643</Content-Length>
					<Content-Type>application/octet-stream</Content-Type>
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
	beatsJSONContainer: `<?xml version="1.0" encoding="utf-8"?>
		<EnumerationResults ServiceEndpoint="https://127.0.0.1/" ContainerName="beatsjsoncontainer">
			<Blobs>
				<Blob>
					<Name>log.json</Name>
					<Properties>
						<Last-Modified>Wed, 14 Sep 2022 12:12:28 GMT</Last-Modified>
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
					<Name>events-array.json</Name>
					<Properties>
						<Last-Modified>Wed, 14 Sep 2022 12:12:28 GMT</Last-Modified>
						<Etag>0x8DA964A64516C83</Etag>
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
	beatsNdJSONContainer: `<?xml version="1.0" encoding="utf-8"?>
			<EnumerationResults ServiceEndpoint="https://127.0.0.1/" ContainerName="beatsndjsoncontainer">
				<Blobs>
					<Blob>
						<Name>log.ndjson</Name>
						<Properties>
							<Last-Modified>Wed, 14 Sep 2022 12:12:28 GMT</Last-Modified>
							<Etag>0x8DA964A64516C82</Etag>
							<Content-Length>643</Content-Length>
							<Content-Type>application/x-ndjson</Content-Type>
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
	beatsJSONWithArrayContainer: `<?xml version="1.0" encoding="utf-8"?>
	<EnumerationResults ServiceEndpoint="https://127.0.0.1/" ContainerName="beatsjsonwitharraycontainer">
		<Blobs>
			<Blob>
				<Name>array-at-root.json</Name>
				<Properties>
					<Last-Modified>Wed, 14 Sep 2022 12:12:28 GMT</Last-Modified>
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
				<Name>nested-arrays.json</Name>
				<Properties>
					<Last-Modified>Wed, 14 Sep 2022 12:12:28 GMT</Last-Modified>
					<Etag>0x8DA964A64516C83</Etag>
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
	beatsGzJSONContainer: `<?xml version="1.0" encoding="utf-8"?>
				<EnumerationResults ServiceEndpoint="https://127.0.0.1/" ContainerName="beatsgzjsoncontainer">
					<Blobs>
						<Blob>
							<Name>multiline.json.gz</Name>
							<Properties>
								<Last-Modified>Wed, 14 Sep 2022 12:12:28 GMT</Last-Modified>
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
						</Blobs>
						<NextMarker />
					</EnumerationResults>`,
}

var BeatsFilesContainer_multiline_json = []string{
	"{\n    \"@timestamp\": \"2021-05-25T17:25:42.806Z\",\n    \"log.level\": \"error\",\n    \"message\": \"error making request\"\n}",
	"{\n    \"@timestamp\": \"2021-05-25T17:25:51.391Z\",\n    \"log.level\": \"info\",\n    \"message\": \"available space 44.3gb\"\n}",
}

var BeatsFilesContainer_log_json = []string{
	`{"@timestamp":"2021-05-25T17:25:42.806Z","log.level":"error","message":"error making http request"}`,
	`{"@timestamp":"2021-05-25T17:25:51.391Z","log.level":"info","message":"available disk space 44.3gb"}`,
	"{\n    \"Events\": [\n        {\n            \"time\": \"2021-05-25 18:20:58 UTC\",\n            \"msg\": \"hello\"\n        },\n        {\n            \"time\": \"2021-05-26 22:21:40 UTC\",\n            \"msg\": \"world\"\n        }\n    ]\n}",
}

var BeatsFilesContainer_log_ndjson = []string{
	`{"@timestamp":"2021-05-25T17:25:42.806Z","log.level":"error","message":"error in http request"}`,
	`{"@timestamp":"2021-05-25T17:25:51.391Z","log.level":"info","message":"available space is 44.3gb"}`,
}

var BeatsFilesContainer_events_array_json = []string{
	"{\n            \"time\": \"2021-05-25 18:20:58 UTC\",\n            \"msg\": \"hello\"\n        }",
	"{\n            \"time\": \"2021-05-26 22:21:40 UTC\",\n            \"msg\": \"world\"\n        }",
}

var BeatsFilesContainer_json_array = []string{
	"{\n        \"time\": \"2021-05-25 18:20:58 UTC\",\n        \"msg\": \"hello\"\n    }",
	"{\n        \"time\": \"2021-05-26 22:21:40 UTC\",\n        \"msg\": \"world\"\n    }",
	"[\n       {\n           \"time\": \"2021-05-25 18:20:58 UTC\",\n           \"msg\": \"hello\"\n       },\n       {\n           \"time\": \"2021-05-26 22:21:40 UTC\",\n           \"msg\": \"world\"\n       }\n   ]",
	"[\n       {\n           \"time\": \"2021-05-25 18:20:58 UTC\",\n           \"msg\": \"hi\"\n       },\n       {\n           \"time\": \"2021-05-26 22:21:40 UTC\",\n           \"msg\": \"seoul\"\n       }\n   ]",
}

var BeatsFilesContainer_multiline_json_gz = []string{
	"{\n    \"@timestamp\": \"2021-05-25T17:25:42.806Z\",\n    \"log.level\": \"error\",\n    \"message\": \"error making http request\"\n}",
	"{\n    \"@timestamp\": \"2021-05-25T17:25:51.391Z\",\n    \"log.level\": \"info\",\n    \"message\": \"available disk space 44.3gb\"\n}",
}
