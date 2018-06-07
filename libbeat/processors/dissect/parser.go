package dissect

import (
	"sort"
)

// parser extracts the useful information from the raw tokenizer string, fields, delimiters and
// skip fields.
type parser struct {
	delimiters []delimiter
	fields     []field
	skipFields []field
}

func newParser(tokenizer string) (*parser, error) {
	// returns pair of delimiter + key
	matches := delimiterRE.FindAllStringSubmatchIndex(tokenizer, -1)
	if len(matches) == 0 {
		return nil, errInvalidTokenizer
	}

	var delimiters []delimiter
	var fields []field

	pos := 0
	for id, m := range matches {
		d := newDelimiter(tokenizer[m[2]:m[3]])
		key := tokenizer[m[4]:m[5]]
		field, err := newField(id, key, d)
		if err != nil {
			return nil, err
		}
		if field.IsGreedy() {
			d.MarkGreedy()
		}
		fields = append(fields, field)
		delimiters = append(delimiters, d)
		pos = m[5] + 1
	}

	if pos < len(tokenizer) {
		d := newDelimiter(tokenizer[pos:])
		delimiters = append(delimiters, d)
	}

	// Chain delimiters between them to make it easier to match them with the string.
	// Some delimiters also need information about their surrounding for decision.
	for i := 0; i < len(delimiters); i++ {
		if i+1 < len(delimiters) {
			delimiters[i].SetNext(delimiters[i+1])
		}
	}

	// group and order append field at the end so the string join is from left to right.
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].Ordinal() < fields[j].Ordinal()
	})

	// List of fields needed for indirection but don't need to appear in the final event.
	var skipFields []field
	for _, f := range fields {
		if !f.IsSaveable() {
			skipFields = append(skipFields, f)
		}
	}

	return &parser{
		delimiters: delimiters,
		fields:     fields,
		skipFields: skipFields,
	}, nil
}
