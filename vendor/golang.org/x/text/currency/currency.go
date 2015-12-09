// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:generate go run gen.go gen_common.go -output tables.go

// Package currency contains currency-related functionality.
package currency

import (
	"errors"
	"sort"

	"golang.org/x/text/internal/tag"
	"golang.org/x/text/language"
)

// TODO:
// - language-specific currency names.
// - currency formatting.
// - currency information per region
// - register currency code (there are no private use area)

// TODO: remove Currency type from package language.

// Kind determines the rounding and rendering properties of a currency value.
type Kind struct {
	rounding rounding
	// TODO: formatting type: standard, accounting. See CLDR.
}

type rounding byte

const (
	standard rounding = iota
	cash
)

var (
	// Standard defines standard rounding and formatting for currencies.
	Standard Kind = Kind{rounding: standard}

	// Cash defines rounding and formatting standards for cash transactions.
	Cash Kind = Kind{rounding: cash}

	// Accounting defines rounding and formatting standards for accounting.
	Accounting Kind = Kind{rounding: standard}
)

// Rounding reports the rounding characteristics for the given currency, where
// scale is the number of fractional decimals and increment is the number of
// units in terms of 10^(-scale) to which to round to.
func (k Kind) Rounding(c Currency) (scale, increment int) {
	info := currency.Elem(int(c.index))[3]
	switch k.rounding {
	case standard:
		info &= roundMask
	case cash:
		info >>= cashShift
	}
	return int(roundings[info].scale), int(roundings[info].increment)
}

// Currency is an ISO 4217 currency designator.
type Currency struct {
	index uint16
}

// String returns the ISO code of c.
func (c Currency) String() string {
	if c.index == 0 {
		return "XXX"
	}
	return currency.Elem(int(c.index))[:3]
}

// Value creates a Value for the given currency and amount.
func (c Currency) Value(amount interface{}) Value {
	// TODO: verify amount is a supported number type
	return Value{amount: amount, currency: c}
}

var (
	errSyntax = errors.New("currency: tag is not well-formed")
	errValue  = errors.New("currency: tag is not a recognized currency")
)

// ParseISO parses a 3-letter ISO 4217 code. It returns an error if s not
// well-formed or not a recognized currency code.
func ParseISO(s string) (Currency, error) {
	var buf [4]byte // Take one byte more to detect oversize keys.
	key := buf[:copy(buf[:], s)]
	if !tag.FixCase("XXX", key) {
		return Currency{}, errSyntax
	}
	if i := currency.Index(key); i >= 0 {
		return Currency{uint16(i)}, nil
	}
	return Currency{}, errValue
}

// MustParseISO is like ParseISO, but panics if the given currency
// cannot be parsed. It simplifies safe initialization of Currency values.
func MustParseISO(s string) Currency {
	c, err := ParseISO(s)
	if err != nil {
		panic(err)
	}
	return c
}

// FromRegion reports the Currency that is currently legal tender in the given
// region according to CLDR. It will return false if region currently does not
// have a legal tender.
func FromRegion(r language.Region) (tender Currency, ok bool) {
	x := regionToCode(r)
	i := sort.Search(len(regionToCurrency), func(i int) bool {
		return regionToCurrency[i].region >= x
	})
	if i < len(regionToCurrency) && regionToCurrency[i].region == x {
		return Currency{regionToCurrency[i].code}, true
	}
	return Currency{0}, false
}

// FromTag reports the most likely currency for the given tag. It considers the
// currency defined in the -u extension and infers the region if necessary.
func FromTag(t language.Tag) (Currency, language.Confidence) {
	if cur := t.TypeForKey("cu"); len(cur) == 3 {
		var buf [3]byte
		copy(buf[:], cur)
		tag.FixCase("XXX", buf[:])
		if x := currency.Index(buf[:]); x > 0 {
			return Currency{uint16(x)}, language.Exact
		}
	}
	r, conf := t.Region()
	if cur, ok := FromRegion(r); ok {
		return cur, conf
	}
	return Currency{}, language.No
}

var (
	// Undefined and testing.
	XXX Currency = Currency{xxx}
	XTS Currency = Currency{xts}

	// G10 currencies https://en.wikipedia.org/wiki/G10_currencies.
	USD Currency = Currency{usd}
	EUR Currency = Currency{eur}
	JPY Currency = Currency{jpy}
	GBP Currency = Currency{gbp}
	CHF Currency = Currency{chf}
	AUD Currency = Currency{aud}
	NZD Currency = Currency{nzd}
	CAD Currency = Currency{cad}
	SEK Currency = Currency{sek}
	NOK Currency = Currency{nok}

	// Additional common currencies as defined by CLDR.
	BRL Currency = Currency{brl}
	CNY Currency = Currency{cny}
	DKK Currency = Currency{dkk}
	INR Currency = Currency{inr}
	RUB Currency = Currency{rub}
	HKD Currency = Currency{hkd}
	IDR Currency = Currency{idr}
	KRW Currency = Currency{krw}
	MXN Currency = Currency{mxn}
	PLN Currency = Currency{pln}
	SAR Currency = Currency{sar}
	THB Currency = Currency{thb}
	TRY Currency = Currency{try}
	TWD Currency = Currency{twd}
	ZAR Currency = Currency{zar}

	// Precious metals.
	XAG Currency = Currency{xag}
	XAU Currency = Currency{xau}
	XPT Currency = Currency{xpt}
	XPD Currency = Currency{xpd}
)
