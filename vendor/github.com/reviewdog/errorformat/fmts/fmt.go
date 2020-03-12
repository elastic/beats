//go:generate go run gendoc.go

package fmts

import (
	"fmt"
	"os"
)

// Fmt represents defined errorformat
type Fmt struct {
	// name of this errorformat (recommends program name and must be uniq)
	Name string

	// Errorformat is list of 'errorformat'
	Errorformat []string

	// one-line description
	Description string

	// Reference URL if any
	URL string

	// Target Programming Language of the program.
	Language string
}

// Fmts holds all defined Fmt in this package. key is Fmt.Name.
type Fmts map[string]*Fmt

var (
	fmts       Fmts = make(map[string]*Fmt)
	langToFmts      = make(map[string]Fmts)
)

// register must be called in init().
func register(f *Fmt) {
	if _, ok := fmts[f.Name]; ok {
		fmt.Fprintf(os.Stderr, "%v is already defined: %#v", f.Name, f)
		os.Exit(1)
	}
	fmts[f.Name] = f
	lang := f.Language
	if lang == "" {
		lang = "general"
	}
	if langToFmts[lang] == nil {
		langToFmts[lang] = make(map[string]*Fmt)
	}
	langToFmts[lang][f.Name] = f
}

// DefinedFmts returns all defined errorformats.
func DefinedFmts() Fmts {
	return fmts
}

// DefinedFmtsByLang returns all defined errorformats by language.
func DefinedFmtsByLang() map[string]Fmts {
	return langToFmts
}
