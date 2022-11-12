// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws_vpcflow

// NOTE: This is a minor optimization to avoid one allocation
// from strings.Fields during the run() method. In the benchmarks
// it saves 480 bytes and one allocation. That equates to an increase
// of +628 EPS on an M1 Max, but relatively minor ns/op improvement
// of +0.21%. In other words, it might not be worth it.

// wordIterator iterates over space separated words in an ASCII string.
type wordIterator struct {
	source          string
	currentWord     string
	currentPosition int
	wordIndex       int
}

func newWordIterator(s string) *wordIterator {
	return &wordIterator{source: s, wordIndex: -1}
}

func (itr *wordIterator) Next() bool {
	// ASCII fast path
	s := itr.source[itr.currentPosition:]
	fieldStart := 0
	i := 0
	// Skip spaces in the front of the input.
	for i < len(s) && s[i] == ' ' {
		i++
	}
	fieldStart = i

	for i < len(s) {
		if s[i] != ' ' {
			i++
			continue
		}
		itr.currentWord = s[fieldStart:i]
		itr.currentPosition += i
		itr.wordIndex++
		return true
	}
	if fieldStart < len(s) { // Last field might end at EOF.
		itr.currentWord = s[fieldStart:]
		itr.currentPosition += len(s)
		itr.wordIndex++
		return true
	}
	itr.currentWord = ""
	itr.wordIndex = -1
	return false
}

func (itr *wordIterator) Word() string {
	return itr.currentWord
}

func (itr *wordIterator) WordIndex() int {
	return itr.wordIndex
}

func (itr *wordIterator) Count() int {
	n := 0
	wasSpace := true

	for i := 0; i < len(itr.source); i++ {
		r := itr.source[i]
		isSpace := r == ' '
		if wasSpace && !isSpace {
			n++
		}
		wasSpace = isSpace
	}
	return n
}
