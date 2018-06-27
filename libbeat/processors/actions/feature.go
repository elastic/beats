package actions

import "github.com/elastic/beats/libbeat/feature"

// Bundle bundles all the actions feature.
var Bundle = feature.MustBundle(
	DecodeJSONFieldsFeature,
	DropEventFeature,
	IncludeFieldsFeature,
	RenameFeature,
)
