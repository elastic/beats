package gotype

import "strings"

type tagOptions struct {
	squash    bool
	omitEmpty bool
}

var defaultTagOptions = tagOptions{
	squash:    false,
	omitEmpty: false,
}

func parseTags(tag string) (string, tagOptions) {
	s := strings.Split(tag, ",")
	if len(s) == 0 {
		return "", defaultTagOptions
	}
	opts := defaultTagOptions
	for _, opt := range s[1:] {
		switch strings.TrimSpace(opt) {
		case "squash", "inline":
			opts.squash = true
		case "omitempty":
			opts.omitEmpty = true
		}
	}
	return strings.TrimSpace(s[0]), opts
}
