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

package actions

import (
	"fmt"
	"io"
	"os"
	"regexp"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/formats/elf"
	"github.com/elastic/beats/v7/libbeat/formats/lnk"
	"github.com/elastic/beats/v7/libbeat/formats/macho"
	"github.com/elastic/beats/v7/libbeat/formats/pe"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/mime"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/processors/checks"
)

const (
	addFileDataName      = "add_file_data"
	addFileDataLogName   = "processor." + addFileDataName
	defaultFilePathField = "file.path"
	defaultTargetField   = "file"
)

func init() {
	processors.RegisterPlugin(addFileDataName,
		checks.ConfigChecked(NewAddFileData,
			checks.AllowedFields("field", "target", "exclude", "only", "pattern", "ignore_failure")))
}

type addFileDataProcessor struct {
	Field         string    `config:"field"`
	Target        string    `config:"target"`
	Exclude       *[]string `config:"exclude"`
	Only          *[]string `config:"only"`
	Pattern       string    `config:"pattern"`
	IgnoreFailure bool      `config:"ignore_failure"`

	parsers  []*parser
	compiled *regexp.Regexp
	log      *logp.Logger
}

// NewAddFileData constructs a add format data processor.
func NewAddFileData(cfg *common.Config) (processors.Processor, error) {
	cfgwarn.Beta("The " + addFileDataName + " processor is beta.")
	log := logp.NewLogger(addFileDataLogName)
	addFileData := &addFileDataProcessor{
		Field:  defaultFilePathField,
		Target: defaultTargetField,
		log:    log,
	}
	if err := cfg.Unpack(addFileData); err != nil {
		return nil, errors.Wrapf(err, "fail to unpack the "+addFileDataName+" configuration")
	}
	if addFileData.Pattern != "" {
		compiled, err := regexp.Compile(addFileData.Pattern)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("invalid pattern for "+addFileDataName+": '%s'", addFileData.Pattern))
		}
		addFileData.compiled = compiled
	}
	parsers := allParsers
	// only takes precedence to exclude
	if addFileData.Only != nil {
		parsers = onlyParsers(*addFileData.Only)
	}
	if addFileData.Exclude != nil {
		parsers = filterParsers(*addFileData.Exclude)
	}
	addFileData.parsers = parsers

	return addFileData, nil
}

func (a *addFileDataProcessor) applyParser(event *beat.Event, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	mimeType := mime.DetectReader(file)
	if mimeType == "" {
		// we couldn't identify the file, don't parse it
		return nil
	}
	for _, parser := range a.parsers {
		if mimeType == parser.mimeType {
			data, err := parser.parse(file)
			if err != nil {
				return err
			}
			target := a.Target + "." + parser.target
			event.Fields.DeepUpdate(common.MapStr{
				target: data,
			})
			return nil
		}
	}
	return nil
}

func (a *addFileDataProcessor) Run(event *beat.Event) (*beat.Event, error) {
	valI, err := event.GetValue(a.Field)
	if err != nil {
		// doesn't have the required fieldd value to analyze
		return event, nil
	}
	val, _ := valI.(string)
	if val == "" {
		// wrong type or not set
		return event, nil
	}
	if a.compiled != nil {
		if !a.compiled.MatchString(val) {
			// we filtered out this event
			return event, nil
		}
	}
	if err := a.applyParser(event, val); err != nil {
		if a.IgnoreFailure {
			a.log.Debugf("failed to parse file because of error: %v", err)
			return event, nil
		}
		return event, err
	}
	return event, nil
}

func (a *addFileDataProcessor) String() string {
	return fmt.Sprintf("%s=%+v,%+v,%+v", addFileDataName, a.Field, a.Exclude, a.Only)
}

type parser struct {
	name     string
	target   string
	mimeType string
	parse    func(r io.ReaderAt) (interface{}, error)
}

var allParsers = []*parser{
	makeParser("pe", "pe", "application/vnd.microsoft.portable-executable", pe.Parse),
	makeParser("macho", "macho", "application/x-mach-binary", macho.Parse),
	makeParser("elf", "elf", "application/x-executable", elf.Parse),
	makeParser("lnk", "lnk", "application/x-ms-shortcut", lnk.Parse),
}

func makeParser(name, target, mimeType string, parse func(r io.ReaderAt) (interface{}, error)) *parser {
	return &parser{
		name:     name,
		target:   target,
		mimeType: mimeType,
		parse:    parse,
	}
}

func filterParsers(exclude []string) []*parser {
	parsers := []*parser{}
	exclusionSet := map[string]struct{}{}
	for _, exclusion := range exclude {
		exclusionSet[exclusion] = struct{}{}
	}

	for _, parser := range allParsers {
		if _, ok := exclusionSet[parser.name]; ok {
			continue
		}
		parsers = append(parsers, parser)
	}
	return parsers
}

func onlyParsers(only []string) []*parser {
	parsers := []*parser{}
	inclusionSet := map[string]struct{}{}
	for _, inclusion := range only {
		inclusionSet[inclusion] = struct{}{}
	}

	for _, parser := range allParsers {
		if _, ok := inclusionSet[parser.name]; ok {
			parsers = append(parsers, parser)
		}
	}
	return parsers
}
