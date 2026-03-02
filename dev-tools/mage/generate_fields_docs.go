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

package mage

import (
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v2"
)

// fieldsYAMLSection represents a top-level section in fields.yml.
type fieldsYAMLSection struct {
	Key         string            `yaml:"key"`
	Title       string            `yaml:"title"`
	Anchor      string            `yaml:"anchor,omitempty"`
	Description string            `yaml:"description,omitempty"`
	Name        string            `yaml:"name,omitempty"`
	Release     string            `yaml:"release,omitempty"`
	HasSkipDocs bool              `yaml:"-"`
	Version     map[string]string `yaml:"version,omitempty"`
	Deprecated  string            `yaml:"deprecated,omitempty"`
	Fields      []fieldsYAMLField `yaml:"fields,omitempty"`
}

// fieldsYAMLField represents a field entry in a fields.yml section.
type fieldsYAMLField struct {
	Name           string            `yaml:"name,omitempty"`
	HasName        bool              `yaml:"-"`
	Type           string            `yaml:"type,omitempty"`
	Description    string            `yaml:"description,omitempty"`
	HasDescription bool              `yaml:"-"`
	Example        interface{}       `yaml:"example,omitempty"`
	Format         string            `yaml:"format,omitempty"`
	Required       interface{}       `yaml:"required,omitempty"`
	Path           string            `yaml:"path,omitempty"`
	FieldPath      string            `yaml:"field_path,omitempty"`
	Index          *bool             `yaml:"index,omitempty"`
	Enabled        *bool             `yaml:"enabled,omitempty"`
	Release        string            `yaml:"release,omitempty"`
	HasSkipDocs    bool              `yaml:"-"`
	Version        map[string]string `yaml:"version,omitempty"`
	Deprecated     string            `yaml:"deprecated,omitempty"`
	MultiFields    []fieldsYAMLField `yaml:"multi_fields,omitempty"`
	Fields         []fieldsYAMLField `yaml:"fields,omitempty"`
}

// GenerateFieldsDocs generates Markdown documentation files from a fields.yml file.
// Replaces libbeat/scripts/generate_fields_docs.py.
func GenerateFieldsDocs(fieldsYMLPath, outputPath, beat string) error {
	data, err := os.ReadFile(fieldsYMLPath)
	if err != nil {
		return fmt.Errorf("failed to read fields file %s: %w", fieldsYMLPath, err)
	}

	var docs []fieldsYAMLSection
	if err := yaml.Unmarshal(data, &docs); err != nil {
		return fmt.Errorf("failed to parse fields YAML: %w", err)
	}

	if len(docs) == 0 {
		log.Println("fields.yml file is empty. exported-fields docs cannot be generated.")
		return nil
	}

	markKeyPresence(data, docs)
	if err := deduplicateFields(docs); err != nil {
		return err
	}

	if err := os.MkdirAll(outputPath, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", outputPath, err)
	}

	if err := writeFieldsIndex(docs, outputPath, beat); err != nil {
		return fmt.Errorf("failed to write exported-fields.md: %w", err)
	}

	if err := writeFieldsSectionFiles(docs, outputPath, beat); err != nil {
		return fmt.Errorf("failed to write section field docs: %w", err)
	}

	return nil
}

// markKeyPresence walks the raw YAML to detect keys that Go's struct unmarshaling
// cannot distinguish from absent keys. Python's yaml.load preserves key presence
// (e.g. `description: null` vs missing `description`), so we track it via Has* flags.
// It also normalises null descriptions to "None" and converts non-standard
// timestamps in Example values to match Python's datetime formatting.
func markKeyPresence(data []byte, docs []fieldsYAMLSection) {
	var rawSections []map[interface{}]interface{}
	if err := yaml.Unmarshal(data, &rawSections); err != nil {
		return
	}
	for i, rawSection := range rawSections {
		if i >= len(docs) {
			break
		}
		desc, hasDesc := rawSection["description"]
		if hasDesc && desc == nil {
			docs[i].Description = "None"
		}
		if _, has := rawSection["skipdocs"]; has {
			docs[i].HasSkipDocs = true
		}
		rawFields, _ := rawSection["fields"].([]interface{})
		markFieldsKeyPresence(rawFields, docs[i].Fields)
	}
}

func markFieldsKeyPresence(rawFields []interface{}, fields []fieldsYAMLField) {
	for i, raw := range rawFields {
		if i >= len(fields) {
			break
		}
		m, ok := raw.(map[interface{}]interface{})
		if !ok {
			continue
		}
		if _, has := m["name"]; has {
			fields[i].HasName = true
		}
		desc, hasDesc := m["description"]
		if hasDesc {
			fields[i].HasDescription = true
			if desc == nil && fields[i].Description == "" {
				fields[i].Description = "None"
			}
		}
		if _, has := m["skipdocs"]; has {
			fields[i].HasSkipDocs = true
		}
		if s, ok := fields[i].Example.(string); ok {
			if t, ok := parsePythonOnlyTimestamp(s); ok {
				fields[i].Example = formatPythonDatetime(t)
			}
		}
		subRaw, _ := m["fields"].([]interface{})
		if len(subRaw) > 0 && len(fields[i].Fields) > 0 {
			markFieldsKeyPresence(subRaw, fields[i].Fields)
		}
		subRaw, _ = m["multi_fields"].([]interface{})
		if len(subRaw) > 0 && len(fields[i].MultiFields) > 0 {
			markFieldsKeyPresence(subRaw, fields[i].MultiFields)
		}
	}
}

// singleDigitDateRe matches timestamps where the month or day is a single digit
// (e.g. "2013-6-25T..." or "2013-06-5T..."). Python's yaml.FullLoader parses
// these as datetime, but Go's yaml.v2 keeps them as plain strings.
var singleDigitDateRe = regexp.MustCompile(
	`^\d{4}-(\d-\d|\d{2}-\d[^0-9]|\d-\d{2})`)

// parsePythonOnlyTimestamp converts timestamp strings that Python's yaml.FullLoader
// would parse as datetime but Go's yaml.v2 leaves as strings. Only targets
// non-standard date formats (single-digit month or day).
func parsePythonOnlyTimestamp(s string) (time.Time, bool) {
	if !singleDigitDateRe.MatchString(s) {
		return time.Time{}, false
	}
	for _, layout := range []string{
		"2006-1-2T15:04:05Z",
		"2006-1-2T15:04:05-07:00",
		"2006-1-2 15:04:05Z",
		"2006-1-2 15:04:05-07:00",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func deduplicateFields(docs []fieldsYAMLSection) error {
	for i := range docs {
		if len(docs[i].Fields) == 0 {
			continue
		}
		seen := make(map[string]int) // name -> index in deduplicated slice
		var deduped []fieldsYAMLField
		for _, field := range docs[i].Fields {
			if idx, ok := seen[field.Name]; ok {
				if deduped[idx].Type != field.Type {
					return fmt.Errorf("field %q in section %q redefined with different type %q (previously %q)", field.Name, docs[i].Key, field.Type, deduped[idx].Type)
				}
				deduped[idx] = field
			} else {
				seen[field.Name] = len(deduped)
				deduped = append(deduped, field)
			}
		}
		docs[i].Fields = deduped
	}
	return nil
}

func writeFieldsIndex(docs []fieldsYAMLSection, outputPath, beat string) error {
	f, err := os.Create(filepath.Join(outputPath, "exported-fields.md"))
	if err != nil {
		return err
	}
	defer f.Close()

	titleCaser := cases.Title(language.English)
	beatTitle := titleCaser.String(beat)

	fmt.Fprintf(f, `---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/%s/current/exported-fields.html
applies_to:
  stack: ga
  serverless: ga
---

%% This file is generated! See dev-tools/mage/generate_fields_docs.go

# Exported fields [exported-fields]

This document describes the fields that are exported by %s. They are grouped in the following categories:

`, beat, beatTitle)

	sorted := sortedSections(docs)
	for _, section := range sorted {
		anchor := section.Anchor
		if anchor == "" {
			anchor = section.Key
		}
		if section.HasSkipDocs {
			continue
		}
		fmt.Fprintf(f, "* [*%s fields*](/reference/%s/exported-fields-%s.md)\n", section.Title, beat, anchor)
	}

	return nil
}

func writeFieldsSectionFiles(docs []fieldsYAMLSection, outputPath, beat string) error {
	sorted := sortedSections(docs)
	for _, section := range sorted {
		if section.Anchor == "" {
			section.Anchor = section.Key
		}
		section.Name = section.Title
		if section.Description == "" {
			section.Description = section.Key
		}
		if len(section.Fields) == 0 {
			continue
		}

		outFile := filepath.Join(outputPath, fmt.Sprintf("exported-fields-%s.md", section.Anchor))
		f, err := os.Create(outFile)
		if err != nil {
			return fmt.Errorf("failed to create %s: %w", outFile, err)
		}

		writeFieldsSection(f, &section, "", beat)
		if err := f.Close(); err != nil {
			return fmt.Errorf("failed to close %s: %w", outFile, err)
		}
	}
	return nil
}

func writeFieldsSection(w io.Writer, section *fieldsYAMLSection, path, beat string) {
	if section.HasSkipDocs {
		return
	}

	if section.Anchor != "" {
		fmt.Fprint(w, "---\n")
		fmt.Fprint(w, "mapped_pages:\n")
		fmt.Fprintf(w, "  - https://www.elastic.co/guide/en/beats/%s/current/exported-fields-%s.html\n", beat, section.Anchor)

		appliesTo := getAppliesToFromSection(section, true)
		if len(appliesTo) > 0 {
			fmt.Fprint(w, "applies_to:\n")
			fmt.Fprintf(w, "  stack: %s\n", strings.Join(appliesTo, ", "))
			fmt.Fprintf(w, "  serverless: %s\n", getServerlessLifecycle(appliesTo))
		}
		fmt.Fprint(w, "---\n\n")
		fmt.Fprint(w, "% This file is generated! See dev-tools/mage/generate_fields_docs.go\n\n")
	}

	if section.Description != "" {
		desc := section.Description
		if section.Anchor != "" && section.Name == "ECS" {
			titleCaser := cases.Title(language.English)
			fmt.Fprintf(w, "# %s fields [exported-fields-ecs]\n\n", section.Name)
			fmt.Fprintf(w, `This section defines Elastic Common Schema (ECS) fields—a common set of fields
to be used when storing event data in {{es}}.

This is an exhaustive list, and fields listed here are not necessarily used by %s.
The goal of ECS is to enable and encourage users of {{es}} to normalize their event data,
so that they can better analyze, visualize, and correlate the data represented in their events.

See the [ECS reference](ecs://reference/index.md) for more information.
`, titleCaser.String(beat))
		} else if section.Anchor != "" {
			fmt.Fprintf(w, "# %s fields [exported-fields-%s]\n\n", section.Title, section.Anchor)
			fmt.Fprintf(w, "%s\n\n", strings.TrimSpace(desc))
		} else {
			fmt.Fprintf(w, "## %s [_%s]\n\n", section.Name, section.Name)

			appliesTo := getAppliesToFromSection(section, false)
			if len(appliesTo) > 0 {
				fmt.Fprintf(w, "```{applies_to}\nstack: %s\n```\n\n", strings.Join(appliesTo, ", "))
			}

			fmt.Fprintf(w, "%s\n\n", strings.TrimSpace(desc))
		}
	}

	if len(section.Fields) == 0 {
		return
	}

	for i := range section.Fields {
		field := &section.Fields[i]
		if !field.HasName {
			continue
		}

		newpath := fieldPath(path, field.Name)

		if field.Type == "group" {
			writeFieldsGroup(w, field, newpath, beat)
		} else if !field.HasSkipDocs {
			writeField(w, field, newpath)
		}
	}
}

func writeFieldsGroup(w io.Writer, field *fieldsYAMLField, path, beat string) {
	if field.HasSkipDocs {
		return
	}

	if field.HasDescription {
		fmt.Fprintf(w, "## %s [_%s]\n\n", field.Name, field.Name)

		appliesTo := getAppliesToFromField(field, false)
		if len(appliesTo) > 0 {
			fmt.Fprintf(w, "```{applies_to}\nstack: %s\n```\n\n", strings.Join(appliesTo, ", "))
		}

		fmt.Fprintf(w, "%s\n\n", strings.TrimSpace(field.Description))
	}

	if len(field.Fields) == 0 {
		return
	}

	for i := range field.Fields {
		subfield := &field.Fields[i]
		if !subfield.HasName {
			continue
		}

		newpath := fieldPath(path, subfield.Name)

		if subfield.Type == "group" {
			writeFieldsGroup(w, subfield, newpath, beat)
		} else if !subfield.HasSkipDocs {
			writeField(w, subfield, newpath)
		}
	}
}

func writeField(w io.Writer, field *fieldsYAMLField, fieldPath string) {
	fp := field.FieldPath
	if fp == "" {
		fp = fieldPath
	}

	fmt.Fprintf(w, "**`%s`**", fp)

	appliesTo := getAppliesToFromField(field, false)
	if len(appliesTo) > 0 {
		fmt.Fprintf(w, " {applies_to}`stack: %s`", strings.Join(appliesTo, ", "))
	}

	fmt.Fprint(w, "\n:   ")

	if field.HasDescription && strings.TrimSpace(field.Description) != "" {
		parts := strings.Split(field.Description, "\n")
		var nonEmpty []string
		for _, p := range parts {
			if p != "" {
				nonEmpty = append(nonEmpty, p)
			}
		}
		fmt.Fprintf(w, "%s\n\n", strings.TrimSpace(strings.Join(nonEmpty, " ")))
	}
	if field.Type != "" {
		if field.HasDescription {
			fmt.Fprint(w, "    ")
		}
		fmt.Fprintf(w, "type: %s\n\n", field.Type)
	}
	if field.Example != nil {
		fmt.Fprintf(w, "    example: %s\n\n", formatYAMLValue(field.Example))
	}
	if field.Format != "" {
		fmt.Fprintf(w, "    format: %s\n\n", field.Format)
	}
	if field.Required != nil {
		fmt.Fprintf(w, "    required: %s\n\n", formatYAMLValue(field.Required))
	}
	if field.Path != "" {
		fmt.Fprintf(w, "    alias to: %s\n\n", field.Path)
	}
	if field.Index != nil && !*field.Index {
		fmt.Fprint(w, "Field is not indexed.\n\n")
	}
	if field.Enabled != nil && !*field.Enabled {
		fmt.Fprint(w, "Object is not enabled.\n\n")
	}

	fmt.Fprint(w, "\n")

	for i := range field.MultiFields {
		subfield := &field.MultiFields[i]
		writeField(w, subfield, fp+"."+subfield.Name)
	}
}

func sortedSections(docs []fieldsYAMLSection) []fieldsYAMLSection {
	sorted := make([]fieldsYAMLSection, len(docs))
	copy(sorted, docs)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Key < sorted[j].Key
	})
	return sorted
}

func fieldPath(base, name string) string {
	if name == "" {
		return base
	}
	if base == "" {
		return name
	}
	return base + "." + name
}

func getAppliesToFromSection(section *fieldsYAMLSection, pageLevel bool) []string {
	return getAppliesTo(section.Version, section.Release, section.Deprecated, pageLevel)
}

func getAppliesToFromField(field *fieldsYAMLField, pageLevel bool) []string {
	return getAppliesTo(field.Version, field.Release, field.Deprecated, pageLevel)
}

func getAppliesTo(version map[string]string, release, deprecated string, pageLevel bool) []string {
	var appliesTo []string

	if len(version) > 0 {
		for _, lifecycle := range []string{"preview", "beta", "ga", "deprecated", "removed"} {
			if v, ok := version[lifecycle]; ok {
				appliesTo = append(appliesTo, fmt.Sprintf("%s %s", lifecycle, v))
			}
		}
	} else if release != "" {
		if pageLevel {
			appliesTo = append(appliesTo, release)
		} else if release != "ga" {
			appliesTo = append(appliesTo, release)
		}
	} else if pageLevel {
		appliesTo = append(appliesTo, "ga")
	}

	if deprecated != "" {
		if len(version) == 0 || version["deprecated"] == "" {
			appliesTo = append(appliesTo, fmt.Sprintf("deprecated %s", deprecated))
		}
	}

	return appliesTo
}

func getServerlessLifecycle(appliesTo []string) string {
	lifecycleOrder := []string{"preview", "beta", "ga", "deprecated", "removed"}
	latestIdx := -1
	for _, item := range appliesTo {
		lifecycle := strings.Fields(item)[0]
		for i, l := range lifecycleOrder {
			if lifecycle == l && i > latestIdx {
				latestIdx = i
			}
		}
	}
	if latestIdx >= 0 {
		return lifecycleOrder[latestIdx]
	}
	return "ga"
}

// GetServerlessLifecycleFromString determines the serverless lifecycle
// from a comma-separated applies_to string.
func GetServerlessLifecycleFromString(appliesTo string) string {
	if appliesTo == "" {
		return "ga"
	}
	var parts []string
	for _, p := range strings.Split(appliesTo, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			parts = append(parts, p)
		}
	}
	return getServerlessLifecycle(parts)
}

// moduleFieldsMeta is the minimal structure to read title and lifecycle from a module's fields.yml.
type moduleFieldsMeta struct {
	Title   string            `yaml:"title"`
	Release string            `yaml:"release"`
	Version map[string]string `yaml:"version"`
}

// LoadModuleMeta reads a module's _meta/fields.yml and returns the title and applies_to string.
func LoadModuleMeta(fieldsYMLPath string) (title, appliesTo string, err error) {
	data, err := os.ReadFile(fieldsYMLPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to read %s: %w", fieldsYMLPath, err)
	}

	var fields []moduleFieldsMeta
	if err := yaml.Unmarshal(data, &fields); err != nil {
		return "", "", fmt.Errorf("failed to parse %s: %w", fieldsYMLPath, err)
	}
	if len(fields) == 0 {
		return "", "", fmt.Errorf("no entries found in %s", fieldsYMLPath)
	}

	title = fields[0].Title
	appliesTo = buildAppliesToString(fields[0].Version, fields[0].Release)
	return title, appliesTo, nil
}

// formatYAMLValue formats a value parsed from YAML to match Python's
// str() representation: booleans are capitalized (True/False), floats
// always have a decimal point, integers stay as-is, lists use Python
// bracket notation, and timestamps use Python's datetime str() format.
func formatYAMLValue(v interface{}) string {
	switch val := v.(type) {
	case bool:
		if val {
			return "True"
		}
		return "False"
	case float64:
		if val == math.Trunc(val) && !math.IsInf(val, 0) && !math.IsNaN(val) {
			return strconv.FormatFloat(val, 'f', 1, 64)
		}
		return strconv.FormatFloat(val, 'f', -1, 64)
	case int:
		return strconv.Itoa(val)
	case time.Time:
		return formatPythonDatetime(val)
	case []interface{}:
		return formatPythonList(val)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// formatPythonDatetime formats a time.Time to match Python's str(datetime) output.
func formatPythonDatetime(t time.Time) string {
	_, offset := t.Zone()
	sign := "+"
	if offset < 0 {
		sign = "-"
		offset = -offset
	}
	hours := offset / 3600
	minutes := (offset % 3600) / 60
	return fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d%s%02d:%02d",
		t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second(),
		sign, hours, minutes)
}

// formatPythonList formats a []interface{} to match Python's str(list) output,
// e.g. ['foo', 'bar'] for strings, [1, 2] for numbers.
func formatPythonList(items []interface{}) string {
	parts := make([]string, len(items))
	for i, item := range items {
		switch v := item.(type) {
		case string:
			parts[i] = fmt.Sprintf("'%s'", v)
		default:
			parts[i] = formatYAMLValue(v)
		}
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func buildAppliesToString(version map[string]string, release string) string {
	parts := getAppliesTo(version, release, "", true)
	return strings.Join(parts, ", ")
}
