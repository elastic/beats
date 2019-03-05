package parse

import (
	"errors"
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/magefile/mage/internal"
)

const importTag = "mage:import"

var debug = log.New(ioutil.Discard, "DEBUG: ", log.Ltime|log.Lmicroseconds)

// EnableDebug turns on debug logging.
func EnableDebug() {
	debug.SetOutput(os.Stderr)
}

// PkgInfo contains inforamtion about a package of files according to mage's
// parsing rules.
type PkgInfo struct {
	AstPkg      *ast.Package
	DocPkg      *doc.Package
	Description string
	Funcs       []*Function
	DefaultFunc *Function
	Aliases     map[string]*Function
	Imports     []*Import
}

// Function represented a job function from a mage file
type Function struct {
	PkgAlias   string
	Package    string
	ImportPath string
	Name       string
	Receiver   string
	IsError    bool
	IsContext  bool
	Synopsis   string
	Comment    string
}

// ID returns user-readable information about where this function is defined.
func (f Function) ID() string {
	path := "<current>"
	if f.ImportPath != "" {
		path = f.ImportPath
	}
	receiver := ""
	if f.Receiver != "" {
		receiver = f.Receiver + "."
	}
	return fmt.Sprintf("%s.%s%s", path, receiver, f.Name)
}

// TargetName returns the name of the target as it should appear when used from
// the mage cli.  It is always lowercase.
func (f Function) TargetName() string {
	var names []string

	for _, s := range []string{f.PkgAlias, f.Receiver, f.Name} {
		if s != "" {
			names = append(names, s)
		}
	}
	return strings.Join(names, ":")
}

// ExecCode returns code for the template switch to run the target.
// It wraps each target call to match the func(context.Context) error that
// runTarget requires.
func (f Function) ExecCode() (string, error) {
	name := f.Name
	if f.Receiver != "" {
		name = f.Receiver + "{}." + name
	}
	if f.Package != "" {
		name = f.Package + "." + name
	}

	if f.IsContext && f.IsError {
		out := `
			wrapFn := func(ctx context.Context) error {
				return %s(ctx)
			}
			err := runTarget(wrapFn)`[1:]
		return fmt.Sprintf(out, name), nil
	}
	if f.IsContext && !f.IsError {
		out := `
			wrapFn := func(ctx context.Context) error {
				%s(ctx)
				return nil
			}
			err := runTarget(wrapFn)`[1:]
		return fmt.Sprintf(out, name), nil
	}
	if !f.IsContext && f.IsError {
		out := `
			wrapFn := func(ctx context.Context) error {
				return %s()
			}
			err := runTarget(wrapFn)`[1:]
		return fmt.Sprintf(out, name), nil
	}
	if !f.IsContext && !f.IsError {
		out := `
			wrapFn := func(ctx context.Context) error {
				%s()
				return nil
			}
			err := runTarget(wrapFn)`[1:]
		return fmt.Sprintf(out, name), nil
	}
	return "", fmt.Errorf("Error formatting ExecCode code for %#v", f)
}

// PrimaryPackage parses a package.  If files is non-empty, it will only parse the files given.
func PrimaryPackage(gocmd, path string, files []string) (*PkgInfo, error) {
	info, err := Package(path, files)
	if err != nil {
		return nil, err
	}

	if err := setImports(gocmd, info); err != nil {
		return nil, err
	}

	setDefault(info)
	setAliases(info)
	return info, nil
}

func checkDupes(info *PkgInfo, imports []*Import) error {
	funcs := map[string][]*Function{}
	for _, f := range info.Funcs {
		funcs[strings.ToLower(f.TargetName())] = append(funcs[strings.ToLower(f.TargetName())], f)
	}
	for _, imp := range imports {
		for _, f := range imp.Info.Funcs {
			target := strings.ToLower(f.TargetName())
			funcs[target] = append(funcs[target], f)
		}
	}
	for alias, f := range info.Aliases {
		if len(funcs[alias]) != 0 {
			var ids []string
			for _, f := range funcs[alias] {
				ids = append(ids, f.ID())
			}
			return fmt.Errorf("alias %q duplicates existing target(s): %s\n", alias, strings.Join(ids, ", "))
		}
		funcs[alias] = append(funcs[alias], f)
	}
	var dupes []string
	for target, list := range funcs {
		if len(list) > 1 {
			dupes = append(dupes, target)
		}
	}
	if len(dupes) == 0 {
		return nil
	}
	errs := make([]string, 0, len(dupes))
	for _, d := range dupes {
		var ids []string
		for _, f := range funcs[d] {
			ids = append(ids, f.ID())
		}
		errs = append(errs, fmt.Sprintf("%q target has multiple definitions: %s\n", d, strings.Join(ids, ", ")))
	}
	return errors.New(strings.Join(errs, "\n"))
}

// Package compiles information about a mage package.
func Package(path string, files []string) (*PkgInfo, error) {
	start := time.Now()
	defer func() {
		debug.Println("time parse Magefiles:", time.Since(start))
	}()
	fset := token.NewFileSet()
	pkg, err := getPackage(path, files, fset)
	if err != nil {
		return nil, err
	}
	p := doc.New(pkg, "./", 0)
	pi := &PkgInfo{
		AstPkg:      pkg,
		DocPkg:      p,
		Description: toOneLine(p.Doc),
	}

	setNamespaces(pi)
	setFuncs(pi)

	hasDupes, names := checkDupeTargets(pi)
	if hasDupes {
		msg := "Build targets must be case insensitive, thus the following targets conflict:\n"
		for _, v := range names {
			if len(v) > 1 {
				msg += "  " + strings.Join(v, ", ") + "\n"
			}
		}
		return nil, errors.New(msg)
	}

	return pi, nil
}

func getNamedImports(gocmd string, pkgs map[string]string) ([]*Import, error) {
	var imports []*Import
	for alias, pkg := range pkgs {
		debug.Printf("getting import package %q, alias %q", pkg, alias)
		imp, err := getImport(gocmd, pkg, alias)
		if err != nil {
			return nil, err
		}
		imports = append(imports, imp)
	}
	return imports, nil
}

func getImport(gocmd, importpath, alias string) (*Import, error) {
	out, err := internal.OutputDebug(gocmd, "list", "-f", "{{.Dir}}||{{.Name}}", importpath)
	if err != nil {
		return nil, err
	}
	parts := strings.Split(out, "||")
	if len(parts) != 2 {
		return nil, fmt.Errorf("incorrect data from go list: %s", out)
	}
	dir, name := parts[0], parts[1]
	debug.Printf("parsing imported package %q from dir %q", importpath, dir)
	info, err := Package(dir, nil)
	if err != nil {
		return nil, err
	}
	for i := range info.Funcs {
		debug.Printf("setting alias %q and package %q on func %v", alias, name, info.Funcs[i].Name)
		info.Funcs[i].PkgAlias = alias
		info.Funcs[i].ImportPath = importpath
	}
	return &Import{Alias: alias, Name: name, Path: importpath, Info: *info}, nil
}

type Import struct {
	Alias      string
	Name       string
	UniqueName string // a name unique across all imports
	Path       string
	Info       PkgInfo
}

func setFuncs(pi *PkgInfo) {
	for _, f := range pi.DocPkg.Funcs {
		if f.Recv != "" {
			debug.Printf("skipping method %s.%s", f.Recv, f.Name)
			// skip methods
			continue
		}
		if !ast.IsExported(f.Name) {
			debug.Printf("skipping non-exported function %s", f.Name)
			// skip non-exported functions
			continue
		}
		if typ := funcType(f.Decl.Type); typ != invalidType {
			debug.Printf("found target %v", f.Name)
			pi.Funcs = append(pi.Funcs, &Function{
				Name:      f.Name,
				Comment:   toOneLine(f.Doc),
				Synopsis:  sanitizeSynopsis(f),
				IsError:   typ == errorType || typ == contextErrorType,
				IsContext: typ == contextVoidType || typ == contextErrorType,
			})
		} else {
			debug.Printf("skipping function with invalid signature func %s(%v)(%v)", f.Name, fieldNames(f.Decl.Type.Params), fieldNames(f.Decl.Type.Results))
		}
	}
}

func setNamespaces(pi *PkgInfo) {
	for _, t := range pi.DocPkg.Types {
		if !isNamespace(t) {
			continue
		}
		debug.Printf("found namespace %s %s", pi.DocPkg.ImportPath, t.Name)
		for _, f := range t.Methods {
			if !ast.IsExported(f.Name) {
				continue
			}
			typ := funcType(f.Decl.Type)
			if typ == invalidType {
				continue
			}
			debug.Printf("found namespace method %s %s.%s", pi.DocPkg.ImportPath, t.Name, f.Name)
			pi.Funcs = append(pi.Funcs, &Function{
				Name:      f.Name,
				Receiver:  t.Name,
				Comment:   toOneLine(f.Doc),
				Synopsis:  sanitizeSynopsis(f),
				IsError:   typ == errorType || typ == contextErrorType,
				IsContext: typ == contextVoidType || typ == contextErrorType,
			})
		}
	}
}

func setImports(gocmd string, pi *PkgInfo) error {
	importNames := map[string]string{}
	rootImports := []string{}
	for _, f := range pi.AstPkg.Files {
		for _, d := range f.Decls {
			gen, ok := d.(*ast.GenDecl)
			if !ok || gen.Tok != token.IMPORT {
				continue
			}
			for j := 0; j < len(gen.Specs); j++ {
				spec := gen.Specs[j]
				impspec := spec.(*ast.ImportSpec)
				if len(gen.Specs) == 1 && gen.Lparen == token.NoPos && impspec.Doc == nil {
					impspec.Doc = gen.Doc
				}
				name, alias, ok := getImportPath(impspec)
				if !ok {
					continue
				}
				if alias != "" {
					debug.Printf("found %s: %s (%s)", importTag, name, alias)
					if importNames[alias] != "" {
						return fmt.Errorf("duplicate import alias: %q", alias)
					}
					importNames[alias] = name
				} else {
					debug.Printf("found %s: %s", importTag, name)
					rootImports = append(rootImports, name)
				}
			}
		}
	}
	imports, err := getNamedImports(gocmd, importNames)
	if err != nil {
		return err
	}
	for _, s := range rootImports {
		imp, err := getImport(gocmd, s, "")
		if err != nil {
			return err
		}
		imports = append(imports, imp)
	}
	if err := checkDupes(pi, imports); err != nil {
		return err
	}

	// have to set unique package names on imports
	used := map[string]bool{}
	for _, imp := range imports {
		unique := imp.Name + "_mageimport"
		x := 1
		for used[unique] {
			unique = fmt.Sprintf("%s_mageimport%d", imp.Name, x)
			x++
		}
		used[unique] = true
		imp.UniqueName = unique
		for _, f := range imp.Info.Funcs {
			f.Package = unique
		}
	}
	pi.Imports = imports
	return nil
}

func getImportPath(imp *ast.ImportSpec) (path, alias string, ok bool) {
	if imp.Doc == nil || len(imp.Doc.List) == 9 {
		return "", "", false
	}
	// import is always the last comment
	s := imp.Doc.List[len(imp.Doc.List)-1].Text

	// trim comment start and normalize for anyone who has spaces or not between
	// "//"" and the text
	vals := strings.Fields(strings.ToLower(s[2:]))
	if len(vals) == 0 {
		return "", "", false
	}
	if vals[0] != importTag {
		return "", "", false
	}
	path, ok = lit2string(imp.Path)
	if !ok {
		return "", "", false
	}

	switch len(vals) {
	case 1:
		// just the import tag, this is a root import
		return path, "", true
	case 2:
		// also has an alias
		return path, vals[1], true
	default:
		log.Println("warning: ignoring malformed", importTag, "for import", path)
		return "", "", false
	}
}

func isNamespace(t *doc.Type) bool {
	if len(t.Decl.Specs) != 1 {
		return false
	}
	id, ok := t.Decl.Specs[0].(*ast.TypeSpec)
	if !ok {
		return false
	}
	sel, ok := id.Type.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	return ident.Name == "mg" && sel.Sel.Name == "Namespace"
}

func fieldNames(flist *ast.FieldList) string {
	if flist == nil {
		return ""
	}
	list := flist.List
	if len(list) == 0 {
		return ""
	}
	args := make([]string, 0, len(list))
	for _, f := range list {
		names := make([]string, 0, len(f.Names))
		for _, n := range f.Names {
			if n.Name != "" {
				names = append(names, n.Name)
			}
		}
		nms := strings.Join(names, ", ")
		if nms != "" {
			nms += " "
		}
		args = append(args, nms+fmt.Sprint(f.Type))
	}
	return strings.Join(args, ", ")
}

// checkDupeTargets checks a package for duplicate target names.
func checkDupeTargets(info *PkgInfo) (hasDupes bool, names map[string][]string) {
	names = map[string][]string{}
	lowers := map[string]bool{}
	for _, f := range info.Funcs {
		low := strings.ToLower(f.Name)
		if f.Receiver != "" {
			low = strings.ToLower(f.Receiver) + ":" + low
		}
		if lowers[low] {
			hasDupes = true
		}
		lowers[low] = true
		names[low] = append(names[low], f.Name)
	}
	return hasDupes, names
}

// sanitizeSynopsis sanitizes function Doc to create a summary.
func sanitizeSynopsis(f *doc.Func) string {
	synopsis := doc.Synopsis(f.Doc)

	// If the synopsis begins with the function name, remove it. This is done to
	// not repeat the text.
	// From:
	// clean	Clean removes the temporarily generated files
	// To:
	// clean 	removes the temporarily generated files
	if syns := strings.Split(synopsis, " "); strings.EqualFold(f.Name, syns[0]) {
		return strings.Join(syns[1:], " ")
	}

	return synopsis
}

func setDefault(pi *PkgInfo) {
	for _, v := range pi.DocPkg.Vars {
		for x, name := range v.Names {
			if name != "Default" {
				continue
			}
			spec := v.Decl.Specs[x].(*ast.ValueSpec)
			if len(spec.Values) != 1 {
				log.Println("warning: default declaration has multiple values")
			}

			f, err := getFunction(spec.Values[0], pi)
			if err != nil {
				log.Println("warning, default declaration malformed:", err)
				return
			}
			pi.DefaultFunc = f
			return
		}
	}
}

func lit2string(l *ast.BasicLit) (string, bool) {
	if !strings.HasPrefix(l.Value, `"`) || !strings.HasSuffix(l.Value, `"`) {
		return "", false
	}
	return strings.Trim(l.Value, `"`), true
}

func setAliases(pi *PkgInfo) {
	for _, v := range pi.DocPkg.Vars {
		for x, name := range v.Names {
			if name != "Aliases" {
				continue
			}
			spec, ok := v.Decl.Specs[x].(*ast.ValueSpec)
			if !ok {
				log.Println("warning: aliases declaration is not a value")
				return
			}
			if len(spec.Values) != 1 {
				log.Println("warning: aliases declaration has multiple values")
			}
			comp, ok := spec.Values[0].(*ast.CompositeLit)
			if !ok {
				log.Println("warning: aliases declaration is not a map")
				return
			}
			pi.Aliases = map[string]*Function{}
			for _, elem := range comp.Elts {
				kv, ok := elem.(*ast.KeyValueExpr)
				if !ok {
					log.Printf("warning: alias declaration %q is not a map element", elem)
					continue
				}
				k, ok := kv.Key.(*ast.BasicLit)
				if !ok || k.Kind != token.STRING {
					log.Printf("warning: alias key is not a string literal %q", elem)
					continue
				}

				alias, ok := lit2string(k)
				if !ok {
					log.Println("warning: malformed name for alias", elem)
					continue
				}
				f, err := getFunction(kv.Value, pi)
				if err != nil {
					log.Printf("warning, alias malformed: %v", err)
					continue
				}
				pi.Aliases[alias] = f
			}
			return
		}
	}
}

func getFunction(exp ast.Expr, pi *PkgInfo) (*Function, error) {

	// selector expressions are in LIFO format.
	// So, in  foo.bar.baz the first selector.Name is
	// actually "baz", the second is "bar", and the last is "foo"

	var pkg, receiver, funcname string
	switch v := exp.(type) {
	case *ast.Ident:
		// "foo" : Bar
		funcname = v.Name
	case *ast.SelectorExpr:
		// need to handle
		// namespace.Func
		// import.Func
		// import.namespace.Func

		// "foo" : ?.bar
		funcname = v.Sel.Name
		switch x := v.X.(type) {
		case *ast.Ident:
			// "foo" : baz.bar
			// this is either a namespace or package
			firstname := x.Name
			for _, f := range pi.Funcs {
				if firstname == f.Receiver && funcname == f.Name {
					return f, nil
				}
			}
			// not a namespace, let's try imported packages
			for _, imp := range pi.Imports {
				if firstname == imp.Name {
					for _, f := range imp.Info.Funcs {
						if funcname == f.Name {
							return f, nil
						}
					}
					break
				}
			}
			return nil, fmt.Errorf("%q is not a known target", exp)
		case *ast.SelectorExpr:
			// "foo" : bar.Baz.Bat
			// must be package.Namespace.Func
			sel, ok := v.X.(*ast.SelectorExpr)
			if !ok {
				return nil, fmt.Errorf("%q is must denote a target function but was %T", exp, v.X)
			}
			receiver = sel.Sel.Name
			id, ok := sel.X.(*ast.Ident)
			if !ok {
				return nil, fmt.Errorf("%q is must denote a target function but was %T", exp, v.X)
			}
			pkg = id.Name
		default:
			return nil, fmt.Errorf("%q is not valid", exp)
		}
	default:
		return nil, fmt.Errorf("target %s is not a function", exp)
	}
	if pkg == "" {
		for _, f := range pi.Funcs {
			if f.Name == funcname && f.Receiver == receiver {
				return f, nil
			}
		}
		return nil, fmt.Errorf("unknown function %s.%s", receiver, funcname)
	}
	for _, imp := range pi.Imports {
		if imp.Name == pkg {
			for _, f := range imp.Info.Funcs {
				if f.Name == funcname && f.Receiver == receiver {
					return f, nil
				}
			}
			return nil, fmt.Errorf("unknown function %s.%s.%s", pkg, receiver, funcname)
		}
	}
	return nil, fmt.Errorf("unknown package for function %q", exp)
}

// getPackage returns the non-test package at the given path.
func getPackage(path string, files []string, fset *token.FileSet) (*ast.Package, error) {
	var filter func(f os.FileInfo) bool
	if len(files) > 0 {
		fm := make(map[string]bool, len(files))
		for _, f := range files {
			fm[f] = true
		}

		filter = func(f os.FileInfo) bool {
			return fm[f.Name()]
		}
	}

	pkgs, err := parser.ParseDir(fset, path, filter, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse directory: %v", err)
	}

	for name, pkg := range pkgs {
		if !strings.HasSuffix(name, "_test") {
			return pkg, nil
		}
	}
	return nil, fmt.Errorf("no non-test packages found in %s", path)
}

func hasContextParam(ft *ast.FuncType) bool {
	if ft.Params.NumFields() != 1 {
		return false
	}
	ret := ft.Params.List[0]
	sel, ok := ret.Type.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	if pkg.Name != "context" {
		return false
	}
	return sel.Sel.Name == "Context"
}

func hasVoidReturn(ft *ast.FuncType) bool {
	res := ft.Results
	return res.NumFields() == 0
}

func hasErrorReturn(ft *ast.FuncType) bool {
	res := ft.Results
	if res.NumFields() != 1 {
		return false
	}
	ret := res.List[0]
	if len(ret.Names) > 1 {
		return false
	}
	return fmt.Sprint(ret.Type) == "error"
}

type functype int

const (
	invalidType functype = iota
	voidType
	errorType
	contextVoidType
	contextErrorType
)

func funcType(ft *ast.FuncType) functype {
	if hasContextParam(ft) {
		if hasVoidReturn(ft) {
			return contextVoidType
		}
		if hasErrorReturn(ft) {
			return contextErrorType
		}
	}
	if ft.Params.NumFields() == 0 {
		if hasVoidReturn(ft) {
			return voidType
		}
		if hasErrorReturn(ft) {
			return errorType
		}
	}
	return invalidType
}

func toOneLine(s string) string {
	return strings.TrimSpace(strings.Replace(s, "\n", " ", -1))
}
