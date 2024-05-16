// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package websocket

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"regexp"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"

	"github.com/elastic/beats/v7/libbeat/version"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/useragent"
	"github.com/elastic/mito/lib"
)

var (
	// mimetypes holds supported MIME type mappings.
	mimetypes = map[string]interface{}{
		"application/gzip":         func(r io.Reader) (io.Reader, error) { return gzip.NewReader(r) },
		"application/x-ndjson":     lib.NDJSON,
		"application/zip":          lib.Zip,
		"text/csv; header=absent":  lib.CSVNoHeader,
		"text/csv; header=present": lib.CSVHeader,
		"text/csv;header=absent":   lib.CSVNoHeader,
		"text/csv;header=present":  lib.CSVHeader,
	}
)

func regexpsFromConfig(cfg config) (map[string]*regexp.Regexp, error) {
	if len(cfg.Regexps) == 0 {
		return nil, nil
	}
	patterns := make(map[string]*regexp.Regexp)
	for name, expr := range cfg.Regexps {
		var err error
		patterns[name], err = regexp.Compile(expr)
		if err != nil {
			return nil, err
		}
	}
	return patterns, nil
}

// The Filebeat user-agent is provided to the program as useragent.
var userAgent = useragent.UserAgent("Filebeat", version.GetDefaultVersion(), version.Commit(), version.BuildTime().String())

func newProgram(ctx context.Context, src, root string, patterns map[string]*regexp.Regexp, log *logp.Logger) (cel.Program, *cel.Ast, error) {
	opts := []cel.EnvOption{
		cel.Declarations(decls.NewVar(root, decls.Dyn)),
		cel.OptionalTypes(cel.OptionalTypesVersion(lib.OptionalTypesVersion)),
		lib.Collections(),
		lib.Crypto(),
		lib.JSON(nil),
		lib.Strings(),
		lib.Time(),
		lib.Try(),
		lib.Debug(debug(log)),
		lib.MIME(mimetypes),
		lib.Globals(map[string]interface{}{
			"useragent": userAgent,
		}),
	}
	if len(patterns) != 0 {
		opts = append(opts, lib.Regexp(patterns))
	}

	env, err := cel.NewEnv(opts...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create env: %w", err)
	}

	ast, iss := env.Compile(src)
	if iss.Err() != nil {
		return nil, nil, fmt.Errorf("failed compilation: %w", iss.Err())
	}

	prg, err := env.Program(ast)
	if err != nil {
		return nil, nil, fmt.Errorf("failed program instantiation: %w", err)
	}
	return prg, ast, nil
}

func debug(log *logp.Logger) func(string, any) {
	log = log.Named("websocket_debug")
	return func(tag string, value any) {
		level := "DEBUG"
		if _, ok := value.(error); ok {
			level = "ERROR"
		}

		log.Debugw(level, "tag", tag, "value", value)
	}
}
