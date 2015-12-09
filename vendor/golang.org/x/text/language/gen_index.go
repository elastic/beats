// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

package main

// This file generates derivative tables based on the language package itself.

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"reflect"
	"sort"
	"strings"

	"golang.org/x/text/cldr"
	"golang.org/x/text/internal/gen"
	"golang.org/x/text/language"
)

var (
	test = flag.Bool("test", false,
		"test existing tables; can be used to compare web data with package data.")

	draft = flag.String("draft",
		"contributed",
		`Minimal draft requirements (approved, contributed, provisional, unconfirmed).`)
)

func main() {
	gen.Init()

	// Read the CLDR zip file.
	r := gen.OpenCLDRCoreZip()
	defer r.Close()

	d := &cldr.Decoder{}
	data, err := d.DecodeZip(r)
	if err != nil {
		log.Fatalf("DecodeZip: %v", err)
	}

	w := gen.NewCodeWriter()
	defer func() {
		buf := &bytes.Buffer{}

		if _, err = w.WriteGo(buf, "language"); err != nil {
			log.Fatalf("Error formatting file index.go: %v", err)
		}

		// Since we're generating a table for our own package we need to rewrite
		// doing the equivalent of go fmt -r 'language.b -> b'. Using
		// bytes.Replace will do.
		out := bytes.Replace(buf.Bytes(), []byte("language."), nil, -1)
		if err := ioutil.WriteFile("index.go", out, 0600); err != nil {
			log.Fatalf("Could not create file index.go: %v", err)
		}
	}()

	m := map[language.Tag]bool{}
	for _, lang := range data.Locales() {
		if x := data.RawLDML(lang); false ||
			x.LocaleDisplayNames != nil ||
			x.Characters != nil ||
			x.Delimiters != nil ||
			x.Measurement != nil ||
			x.Dates != nil ||
			x.Numbers != nil ||
			x.Units != nil ||
			x.ListPatterns != nil ||
			x.Collations != nil ||
			x.Segmentations != nil ||
			x.Rbnf != nil ||
			x.Annotations != nil ||
			x.Metadata != nil {

			// TODO: support POSIX natively, albeit non-standard.
			tag := language.Make(strings.Replace(lang, "_POSIX", "-u-va-posix", 1))
			m[tag] = true
		}
	}
	var core, special []language.Tag

	for t := range m {
		if x := t.Extensions(); len(x) != 0 && fmt.Sprint(x) != "[u-va-posix]" {
			log.Fatalf("Unexpected extension %v in %v", x, t)
		}
		if len(t.Variants()) == 0 && len(t.Extensions()) == 0 {
			core = append(core, t)
		} else {
			special = append(special, t)
		}
	}

	w.WriteComment(`
	NumCompactTags is the number of common tags. The maximum tag is
	NumCompactTags-1.`)
	w.WriteConst("NumCompactTags", len(core)+len(special))

	sort.Sort(byAlpha(special))
	w.WriteVar("specialTags", special)

	type coreKey struct {
		base   language.Base
		script language.Script
		region language.Region
	}
	w.WriteType(coreKey{})

	// TODO: order by frequency?
	sort.Sort(byAlpha(core))

	// Size computations are just an estimate.
	w.Size += int(reflect.TypeOf(map[coreKey]uint16{}).Size())
	w.Size += len(core) * int(reflect.TypeOf(coreKey{}).Size()+2) // 2 is for uint16

	fmt.Fprintln(w, "var coreTags = map[coreKey]uint16{")
	fmt.Fprintln(w, "coreKey{}: 0, // und")
	i := len(special) + 1 // Und and special tags already written.
	for _, t := range core {
		if t == language.Und {
			continue
		}
		fmt.Fprint(w.Hash, t, i)
		b, s, r := t.Raw()
		key := fmt.Sprintf("%#v", coreKey{b, s, r})
		key = strings.Replace(key[len("main."):], "language.", "", -1)
		fmt.Fprintf(w, "%s: %d, // %s\n", key, i, t)
		i++
	}
	fmt.Fprintln(w, "}")
}

type byAlpha []language.Tag

func (a byAlpha) Len() int           { return len(a) }
func (a byAlpha) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byAlpha) Less(i, j int) bool { return a[i].String() < a[j].String() }
