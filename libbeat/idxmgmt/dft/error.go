package dft

import "errors"

type Error struct {
	reason  error
	cause   error
	message string
}

var (
	ErrESDFDisabled = errors.New("Dataframes are disabled in Elasticsearch")
)
