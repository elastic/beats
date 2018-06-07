package parse

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/doc"
	"go/parser"
	"go/token"
	"go/types"
	"log"
	"os"
	"os/exec"
	"strings"

	mgTypes "github.com/magefile/mage/types"
)

type PkgInfo struct {
	Funcs            []Function
	DefaultIsError   bool
	DefaultIsContext bool
	DefaultName      string
	DefaultFunc      Function
	Aliases          map[string]string
}

// Function represented a job function from a mage file
type Function struct {
	Name      string
	IsError   bool
	IsContext bool
	Synopsis  string
	Comment   string
}

// TemplateString returns code for the template switch to run the target.
// It wraps each target call to match the func(context.Context) error that
// runTarget requires.
func (f Function) TemplateString() string {
	if f.IsContext && f.IsError {
		out := `wrapFn := func(ctx context.Context) error {
				return %s(ctx)
			}
			err := runTarget(wrapFn)`
		return fmt.Sprintf(out, f.Name)
	}
	if f.IsContext && !f.IsError {
		out := `wrapFn := func(ctx context.Context) error {
				%s(ctx)
				return nil
			}
			err := runTarget(wrapFn)`
		return fmt.Sprintf(out, f.Name)
	}
	if !f.IsContext && f.IsError {
		out := `wrapFn := func(ctx context.Context) error {
				return %s()
			}
			err := runTarget(wrapFn)`
		return fmt.Sprintf(out, f.Name)
	}
	if !f.IsContext && !f.IsError {
		out := `wrapFn := func(ctx context.Context) error {
				%s()
				return nil
			}
			err := runTarget(wrapFn)`
		return fmt.Sprintf(out, f.Name)
	}
	return `fmt.Printf("Error formatting job code\n")
	os.Exit(1)`
}

// Package parses a package
func Package(path string, files []string) (*PkgInfo, error) {
	fset := token.NewFileSet()

	pkg, err := getPackage(path, files, fset)
	if err != nil {
		return nil, err
	}

	info, err := makeInfo(path, fset, pkg.Files)
	if err != nil {
		return nil, err
	}

	pi := &PkgInfo{}

	p := doc.New(pkg, "./", 0)
	for _, f := range p.Funcs {
		if f.Recv != "" {
			// skip methods
			continue
		}
		if !ast.IsExported(f.Name) {
			// skip non-exported functions
			continue
		}
		if typ := voidOrError(f.Decl.Type, info); typ != mgTypes.InvalidType {
			pi.Funcs = append(pi.Funcs, Function{
				Name:      f.Name,
				Comment:   f.Doc,
				Synopsis:  sanitizeSynopsis(f),
				IsError:   typ == mgTypes.ErrorType || typ == mgTypes.ContextErrorType,
				IsContext: typ == mgTypes.ContextVoidType || typ == mgTypes.ContextErrorType,
			})
		}
	}

	setDefault(p, pi, info)
	setAliases(p, pi, info)

	return pi, nil
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

func setDefault(p *doc.Package, pi *PkgInfo, info types.Info) {
	for _, v := range p.Vars {
		for x, name := range v.Names {
			if name != "Default" {
				continue
			}
			spec := v.Decl.Specs[x].(*ast.ValueSpec)
			if len(spec.Values) != 1 {
				log.Println("warning: default declaration has multiple values")
			}
			id, ok := spec.Values[0].(*ast.Ident)
			if !ok {
				log.Println("warning: default declaration is not a function name")
			}
			for _, f := range pi.Funcs {
				if f.Name == id.Name {
					pi.DefaultName = f.Name
					pi.DefaultIsError = f.IsError
					pi.DefaultIsContext = f.IsContext
					pi.DefaultFunc = f
					return
				}
			}
			log.Println("warning: default declaration does not reference a mage target")
		}
	}
}

func setAliases(p *doc.Package, pi *PkgInfo, info types.Info) {
	for _, v := range p.Vars {
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
			pi.Aliases = make(map[string]string)
			for _, elem := range comp.Elts {
				kv, ok := elem.(*ast.KeyValueExpr)
				if !ok {
					log.Println("warning: alias declaration is not a map element")
					return
				}
				k, ok := kv.Key.(*ast.BasicLit)
				if !ok || k.Kind != token.STRING {
					log.Println("warning: alias is not a string")
					return
				}
				v, ok := kv.Value.(*ast.Ident)
				if !ok {
					log.Println("warning: alias target is not a function")
					return
				}
				alias := strings.Trim(k.Value, "\"")
				valid := false
				for _, f := range pi.Funcs {
					valid = valid || f.Name == v.Name
				}
				if !valid {
					log.Printf("warning: alias declaration (%s) does not reference a mage target", alias)
				}
				pi.Aliases[alias] = v.Name
			}
			return
		}
	}
}

// getPackage returns the non-test package at the given path.
func getPackage(path string, files []string, fset *token.FileSet) (*ast.Package, error) {
	fm := make(map[string]bool, len(files))
	for _, f := range files {
		fm[f] = true
	}

	filter := func(f os.FileInfo) bool {
		return fm[f.Name()]
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

func makeInfo(dir string, fset *token.FileSet, files map[string]*ast.File) (types.Info, error) {
	goroot := os.Getenv("GOROOT")
	if goroot == "" {
		c := exec.Command("go", "env", "GOROOT")
		b, err := c.Output()
		if err != nil {
			return types.Info{}, fmt.Errorf("failed to get GOROOT from 'go env': %v", err)
		}
		goroot = strings.TrimSpace(string(b))
		if goroot == "" {
			return types.Info{}, fmt.Errorf("could not determine GOROOT")
		}
	}

	build.Default.GOROOT = goroot

	cfg := types.Config{
		Importer: getImporter(fset),
	}

	info := types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}

	fs := make([]*ast.File, 0, len(files))
	for _, v := range files {
		fs = append(fs, v)
	}

	_, err := cfg.Check(dir, fset, fs, &info)
	if err != nil {
		return info, fmt.Errorf("failed to check types in directory: %v", err)
	}
	return info, nil
}

// errorOrVoid filters the list of functions to only those that return only an
// error or have no return value, and have no parameters.
func errorOrVoid(fns []*ast.FuncDecl, info types.Info) []*ast.FuncDecl {
	fds := []*ast.FuncDecl{}

	for _, fn := range fns {
		if voidOrError(fn.Type, info) != mgTypes.InvalidType {
			fds = append(fds, fn)
		}
	}
	return fds
}

func hasContextParam(ft *ast.FuncType, info types.Info) bool {
	if ft.Params.NumFields() == 1 {
		ret := ft.Params.List[0]
		t := info.TypeOf(ret.Type)
		if t != nil && t.String() == "context.Context" {
			return true
		}
	}
	return false
}

func hasVoidReturn(ft *ast.FuncType, info types.Info) bool {
	res := ft.Results
	if res.NumFields() == 0 {
		return true
	}
	return false
}

func hasErrorReturn(ft *ast.FuncType, info types.Info) bool {
	res := ft.Results
	if res.NumFields() == 1 {
		ret := res.List[0]
		if len(ret.Names) > 1 {
			return false
		}
		t := info.TypeOf(ret.Type)
		if t != nil && t.String() == "error" {
			return true
		}
	}
	return false
}

func voidOrError(ft *ast.FuncType, info types.Info) mgTypes.FuncType {
	if hasContextParam(ft, info) {
		if hasVoidReturn(ft, info) {
			return mgTypes.ContextVoidType
		}
		if hasErrorReturn(ft, info) {
			return mgTypes.ContextErrorType
		}
	}
	if ft.Params.NumFields() == 0 {
		if hasVoidReturn(ft, info) {
			return mgTypes.VoidType
		}
		if hasErrorReturn(ft, info) {
			return mgTypes.ErrorType
		}
	}
	return mgTypes.InvalidType
}
