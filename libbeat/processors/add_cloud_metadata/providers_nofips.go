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

//go:build !requirefips

package add_cloud_metadata

func init() {
	// Include the Azure provider ONLY in non-FIPS builds, as the Azure provider depends on
	// the Azure SDK which, in turn, depends on the golang.org/x/crypto/pkcs12 package, which
	// is not FIPS-compliant, and the SDK doesn't plan to offer a way to disable the use of
	// this package at compile time (see https://github.com/Azure/azure-sdk-for-go/issues/24336).
	cloudMetaProviders["azure"] = azureVMMetadataFetcher
	priorityProviders = append(priorityProviders, "azure")
}
