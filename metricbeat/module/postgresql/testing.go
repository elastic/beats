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

package postgresql

import "os"

//GetEnvDSN returns the Data Source Name
func GetEnvDSN() string {
	return os.Getenv("POSTGRESQL_DSN")
}

//GetEnvUsername returns the username
func GetEnvUsername() string {
	return os.Getenv("POSTGRESQL_USERNAME")
}

//GetEnvPassword returns the password
func GetEnvPassword() string {
	return os.Getenv("POSTGRESQL_PASSWORD")
}
