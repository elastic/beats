// Copyright 2020 The OPA Authors.  All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

// Package compile implements bundles compilation and linking.
package compile

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"

	"github.com/pkg/errors"

	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/bundle"
	"github.com/open-policy-agent/opa/internal/compiler/wasm"
	"github.com/open-policy-agent/opa/internal/debug"
	"github.com/open-policy-agent/opa/internal/ir"
	"github.com/open-policy-agent/opa/internal/planner"
	"github.com/open-policy-agent/opa/internal/ref"
	initload "github.com/open-policy-agent/opa/internal/runtime/init"
	"github.com/open-policy-agent/opa/internal/wasm/encoding"
	"github.com/open-policy-agent/opa/loader"
	"github.com/open-policy-agent/opa/rego"
	"github.com/open-policy-agent/opa/storage"
	"github.com/open-policy-agent/opa/storage/inmem"
)

const (
	// TargetRego is the default target. The source rego is copied (potentially
	// rewritten for optimization purpsoes) into the bundle. The target supports
	// base documents.
	TargetRego = "rego"

	// TargetWasm is an alternative target that compiles the policy into a wasm
	// module instead of Rego. The target supports base documents.
	TargetWasm = "wasm"
)

const wasmResultVar = ast.Var("result")

var validTargets = map[string]struct{}{
	TargetRego: {},
	TargetWasm: {},
}

// Compiler implements bundle compilation and linking.
type Compiler struct {
	capabilities      *ast.Capabilities          // the capabilities that compiled policies may require
	bundle            *bundle.Bundle             // the bundle that the compiler operates on
	revision          *string                    // the revision to set on the output bundle
	asBundle          bool                       // whether to assume bundle layout on file loading or not
	filter            loader.Filter              // filter to apply to file loader
	paths             []string                   // file paths to load. TODO(tsandall): add support for supplying readers for embedded users.
	entrypoints       orderedStringSet           // policy entrypoints required for optimization and certain targets
	optimizationLevel int                        // how aggressive should optimization be
	target            string                     // target type (wasm, rego, etc.)
	output            *io.Writer                 // output stream to write bundle to
	entrypointrefs    []*ast.Term                // validated entrypoints computed from default decision or manually supplied entrypoints
	compiler          *ast.Compiler              // rego ast compiler used for semantic checks and rewriting
	debug             debug.Debug                // optionally outputs debug information produced during build
	bvc               *bundle.VerificationConfig // represents the key configuration used to verify a signed bundle
	bsc               *bundle.SigningConfig      // represents the key configuration used to generate a signed bundle
	keyID             string                     // represents the name of the default key used to verify a signed bundle
}

// New returns a new compiler instance that can be invoked.
func New() *Compiler {
	return &Compiler{
		asBundle:          false,
		optimizationLevel: 0,
		target:            TargetRego,
		debug:             debug.Discard(),
	}
}

// WithRevision sets the revision to include in the output bundle manifest.
func (c *Compiler) WithRevision(r string) *Compiler {
	c.revision = &r
	return c
}

// WithAsBundle sets file loading mode on the compiler.
func (c *Compiler) WithAsBundle(enabled bool) *Compiler {
	c.asBundle = enabled
	return c
}

// WithEntrypoints sets the policy entrypoints on the compiler. Entrypoints tell the
// compiler what rules to expect and where optimizations can be targeted. The wasm
// target requires at least one entrypoint as does optimization.
func (c *Compiler) WithEntrypoints(e ...string) *Compiler {
	c.entrypoints = c.entrypoints.Append(e...)
	return c
}

// WithOptimizationLevel sets the optimization level on the compiler. By default
// optimizations are disabled. Higher levels apply more aggressive optimizations
// but can take longer.
func (c *Compiler) WithOptimizationLevel(n int) *Compiler {
	c.optimizationLevel = n
	return c
}

// WithTarget sets the output target type to use.
func (c *Compiler) WithTarget(t string) *Compiler {
	c.target = t
	return c
}

// WithOutput sets the output stream to write the bundle to.
func (c *Compiler) WithOutput(w io.Writer) *Compiler {
	c.output = &w
	return c
}

// WithDebug sets the output stream to write debug info to.
func (c *Compiler) WithDebug(sink io.Writer) *Compiler {
	if sink != nil {
		c.debug = debug.New(sink)
	}
	return c
}

// WithPaths adds input filepaths to read policy and data from.
func (c *Compiler) WithPaths(p ...string) *Compiler {
	c.paths = append(c.paths, p...)
	return c
}

// WithFilter sets the loader filter to use when reading non-bundle input files.
func (c *Compiler) WithFilter(filter loader.Filter) *Compiler {
	c.filter = filter
	return c
}

// WithBundle sets the input bundle to compile. This should be used as an
// alternative to reading from paths. This function overrides any file
// loading options.
func (c *Compiler) WithBundle(b *bundle.Bundle) *Compiler {
	c.bundle = b
	return c
}

// WithBundleVerificationConfig sets the key configuration to use to verify a signed bundle
func (c *Compiler) WithBundleVerificationConfig(config *bundle.VerificationConfig) *Compiler {
	c.bvc = config
	return c
}

// WithBundleSigningConfig sets the key configuration to use to generate a signed bundle
func (c *Compiler) WithBundleSigningConfig(config *bundle.SigningConfig) *Compiler {
	c.bsc = config
	return c
}

// WithBundleVerificationKeyID sets the key to use to verify a signed bundle.
// If provided, the "keyid" claim in the bundle signature, will be set to this value
func (c *Compiler) WithBundleVerificationKeyID(keyID string) *Compiler {
	c.keyID = keyID
	return c
}

// WithCapabilities sets the capabilities to use while checking policies.
func (c *Compiler) WithCapabilities(capabilities *ast.Capabilities) *Compiler {
	c.capabilities = capabilities
	return c
}

// Build compiles and links the input files and outputs a bundle to the writer.
func (c *Compiler) Build(ctx context.Context) error {

	if err := c.init(); err != nil {
		return err
	}

	if err := c.initBundle(); err != nil {
		return err
	}

	if err := c.optimize(ctx); err != nil {
		return err
	}

	if c.target == TargetWasm {
		if err := c.compileWasm(ctx); err != nil {
			return err
		}
	}

	if c.revision != nil {
		c.bundle.Manifest.Revision = *c.revision
	}

	if err := c.bundle.FormatModules(false); err != nil {
		return err
	}

	if c.bsc != nil {
		if err := c.bundle.GenerateSignature(c.bsc, c.keyID, false); err != nil {
			return err
		}
	}

	if c.output == nil {
		return nil
	}

	return bundle.NewWriter(*c.output).Write(*c.bundle)
}

func (c *Compiler) init() error {

	if c.capabilities == nil {
		c.capabilities = ast.CapabilitiesForThisVersion()
	}

	if _, ok := validTargets[c.target]; !ok {
		return fmt.Errorf("invalid target %q", c.target)
	}

	for _, e := range c.entrypoints {

		r, err := ref.ParseDataPath(e)
		if err != nil {
			return fmt.Errorf("entrypoint %v not valid: use <package>/<rule>", e)
		}

		c.entrypointrefs = append(c.entrypointrefs, ast.NewTerm(r))
	}

	if c.optimizationLevel > 0 && len(c.entrypointrefs) == 0 {
		return errors.New("bundle optimizations require at least one entrypoint")
	}

	if c.target == TargetWasm && len(c.entrypointrefs) == 0 {
		return errors.New("wasm compilation requires at least one entrypoint")
	}

	return nil
}

// Bundle returns the compiled bundle. This function can be called to retrieve the
// output of the compiler (as an alternative to having the bundle written to a stream.)
func (c *Compiler) Bundle() *bundle.Bundle {
	return c.bundle
}

func (c *Compiler) initBundle() error {

	// If the bundle is already set, skip file loading.
	if c.bundle != nil {
		return nil
	}

	// TODO(tsandall): the metrics object should passed through here so we that
	// we can track read and parse times.

	load, err := initload.LoadPaths(c.paths, c.filter, c.asBundle, c.bvc, false)
	if err != nil {
		return errors.Wrap(err, "load error")
	}

	if c.asBundle {
		var names []string

		for k := range load.Bundles {
			names = append(names, k)
		}

		sort.Strings(names)
		var bundles []*bundle.Bundle

		for _, k := range names {
			bundles = append(bundles, load.Bundles[k])
		}

		result, err := bundle.Merge(bundles)
		if err != nil {
			return fmt.Errorf("bundle merge failed: %v", err)
		}

		c.bundle = result
		return nil
	}

	// TODO(tsandall): add support for controlling roots. Either the caller could
	// supply them or the compiler could infer them based on the packages and data
	// contents. The latter would require changes to the loader to preserve the
	// locations where base documents were mounted under data.
	result := &bundle.Bundle{}
	result.Manifest.Init()
	result.Data = load.Files.Documents

	var modules []string

	for k := range load.Files.Modules {
		modules = append(modules, k)
	}

	sort.Strings(modules)

	for _, module := range modules {
		result.Modules = append(result.Modules, bundle.ModuleFile{
			URL:    load.Files.Modules[module].Name,
			Path:   load.Files.Modules[module].Name,
			Parsed: load.Files.Modules[module].Parsed,
			Raw:    load.Files.Modules[module].Raw,
		})
	}

	c.bundle = result

	return nil
}

func (c *Compiler) optimize(ctx context.Context) error {

	if c.optimizationLevel <= 0 {
		var err error
		c.compiler, err = compile(c.capabilities, c.bundle, c.debug)
		return err
	}

	o := newOptimizer(c.capabilities, c.bundle).
		WithEntrypoints(c.entrypointrefs).
		WithDebug(c.debug.Writer()).
		WithShallowInlining(c.optimizationLevel <= 1)

	err := o.Do(ctx)
	if err != nil {
		return err
	}

	c.bundle = o.Bundle()

	return nil
}

func (c *Compiler) compileWasm(ctx context.Context) error {

	// Lazily compile the modules if needed. If optimizations were run, the
	// AST compiler will not be set because the default target does not require it.
	if c.compiler == nil {
		var err error
		c.compiler, err = compile(c.capabilities, c.bundle, c.debug)
		if err != nil {
			return err
		}
	}

	// Find transitive dependents of entrypoints and add them to the set to compile.
	//
	// NOTE(tsandall): We compile entrypoints because the evaluator does not support
	// evaluation of wasm-compiled rules when 'with' statements are in-scope. Compiling
	// out the dependents avoids the need to support that case for now.
	deps := map[*ast.Rule]struct{}{}
	for i := range c.entrypointrefs {
		transitiveDocumentDependents(c.compiler, c.entrypointrefs[i], deps)
	}

	extras := ast.NewSet()
	for rule := range deps {
		extras.Add(ast.NewTerm(rule.Path()))
	}

	sorted := extras.Sorted()

	for i := 0; i < sorted.Len(); i++ {
		p, err := sorted.Elem(i).Value.(ast.Ref).Ptr()
		if err != nil {
			return err
		}

		if !c.entrypoints.Contains(p) {
			c.entrypoints = append(c.entrypoints, p)
			c.entrypointrefs = append(c.entrypointrefs, sorted.Elem(i))
		}
	}

	// Create query sets for each of the entrypoints.
	resultSym := ast.NewTerm(wasmResultVar)
	queries := make([]planner.QuerySet, len(c.entrypointrefs))

	for i := range c.entrypointrefs {

		qc := c.compiler.QueryCompiler()
		query := ast.NewBody(ast.Equality.Expr(resultSym, c.entrypointrefs[i]))
		compiled, err := qc.Compile(query)
		if err != nil {
			return err
		}

		queries[i] = planner.QuerySet{
			Name:          c.entrypoints[i],
			Queries:       []ast.Body{compiled},
			RewrittenVars: qc.RewrittenVars(),
		}
	}

	// Prepare modules and builtins for the planner.
	modules := []*ast.Module{}
	for _, module := range c.compiler.Modules {
		modules = append(modules, module)
	}

	builtins := make(map[string]*ast.Builtin, len(c.capabilities.Builtins))
	for _, bi := range c.capabilities.Builtins {
		builtins[bi.Name] = bi
	}

	compiler := wasm.New()
	found := false
	have := compiler.ABIVersion()
	if c.capabilities.WasmABIVersions == nil { // discern nil from len=0
		c.debug.Printf("no wasm ABI versions in capabilities, building for %v", have)
		found = true
	}
	for _, v := range c.capabilities.WasmABIVersions {
		if v.Version == have.Version && v.Minor <= have.Minor {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("compiler ABI version not in capabilities (have %v, want %d)",
			c.capabilities.WasmABIVersions,
			compiler.ABIVersion(),
		)
	}

	// Plan the query sets.
	p := planner.New().
		WithQueries(queries).
		WithModules(modules).
		WithBuiltinDecls(builtins).
		WithDebug(c.debug.Writer())
	policy, err := p.Plan()
	if err != nil {
		return err
	}

	// dump policy IR (if "debug" wasn't requested, debug.Witer will discard it)
	err = ir.Pretty(c.debug.Writer(), policy)
	if err != nil {
		return err
	}

	// Compile the policy into a wasm binary.
	m, err := compiler.WithPolicy(policy).WithDebug(c.debug.Writer()).Compile()
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := encoding.WriteModule(&buf, m); err != nil {
		return err
	}

	modulePath := bundle.WasmFile

	c.bundle.WasmModules = []bundle.WasmModuleFile{{
		URL:  modulePath,
		Path: modulePath,
		Raw:  buf.Bytes(),
	}}

	// Each entrypoint needs an entry in the manifest
	for i := range c.entrypointrefs {
		entrypointPath := c.entrypoints[i]

		c.bundle.Manifest.WasmResolvers = append(c.bundle.Manifest.WasmResolvers, bundle.WasmResolver{
			Module:     "/" + strings.TrimLeft(modulePath, "/"),
			Entrypoint: entrypointPath,
		})
	}

	// Remove the entrypoints from remaining source rego files
	return pruneBundleEntrypoints(c.bundle, c.entrypointrefs)
}

// pruneBundleEntrypoints will modify modules in the provided bundle to remove
// rules matching the entrypoints along with injecting import statements to
// preserve their ability to compile.
func pruneBundleEntrypoints(b *bundle.Bundle, entrypointrefs []*ast.Term) error {

	// For each package path keep a list of new imports to add.
	requiredImports := map[string][]*ast.Import{}

	for _, entrypoint := range entrypointrefs {
		for i := 0; i < len(b.Modules); i++ {
			mf := &b.Modules[i]

			// Drop any rules that match the entrypoint path.
			var rules []*ast.Rule
			for _, rule := range mf.Parsed.Rules {
				rulePath := rule.Path()
				if !rulePath.Equal(entrypoint.Value) {
					rules = append(rules, rule)
				} else {
					pkgPath := rule.Module.Package.Path.String()
					newImport := &ast.Import{Path: ast.NewTerm(rulePath)}
					shouldAdd := true
					currentImports := requiredImports[pkgPath]
					for _, imp := range currentImports {
						if imp.Equal(newImport) {
							shouldAdd = false
							break
						}
					}
					if shouldAdd {
						requiredImports[pkgPath] = append(currentImports, newImport)
					}
				}
			}

			// If any rules were dropped update the module accordingly
			if len(rules) != len(mf.Parsed.Rules) {
				mf.Parsed.Rules = rules
				// Remove the original raw source, we're editing the AST
				// directly so it wont be in sync anymore.
				mf.Raw = nil
			}
		}
	}

	// Any packages which had rules removed need an import injected for the
	// removed rule to keep the policies valid.
	for i := 0; i < len(b.Modules); i++ {
		mf := &b.Modules[i]
		pkgPath := mf.Parsed.Package.Path.String()
		if imports, ok := requiredImports[pkgPath]; ok {
			mf.Raw = nil
			mf.Parsed.Imports = append(mf.Parsed.Imports, imports...)
		}
	}

	return nil
}

type undefinedEntrypointErr struct {
	Entrypoint *ast.Term
}

func (err undefinedEntrypointErr) Error() string {
	return fmt.Sprintf("undefined entrypoint %v", err.Entrypoint)
}

type optimizer struct {
	capabilities    *ast.Capabilities
	bundle          *bundle.Bundle
	compiler        *ast.Compiler
	entrypoints     []*ast.Term
	nsprefix        string
	resultsymprefix string
	outputprefix    string
	shallow         bool
	debug           debug.Debug
}

func newOptimizer(c *ast.Capabilities, b *bundle.Bundle) *optimizer {
	return &optimizer{
		capabilities:    c,
		bundle:          b,
		nsprefix:        "partial",
		resultsymprefix: ast.WildcardPrefix,
		outputprefix:    "optimized",
		debug:           debug.Discard(),
	}
}

func (o *optimizer) WithDebug(sink io.Writer) *optimizer {
	if sink != nil {
		o.debug = debug.New(sink)
	}
	return o
}

func (o *optimizer) WithEntrypoints(es []*ast.Term) *optimizer {
	o.entrypoints = es
	return o
}

func (o *optimizer) WithShallowInlining(yes bool) *optimizer {
	o.shallow = yes
	return o
}

func (o *optimizer) Do(ctx context.Context) error {

	// NOTE(tsandall): if there are multiple entrypoints, copy the bundle because
	// if any of the optimization steps fail, we do not want to leave the caller's
	// bundle in a partially modified state.
	if len(o.entrypoints) > 1 {
		cpy := o.bundle.Copy()
		o.bundle = &cpy
	}

	// initialize other inputs to the optimization process (store, symbols, etc.)
	data := o.bundle.Data
	if data == nil {
		data = map[string]interface{}{}
	}

	store := inmem.NewFromObject(data)
	resultsym := ast.VarTerm(o.resultsymprefix + "__result__")
	usedFilenames := map[string]int{}
	var unknowns []*ast.Term

	// NOTE(tsandall): the entrypoints are optimized in order so that the optimization
	// of entrypoint[1] sees the optimization of entrypoint[0] and so on. This is needed
	// because otherwise the optimization outputs (e.g., support rules) would have to
	// merged somehow. Instead of dealing with that, just run the optimizations in the
	// order the user supplied the entrypoints in.
	for i, e := range o.entrypoints {

		var err error
		o.compiler, err = compile(o.capabilities, o.bundle, o.debug)
		if err != nil {
			return err
		}

		if unknowns == nil {
			unknowns = o.findUnknowns()
		}

		required := o.findRequiredDocuments(e)

		r := rego.New(
			rego.ParsedQuery(ast.NewBody(ast.Equality.Expr(resultsym, e))),
			rego.PartialNamespace(o.nsprefix),
			rego.DisableInlining(required),
			rego.ShallowInlining(o.shallow),
			rego.SkipPartialNamespace(true),
			rego.ParsedUnknowns(unknowns),
			rego.Compiler(o.compiler),
			rego.Store(store),
		)

		o.debug.Printf("optimizer: entrypoint: %v", e)
		o.debug.Printf("  partial-namespace: %v", o.nsprefix)
		o.debug.Printf("  disable-inlining: %v", required)
		o.debug.Printf("  shallow-inlining: %v", o.shallow)

		for i := range unknowns {
			o.debug.Printf("  unknown: %v", unknowns[i])
		}

		pq, err := r.Partial(ctx)
		if err != nil {
			return err
		}

		// NOTE(tsandall): this might be a bit too strict but in practice it's
		// unlikely users will want to ignore undefined entrypoints. make this
		// optional in the future.
		if len(pq.Queries) == 0 {
			return undefinedEntrypointErr{Entrypoint: e}
		}

		if module := o.getSupportForEntrypoint(pq.Queries, e, resultsym); module != nil {
			pq.Support = append(pq.Support, module)
		}

		modules := make([]bundle.ModuleFile, len(pq.Support))

		for j := range pq.Support {
			fileName := o.getSupportModuleFilename(usedFilenames, pq.Support[j], i, j)
			modules[j] = bundle.ModuleFile{
				URL:    fileName,
				Path:   fileName,
				Parsed: pq.Support[j],
			}
		}

		o.bundle.Modules = o.merge(o.bundle.Modules, modules)
	}

	sort.Slice(o.bundle.Modules, func(i, j int) bool {
		return o.bundle.Modules[i].URL < o.bundle.Modules[j].URL
	})

	// NOTE(tsandall): prune out rules and data that are not referenced in the bundle
	// in the future.
	o.bundle.Manifest.AddRoot(o.nsprefix)
	o.bundle.Manifest.Revision = ""

	return nil
}

func (o *optimizer) Bundle() *bundle.Bundle {
	return o.bundle
}

func (o *optimizer) findRequiredDocuments(ref *ast.Term) []string {

	keep := map[string]*ast.Location{}
	deps := map[*ast.Rule]struct{}{}

	transitiveDocumentDependents(o.compiler, ref, deps)

	for rule := range deps {
		ast.WalkExprs(rule, func(expr *ast.Expr) bool {
			for _, with := range expr.With {
				// TODO(tsandall): this should be improved to exclude refs that are
				// marked as unknown. Since the build command does not allow users to
				// set unknowns, we can hardcode to assume 'input'.
				if !with.Target.Value.(ast.Ref).HasPrefix(ast.InputRootRef) {
					keep[with.Target.String()] = with.Target.Location
				}
			}
			return false
		})
	}

	var result []string

	for k := range keep {
		result = append(result, k)
	}

	sort.Strings(result)

	for _, k := range result {
		o.debug.Printf("%s: disables inlining of %v", keep[k], k)
	}

	return result
}

func (o *optimizer) findUnknowns() []*ast.Term {

	// Initialize set of refs representing the bundle roots.
	refs := newRefSet(stringsToRefs(*o.bundle.Manifest.Roots)...)

	// Initialize set of refs for the result (i.e., refs outside the bundle roots.)
	unknowns := newRefSet(ast.InputRootRef)

	// Find data references that are not prefixed by one of the roots.
	for _, module := range o.compiler.Modules {
		ast.WalkRefs(module, func(x ast.Ref) bool {
			prefix := x.ConstantPrefix()
			if !prefix.HasPrefix(ast.DefaultRootRef) {
				return true
			}
			if !refs.ContainsPrefix(prefix) {
				unknowns.AddPrefix(prefix)
			}
			return false
		})
	}

	return unknowns.Sorted()
}

func (o *optimizer) getSupportForEntrypoint(queries []ast.Body, e *ast.Term, resultsym *ast.Term) *ast.Module {

	path := e.Value.(ast.Ref)
	name := ast.Var(path[len(path)-1].Value.(ast.String))
	module := &ast.Module{Package: &ast.Package{Path: path[:len(path)-1]}}

	for _, query := range queries {
		// NOTE(tsandall): when the query refers to the original entrypoint, throw it
		// away since this would create a recursive rule--this occurs if the entrypoint
		// cannot be partially evaluated.
		stop := false
		ast.WalkRefs(query, func(x ast.Ref) bool {
			if !stop {
				if x.HasPrefix(path) {
					stop = true
				}
			}
			return stop
		})
		if stop {
			return nil
		}
		module.Rules = append(module.Rules, &ast.Rule{
			Head:   ast.NewHead(name, nil, resultsym),
			Body:   query,
			Module: module,
		})
	}

	return module
}

// merge combines two sets of modules and returns the result. The rules from modules
// in 'b' override rules from modules in 'a'. If all rules in a module in 'a' are overridden
// by rules in modules in 'b' then the module from 'a' is discarded.
func (o *optimizer) merge(a, b []bundle.ModuleFile) []bundle.ModuleFile {

	prefixes := ast.NewSet()

	for i := range b {
		// NOTE(tsandall): use a set to memoize the prefix add operation--it's only
		// needed once per rule set and constructing the path for every rule in the
		// module could expensive for PE output (which can contain hundreds of thousands
		// of rules.)
		seen := ast.NewVarSet()
		for _, rule := range b[i].Parsed.Rules {
			if _, ok := seen[rule.Head.Name]; !ok {
				prefixes.Add(ast.NewTerm(rule.Path()))
				seen.Add(rule.Head.Name)
			}
		}

	}

	for i := range a {

		var keep []*ast.Rule

		// NOTE(tsandall): same as above--memoize keep/discard decision. If multiple
		// entrypoints are provided the dst module may contain a large number of rules.
		seen := ast.NewVarSet()
		discard := ast.NewVarSet()

		for _, rule := range a[i].Parsed.Rules {

			if _, ok := discard[rule.Head.Name]; ok {
				continue
			} else if _, ok := seen[rule.Head.Name]; ok {
				keep = append(keep, rule)
				continue
			}

			path := rule.Path()
			overlap := prefixes.Until(func(x *ast.Term) bool {
				ref := x.Value.(ast.Ref)
				return path.HasPrefix(ref)
			})

			if overlap {
				discard.Add(rule.Head.Name)
			} else {
				seen.Add(rule.Head.Name)
				keep = append(keep, rule)
			}
		}

		if len(keep) > 0 {
			a[i].Parsed.Rules = keep
			a[i].Raw = nil
			b = append(b, a[i])
		}

	}

	return b
}

func (o *optimizer) getSupportModuleFilename(used map[string]int, module *ast.Module, entrypointIndex int, supportIndex int) string {

	fileName, err := module.Package.Path.Ptr()

	if err == nil && safePathPattern.MatchString(fileName) {
		fileName = o.outputprefix + "/" + fileName
		if c, ok := used[fileName]; ok {
			fileName += fmt.Sprintf(".%d", c)
		}
		used[fileName]++
		fileName += ".rego"
		return fileName
	}

	return fmt.Sprintf("%v/%v/%v/%v.rego", o.outputprefix, o.nsprefix, entrypointIndex, supportIndex)
}

var safePathPattern = regexp.MustCompile(`^[\w-_/]+$`)

func compile(c *ast.Capabilities, b *bundle.Bundle, dbg debug.Debug) (*ast.Compiler, error) {

	modules := map[string]*ast.Module{}

	for _, mf := range b.Modules {
		if _, ok := modules[mf.URL]; ok {
			return nil, fmt.Errorf("duplicate module URL: %s", mf.URL)
		}

		modules[mf.URL] = mf.Parsed
	}

	compiler := ast.NewCompiler().WithCapabilities(c).WithDebug(dbg.Writer())
	compiler.Compile(modules)

	if compiler.Failed() {
		return nil, compiler.Errors
	}

	return compiler, nil
}

func transitiveDocumentDependents(compiler *ast.Compiler, ref *ast.Term, deps map[*ast.Rule]struct{}) {
	for _, rule := range compiler.GetRules(ref.Value.(ast.Ref)) {
		transitiveDependents(compiler, rule, deps)
	}
}

func transitiveDependents(compiler *ast.Compiler, rule *ast.Rule, deps map[*ast.Rule]struct{}) {
	for x := range compiler.Graph.Dependents(rule) {
		other := x.(*ast.Rule)
		deps[other] = struct{}{}
		transitiveDependents(compiler, other, deps)
	}
}

type orderedStringSet []string

func (ss orderedStringSet) Append(s ...string) orderedStringSet {
	for _, x := range s {
		var found bool
		for _, other := range ss {
			if x == other {
				found = true
			}
		}
		if !found {
			ss = append(ss, x)
		}
	}
	return ss
}

func (ss orderedStringSet) Contains(s string) bool {
	for _, other := range ss {
		if s == other {
			return true
		}
	}
	return false
}

func stringsToRefs(x []string) []ast.Ref {
	result := make([]ast.Ref, len(x))
	for i := range result {
		result[i] = storage.MustParsePath("/" + x[i]).Ref(ast.DefaultRootDocument)
	}
	return result
}

type refSet struct {
	s []ast.Ref
}

func newRefSet(x ...ast.Ref) *refSet {
	result := &refSet{}
	for i := range x {
		result.AddPrefix(x[i])
	}
	return result
}

// ContainsPrefix returns true if r is prefixed by any of the existing refs in the set.
func (rs *refSet) ContainsPrefix(r ast.Ref) bool {
	for i := range rs.s {
		if r.HasPrefix(rs.s[i]) {
			return true
		}
	}
	return false
}

// AddPrefix inserts r into the set if r is not prefixed by any existing
// refs in the set. If any existing refs are prefixed by r, those existing
// refs are removed.
func (rs *refSet) AddPrefix(r ast.Ref) {
	if rs.ContainsPrefix(r) {
		return
	}
	cpy := []ast.Ref{r}
	for i := range rs.s {
		if !rs.s[i].HasPrefix(r) {
			cpy = append(cpy, rs.s[i])
		}
	}
	rs.s = cpy
}

// Sorted returns a sorted slice of terms for refs in the set.
func (rs *refSet) Sorted() []*ast.Term {
	terms := make([]*ast.Term, len(rs.s))
	for i := range rs.s {
		terms[i] = ast.NewTerm(rs.s[i])
	}
	sort.Slice(terms, func(i, j int) bool {
		return terms[i].Value.Compare(terms[j].Value) < 0
	})
	return terms
}
