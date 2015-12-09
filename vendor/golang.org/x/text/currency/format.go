// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package currency

import (
	"fmt"
	"io"
	"sort"

	"golang.org/x/text/internal"
	"golang.org/x/text/internal/format"
	"golang.org/x/text/language"
)

// Value is an amount-currency pair configured for language-specific formatting.
type Value struct {
	amount   interface{}
	currency Currency
	format   *options
}

// Verify implementation
var _ fmt.Formatter = Value{}

// Currency reports the Currency of this value.
func (v Value) Currency() Currency { return v.currency }

// Amount reports the amount of this Value
func (v Value) Amount() interface{} { return v.amount }

var space = []byte(" ")

// Format implements fmt.Formatter. It accepts format.State for
// language-specific rendering.
func (v Value) Format(s fmt.State, verb rune) {
	var lang int
	if state, ok := s.(format.State); ok {
		lang, _ = language.CompactIndex(state.Language())
	}

	// Get the options. Use DefaultFormat if not present.
	opt := v.format
	if opt == nil {
		opt = defaultFormat
	}
	cur := v.currency
	if cur.index == 0 {
		cur = opt.currency
	}

	// TODO: use pattern.
	io.WriteString(s, opt.symbol(lang, cur))
	if v.amount != nil {
		s.Write(space)

		// TODO: apply currency-specific rounding
		scale, _ := opt.kind.Rounding(cur)
		if _, ok := s.Precision(); !ok {
			fmt.Fprintf(s, "%.*f", scale, v.amount)
		} else {
			fmt.Fprint(s, v.amount)
		}
	}
}

// Formatter decorates a given number, Currency or Value with formatting options.
type Formatter func(value interface{}) Value

// TODO: call this a Formatter or FormatFunc?

var dummy = USD.Value(0)

// adjust creates a new Formatter based on the adjustments of fn on f.
func (f Formatter) adjust(fn func(*options)) Formatter {
	var o options = *(f(dummy).format)
	fn(&o)
	return o.format
}

// Default creates a new Formatter that defaults to currency c if a numeric
// value is passed that is not associated with a currency.
func (f Formatter) Default(c Currency) Formatter {
	return f.adjust(func(o *options) { o.currency = c })
}

// Kind sets the kind of the underlying currency.
func (f Formatter) Kind(k Kind) Formatter {
	return f.adjust(func(o *options) { o.kind = k })
}

var defaultFormat *options = ISO(dummy).format

var (
	// Uses Narrow symbols. Overrides Symbol, if present.
	NarrowSymbol Formatter = Formatter(formNarrow)

	// Use Symbols instead of ISO codes, when available.
	Symbol Formatter = Formatter(formSymbol)

	// Use ISO code as symbol.
	ISO Formatter = Formatter(formISO)

	// TODO:
	// // Use full name as symbol.
	// SpellOut Formatter
	//
	// // SpellOutAll causes symbol and numbers to be spelled wide. If used in
	// // combination with the Symbol option, the symbol will be written in ISO
	// // format.
	// SpellOutAll Formatter
)

// options configures rendering and rounding options for a Value.
type options struct {
	currency Currency
	kind     Kind

	symbol func(compactIndex int, c Currency) string
}

func (o *options) format(value interface{}) Value {
	v := Value{format: o}
	switch x := value.(type) {
	case Value:
		v.currency = x.currency
		v.amount = x.amount
	case *Value:
		v.currency = x.currency
		v.amount = x.amount
	case Currency:
		v.currency = x
	case *Currency:
		v.currency = *x
	default:
		if o.currency.index == 0 {
			panic("cannot format number without a currency being set")
		}
		// TODO: Must be a number.
		v.amount = x
	}
	return v
}

var (
	optISO    = options{symbol: lookupISO}
	optSymbol = options{symbol: lookupSymbol}
	optNarrow = options{symbol: lookupNarrow}
)

// These need to be functions, rather than curried methods, as curried methods
// are evaluated at init time, causing tables to be included unconditionally.
func formISO(x interface{}) Value    { return optISO.format(x) }
func formSymbol(x interface{}) Value { return optSymbol.format(x) }
func formNarrow(x interface{}) Value { return optNarrow.format(x) }

func lookupISO(x int, c Currency) string    { return c.String() }
func lookupSymbol(x int, c Currency) string { return normalSymbol.lookup(x, c) }
func lookupNarrow(x int, c Currency) string { return narrowSymbol.lookup(x, c) }

type symbolIndex struct {
	index []uint16 // position corresponds with compact index of language.
	data  []curToIndex
}

var (
	normalSymbol = symbolIndex{normalLangIndex, normalSymIndex}
	narrowSymbol = symbolIndex{narrowLangIndex, narrowSymIndex}
)

func (x *symbolIndex) lookup(lang int, c Currency) string {
	for {
		index := x.data[x.index[lang]:x.index[lang+1]]
		i := sort.Search(len(index), func(i int) bool {
			return index[i].cur >= c.index
		})
		if i < len(index) && index[i].cur == c.index {
			x := index[i].idx
			start := x + 1
			end := start + uint16(symbols[x])
			if start == end {
				return c.String()
			}
			return symbols[start:end]
		}
		if lang == 0 {
			break
		}
		lang = int(internal.Parent[lang])
	}
	return c.String()
}
