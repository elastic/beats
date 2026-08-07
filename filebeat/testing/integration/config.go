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

package integration

import (
	"fmt"
	"strings"
)

// FilestreamOptions holds optional settings for a filestream input.
// Zero values are omitted from the generated config, letting Filebeat use its defaults.
type FilestreamOptions struct {
	// ScanInterval sets prospector.scanner.check_interval. Defaults to "100ms" when empty.
	ScanInterval string
	// IgnoreOlder skips files not modified within this duration, e.g. "2h".
	IgnoreOlder string
	// CloseInactive sets close.on_state_change.inactive, e.g. "1s".
	CloseInactive string
	// CloseRemoved sets close.on_state_change.removed.
	CloseRemoved bool
	// CloseRenamed sets close.on_state_change.renamed.
	CloseRenamed bool
	// CloseEOF sets close.reader.eof, causing the reader to close on EOF.
	CloseEOF bool
	// Symlinks enables prospector.scanner.symlinks.
	Symlinks bool
	// IncludeLines filters to only lines matching these regexes.
	IncludeLines []string
	// ExcludeLines drops lines matching these regexes.
	ExcludeLines []string
	// ExcludeFiles skips files whose names match these regexes.
	ExcludeFiles []string
	// MaxBytes sets the maximum number of bytes per event.
	MaxBytes int
	// NDJSON enables NDJSON (JSON) parsing via the filestream parsers.ndjson block.
	NDJSON *NDJSONOptions
	// Multiline enables multiline log reassembly.
	Multiline *MultilineOptions
	// CleanInactive removes registry entries after the file has been inactive for this long.
	CleanInactive string
	// CleanRemoved removes registry entries when the file is deleted.
	CleanRemoved bool
	// RegistryPath sets the global filebeat.registry.path.
	// The registry will be written to RegistryPath/filebeat/log.json.
	RegistryPath string
	// RegistryCleanupInterval sets filebeat.registry.cleanup_interval, which
	// controls how often the GC scans for expired clean_inactive entries.
	// Defaults to 5 minutes when empty; use a short value (e.g. "1s") in tests
	// that assert on clean_inactive behaviour.
	RegistryCleanupInterval string
	// GlobalProcessors is raw YAML inserted as the top-level processors block.
	// Each entry must be a properly indented YAML list item, e.g.:
	//   "  - drop_fields:\n      fields: [agent]"
	GlobalProcessors string
}

// NDJSONOptions configures the filestream parsers.ndjson block.
type NDJSONOptions struct {
	MessageKey          string
	KeysUnderRoot       bool
	OverwriteKeys       bool
	AddErrorKey         bool
	IgnoreDecodingError bool
}

// MultilineOptions configures multiline log reassembly.
type MultilineOptions struct {
	Pattern string
	Negate  bool
	// Match is "after" or "before".
	Match string
}

// FilestreamInputConfig returns a complete Filebeat YAML configuration that reads
// from path using a filestream input and writes events to stdout (output.console).
// id must be unique per test; use a short descriptive string.
func FilestreamInputConfig(id, path string, opts FilestreamOptions) string {
	var sb strings.Builder

	scanInterval := "100ms"
	if opts.ScanInterval != "" {
		scanInterval = opts.ScanInterval
	}

	fmt.Fprintf(&sb, `filebeat.inputs:
  - type: filestream
    id: %s
    paths:
      - %s
    prospector.scanner.check_interval: %s
    file_identity.native: ~
    prospector.scanner.fingerprint.enabled: false
`, id, path, scanInterval)

	if opts.IgnoreOlder != "" {
		fmt.Fprintf(&sb, "    ignore_older: %s\n", opts.IgnoreOlder)
	}
	if opts.CloseInactive != "" {
		fmt.Fprintf(&sb, "    close.on_state_change.inactive: %s\n", opts.CloseInactive)
	}
	if opts.CloseRemoved {
		fmt.Fprintf(&sb, "    close.on_state_change.removed: true\n")
	}
	if opts.CloseRenamed {
		fmt.Fprintf(&sb, "    close.on_state_change.renamed: true\n")
	}
	if opts.CloseEOF {
		fmt.Fprintf(&sb, "    close.reader.eof: true\n")
	}
	if opts.Symlinks {
		fmt.Fprintf(&sb, "    prospector.scanner.symlinks: true\n")
	}
	if len(opts.IncludeLines) > 0 {
		fmt.Fprintf(&sb, "    include_lines: [%s]\n", joinQuoted(opts.IncludeLines))
	}
	if len(opts.ExcludeLines) > 0 {
		fmt.Fprintf(&sb, "    exclude_lines: [%s]\n", joinQuoted(opts.ExcludeLines))
	}
	if len(opts.ExcludeFiles) > 0 {
		fmt.Fprintf(&sb, "    prospector.scanner.exclude_files: [%s]\n", joinQuoted(opts.ExcludeFiles))
	}
	if opts.MaxBytes > 0 {
		fmt.Fprintf(&sb, "    max_bytes: %d\n", opts.MaxBytes)
	}
	if opts.CleanInactive != "" {
		fmt.Fprintf(&sb, "    clean_inactive: %s\n", opts.CleanInactive)
	}
	if opts.CleanRemoved {
		fmt.Fprintf(&sb, "    clean_removed: true\n")
	}
	if opts.NDJSON != nil {
		sb.WriteString("    parsers:\n      - ndjson:\n")
		if opts.NDJSON.MessageKey != "" {
			fmt.Fprintf(&sb, "          message_key: %s\n", opts.NDJSON.MessageKey)
		}
		if opts.NDJSON.KeysUnderRoot {
			sb.WriteString("          keys_under_root: true\n")
		}
		if opts.NDJSON.OverwriteKeys {
			sb.WriteString("          overwrite_keys: true\n")
		}
		if opts.NDJSON.AddErrorKey {
			sb.WriteString("          add_error_key: true\n")
		}
		if opts.NDJSON.IgnoreDecodingError {
			sb.WriteString("          ignore_decoding_error: true\n")
		}
	}
	if opts.Multiline != nil {
		sb.WriteString("    parsers:\n      - multiline:\n")
		fmt.Fprintf(&sb, "          pattern: '%s'\n", opts.Multiline.Pattern)
		if opts.Multiline.Negate {
			sb.WriteString("          negate: true\n")
		}
		fmt.Fprintf(&sb, "          match: %s\n", opts.Multiline.Match)
		sb.WriteString("          timeout: 1s\n")
	}

	sb.WriteString("\noutput.console:\n  enabled: true\n")

	if opts.RegistryPath != "" {
		fmt.Fprintf(&sb, "\nfilebeat.registry.path: %s\n", opts.RegistryPath)
	}
	if opts.RegistryCleanupInterval != "" {
		fmt.Fprintf(&sb, "\nfilebeat.registry.cleanup_interval: %s\n", opts.RegistryCleanupInterval)
	}

	if opts.GlobalProcessors != "" {
		fmt.Fprintf(&sb, "\nprocessors:\n%s\n", opts.GlobalProcessors)
	}

	return sb.String()
}

func joinQuoted(ss []string) string {
	quoted := make([]string, len(ss))
	for i, s := range ss {
		quoted[i] = fmt.Sprintf("%q", s)
	}
	return strings.Join(quoted, ", ")
}
