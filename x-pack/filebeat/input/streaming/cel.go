// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package streaming

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/url"
	"path"
	"reflect"
	"regexp"
	"runtime"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"

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
		cel.Lib(urlLib{}),
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

// urlLib provides URL and query parsing and formatting functions consistent
// with the equivalent functions in the mito/lib.HTTP library.
//   - parse_url
//   - format_url
//   - parse_query
//   - format_query
type urlLib struct{}

var (
	// Type used in overloads.
	mapStringDyn = cel.MapType(cel.StringType, cel.DynType)

	// Type used for reflect conversion.
	reflectMapStringAnyType         = reflect.TypeFor[map[string]any]()
	reflectMapStringStringSliceType = reflect.TypeFor[map[string][]string]()
)

func (urlLib) CompileOptions() []cel.EnvOption {
	return []cel.EnvOption{
		cel.Function("parse_url",
			cel.MemberOverload(
				"string_parse_url",
				[]*cel.Type{cel.StringType},
				mapStringDyn,
				cel.UnaryBinding(catch(parseURL)),
			),
		),
		cel.Function("format_url",
			cel.MemberOverload(
				"map_format_url",
				[]*cel.Type{mapStringDyn},
				cel.StringType,
				cel.UnaryBinding(catch(formatURL)),
			),
		),

		cel.Function("parse_query",
			cel.MemberOverload(
				"string_parse_query",
				[]*cel.Type{cel.StringType},
				mapStringDyn,
				cel.UnaryBinding(catch(parseQuery)),
			),
		),
		cel.Function("format_query",
			cel.MemberOverload(
				"map_format_query",
				[]*cel.Type{mapStringDyn},
				cel.StringType,
				cel.UnaryBinding(catch(formatQuery)),
			),
		),
	}
}

type (
	unop  = func(value ref.Val) ref.Val
	binop = func(lhs ref.Val, rhs ref.Val) ref.Val
	varop = func(values ...ref.Val) ref.Val

	bindings interface {
		unop | binop | varop
	}
)

func catch[B bindings](binding B) B {
	switch binding := any(binding).(type) {
	case unop:
		return any(func(arg ref.Val) (ret ref.Val) {
			defer handlePanic(&ret)
			return binding(arg)
		}).(B)
	case binop:
		return any(func(arg0, arg1 ref.Val) (ret ref.Val) {
			defer handlePanic(&ret)
			return binding(arg0, arg1)
		}).(B)
	case varop:
		return any(func(args ...ref.Val) (ret ref.Val) {
			defer handlePanic(&ret)
			return binding(args...)
		}).(B)
	default:
		panic("unreachable")
	}
}

func handlePanic(ret *ref.Val) {
	switch r := recover().(type) {
	case nil:
		return
	default:
		// We'll only try 64 stack frames deep. There are a no recursive
		// functions in extensions.
		pc := make([]uintptr, 64)
		n := runtime.Callers(2, pc)
		cf := runtime.CallersFrames(pc[:n])
		for {
			f, more := cf.Next()
			if !more {
				break
			}
			file := f.File
			if strings.Contains(file, "filebeat/input/streaming") {
				_, file, _ := strings.Cut(file, "filebeat/input/")
				*ret = types.NewErr("%s: %s %s:%d", r, path.Base(f.Function), file, f.Line)
				return
			}
		}
		*ret = types.NewErr("%s", r)
	}
}

func (urlLib) ProgramOptions() []cel.ProgramOption { return nil }

func parseURL(arg ref.Val) ref.Val {
	addr, ok := arg.(types.String)
	if !ok {
		return types.ValOrErr(addr, "no such overload for request")
	}
	u, err := url.Parse(string(addr))
	if err != nil {
		return types.NewErr("%s", err)
	}
	var user interface{}
	if u.User != nil {
		password, passwordSet := u.User.Password()
		user = map[string]interface{}{
			"Username":    u.User.Username(),
			"Password":    password,
			"PasswordSet": passwordSet,
		}
	}
	return types.NewStringInterfaceMap(types.DefaultTypeAdapter, map[string]interface{}{
		"Scheme":      u.Scheme,
		"Opaque":      u.Opaque,
		"User":        user,
		"Host":        u.Host,
		"Path":        u.Path,
		"RawPath":     u.RawPath,
		"ForceQuery":  u.ForceQuery,
		"RawQuery":    u.RawQuery,
		"Fragment":    u.Fragment,
		"RawFragment": u.RawFragment,
	})
}

func formatURL(arg ref.Val) ref.Val {
	urlMap, ok := arg.(traits.Mapper)
	if !ok {
		return types.ValOrErr(urlMap, "no such overload")
	}
	v, err := urlMap.ConvertToNative(reflectMapStringAnyType)
	if err != nil {
		return types.NewErr("no such overload for format_url: %v", err)
	}
	m, ok := v.(map[string]interface{})
	if !ok {
		// This should never happen.
		return types.NewErr("unexpected type for url map: %T", v)
	}
	u := url.URL{
		Scheme:      maybeStringLookup(m, "Scheme"),
		Opaque:      maybeStringLookup(m, "Opaque"),
		Host:        maybeStringLookup(m, "Host"),
		Path:        maybeStringLookup(m, "Path"),
		RawPath:     maybeStringLookup(m, "RawPath"),
		ForceQuery:  maybeBoolLookup(m, "ForceQuery"),
		RawQuery:    maybeStringLookup(m, "RawQuery"),
		Fragment:    maybeStringLookup(m, "Fragment"),
		RawFragment: maybeStringLookup(m, "RawFragment"),
	}
	user, ok := urlMap.Find(types.String("User"))
	if ok {
		switch user := user.(type) {
		case nil:
		case traits.Mapper:
			var username types.String
			un, ok := user.Find(types.String("Username"))
			if ok {
				username, ok = un.(types.String)
				if !ok {
					return types.NewErr("invalid type for username: %s", un.Type())
				}
			}
			if user.Get(types.String("PasswordSet")) == types.True {
				var password types.String
				pw, ok := user.Find(types.String("Password"))
				if ok {
					password, ok = pw.(types.String)
					if !ok {
						return types.NewErr("invalid type for password: %s", pw.Type())
					}
				}
				u.User = url.UserPassword(string(username), string(password))
			} else {
				u.User = url.User(string(username))
			}
		default:
			if user != types.NullValue {
				return types.NewErr("unsupported type: %T", user)
			}
		}
	}
	return types.String(u.String())
}

// maybeStringLookup returns a string from m[key] if it is present and the
// empty string if not. It panics is m[key] is not a string.
func maybeStringLookup(m map[string]interface{}, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	return v.(string)
}

// maybeBoolLookup returns a bool from m[key] if it is present and false if
// not. It panics is m[key] is not a bool.
func maybeBoolLookup(m map[string]interface{}, key string) bool {
	v, ok := m[key]
	if !ok {
		return false
	}
	return v.(bool)
}

func parseQuery(arg ref.Val) ref.Val {
	query, ok := arg.(types.String)
	if !ok {
		return types.ValOrErr(query, "no such overload")
	}
	q, err := url.ParseQuery(string(query))
	if err != nil {
		return types.NewErr("%s", err)
	}
	return types.DefaultTypeAdapter.NativeToValue(q)
}

func formatQuery(arg ref.Val) ref.Val {
	queryMap, ok := arg.(traits.Mapper)
	if !ok {
		return types.ValOrErr(queryMap, "no such overload")
	}
	q, err := queryMap.ConvertToNative(reflectMapStringStringSliceType)
	if err != nil {
		return types.NewErr("no such overload for format_query: %v", err)
	}
	switch q := q.(type) {
	case url.Values:
		return types.String(q.Encode())
	case map[string][]string:
		return types.String(url.Values(q).Encode())
	default:
		return types.NewErr("invalid type for format_query: %T", q)
	}
}
