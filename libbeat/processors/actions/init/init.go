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

package init

import (
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/processors/actions"
	"github.com/elastic/beats/v7/libbeat/processors/checks"
	jsprocessor "github.com/elastic/beats/v7/libbeat/processors/script/javascript/module/processor/registry"
)

func init() {
	processors.RegisterPlugin("add_fields",
		checks.ConfigChecked(actions.CreateAddFields,
			checks.RequireFields(actions.FieldsKey),
			checks.AllowedFields(actions.FieldsKey, "target", "when")))

	jsprocessor.RegisterPlugin("AddFields", actions.CreateAddFields)
}

func init() {
	processors.RegisterPlugin("add_labels",
		checks.ConfigChecked(actions.CreateAddLabels,
			checks.RequireFields(actions.LabelsKey),
			checks.AllowedFields(actions.LabelsKey, "when")))
}

func init() {
	processors.RegisterPlugin("add_network_direction",
		checks.ConfigChecked(actions.NewAddNetworkDirection,
			checks.RequireFields("source", "destination", "target", "internal_networks"),
			checks.AllowedFields("source", "destination", "target", "internal_networks")))
	jsprocessor.RegisterPlugin("AddNetworkDirection", actions.NewAddNetworkDirection)
}

func init() {
	processors.RegisterPlugin("add_tags",
		checks.ConfigChecked(actions.CreateAddTags,
			checks.RequireFields("tags"),
			checks.AllowedFields("tags", "target", "when")))
}

func init() {
	processors.RegisterPlugin("append",
		checks.ConfigChecked(actions.NewAppendProcessor,
			checks.RequireFields("target_field"),
		),
	)
	jsprocessor.RegisterPlugin("AppendProcessor", actions.NewAppendProcessor)
}

func init() {
	processors.RegisterPlugin("copy_fields",
		checks.ConfigChecked(actions.NewCopyFields,
			checks.RequireFields("fields"),
		),
	)
	jsprocessor.RegisterPlugin("CopyFields", actions.NewCopyFields)
}

func init() {
	processors.RegisterPlugin("decode_base64_field",
		checks.ConfigChecked(actions.NewDecodeBase64Field,
			checks.RequireFields("field"),
			checks.AllowedFields("field", "when", "ignore_missing", "fail_on_error")))
	jsprocessor.RegisterPlugin("DecodeBase64Field", actions.NewDecodeBase64Field)
}

func init() {
	processors.RegisterPlugin("decode_json_fields",
		checks.ConfigChecked(actions.NewDecodeJSONFields,
			checks.RequireFields("fields"),
			checks.AllowedFields("fields", "max_depth", "overwrite_keys", "add_error_key", "process_array", "target", "when", "document_id", "expand_keys")))

	jsprocessor.RegisterPlugin("DecodeJSONFields", actions.NewDecodeJSONFields)
}

func init() {
	processors.RegisterPlugin("detect_mime_type",
		checks.ConfigChecked(actions.NewDetectMimeType,
			checks.RequireFields("field", "target"),
			checks.AllowedFields("field", "target")))
}

func init() {
	processors.RegisterPlugin("decompress_gzip_field",
		checks.ConfigChecked(actions.NewDecompressGzipFields,
			checks.RequireFields("field"),
			checks.AllowedFields("field", "ignore_missing", "overwrite_keys", "overwrite_keys", "fail_on_error")))
}

func init() {
	processors.RegisterPlugin("drop_event",
		checks.ConfigChecked(actions.NewDropEvent, checks.AllowedFields("when")))
}

func init() {
	processors.RegisterPlugin("drop_fields",
		checks.ConfigChecked(actions.NewDropFields,
			checks.RequireFields("fields"),
			checks.AllowedFields("fields", "when", "ignore_missing")))

	jsprocessor.RegisterPlugin("DropFields", actions.NewDropFields)
}

func init() {
	processors.RegisterPlugin("include_fields",
		checks.ConfigChecked(actions.NewIncludeFields,
			checks.RequireFields("fields"),
			checks.AllowedFields("fields", "when")))
}

func init() {
	processors.RegisterPlugin(
		"lowercase",
		checks.ConfigChecked(
			actions.NewLowerCaseProcessor,
			checks.RequireFields("fields"),
			checks.AllowedFields("fields", "ignore_missing", "fail_on_error", "alter_full_field", "values"),
		),
	)
}

func init() {
	processors.RegisterPlugin("rename",
		checks.ConfigChecked(actions.NewRenameFields,
			checks.RequireFields("fields")))

	jsprocessor.RegisterPlugin("Rename", actions.NewRenameFields)
}

func init() {
	processors.RegisterPlugin("replace",
		checks.ConfigChecked(actions.NewReplaceString,
			checks.RequireFields("fields")))

	jsprocessor.RegisterPlugin("Replace", actions.NewReplaceString)
}

func init() {
	processors.RegisterPlugin("truncate_fields",
		checks.ConfigChecked(actions.NewTruncateFields,
			checks.RequireFields("fields"),
			checks.MutuallyExclusiveRequiredFields("max_bytes", "max_characters"),
		),
	)
	jsprocessor.RegisterPlugin("TruncateFields", actions.NewTruncateFields)
}

func init() {
	processors.RegisterPlugin(
		"uppercase",
		checks.ConfigChecked(
			actions.NewUpperCaseProcessor,
			checks.RequireFields("fields"),
			checks.AllowedFields("fields", "ignore_missing", "fail_on_error", "alter_full_field", "values"),
		),
	)
}
