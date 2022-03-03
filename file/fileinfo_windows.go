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

package file

import (
	"os"
)

func stat(name string, statFunc func(name string) (os.FileInfo, error)) (FileInfo, error) {
	info, err := statFunc(name)
	if err != nil {
		return nil, err
	}

	return wrap(info)
}

func wrap(info os.FileInfo) (FileInfo, error) {
	return fileInfo{FileInfo: info}, nil
}
