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

package dbstats

import (
	s "github.com/menderesk/beats/v7/libbeat/common/schema"
	c "github.com/menderesk/beats/v7/libbeat/common/schema/mapstriface"
)

var schema = s.Schema{
	"db":          c.Str("db"),
	"collections": c.Int("collections"),
	"objects":     c.Int("objects"),
	"avg_obj_size": s.Object{
		"bytes": c.Int("avgObjSize"),
	},
	"data_size": s.Object{
		"bytes": c.Int("dataSize"),
	},
	"storage_size": s.Object{
		"bytes": c.Int("storageSize"),
	},
	"num_extents": c.Int("numExtents"),
	"indexes":     c.Int("indexes"),
	"index_size": s.Object{
		"bytes": c.Int("indexSize"),
	},
	// mmapv1 only
	"ns_size_mb": s.Object{
		"mb": c.Int("nsSizeMB", s.Optional),
	},
	// mmapv1 only
	"file_size": s.Object{
		"bytes": c.Int("fileSize", s.Optional),
	},
	// mmapv1 only
	"data_file_version": c.Dict("dataFileVersion", s.Schema{
		"major": c.Int("major"),
		"minor": c.Int("minor"),
	}, c.DictOptional),
	// mmapv1 only
	"extent_free_list": c.Dict("extentFreeList", s.Schema{
		"num": c.Int("num"),
		"size": s.Object{
			"bytes": c.Int("size", s.Optional),
		},
	}, c.DictOptional),
}
