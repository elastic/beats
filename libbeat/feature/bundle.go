package feature

import "fmt"

// Bundle defines a list of features available in the current beat.
type Bundle struct {
	Features []Featurable
}

// NewBundle creates a new Bundle of feature to be registered.
func NewBundle(features []Featurable) *Bundle {
	return &Bundle{Features: features}
}

// Filter creates a new bundle with only the feature matching the requested stability.
func (b *Bundle) Filter(stability Stability) *Bundle {
	var filtered []Featurable

	for _, feature := range b.Features {
		if feature.Stability() == stability {
			filtered = append(filtered, feature)
		}
	}
	return NewBundle(filtered)
}

// MustBundle takes existing bundle or features and create a new Bundle with all the merged Features,
// will panic on errors.
func MustBundle(features ...interface{}) *Bundle {
	b, err := BundleFeature(features...)
	if err != nil {
		panic(err)
	}
	return b
}

// BundleFeature takes existing bundle or features and create a new Bundle with all the merged
// Features,
func BundleFeature(features ...interface{}) (*Bundle, error) {
	var merged []Featurable
	for _, feature := range features {
		switch v := feature.(type) {
		case Featurable:
			merged = append(merged, v)
		case *Bundle:
			merged = append(merged, v.Features...)
		default:
			return nil, fmt.Errorf("unknown type, expecting 'Featurable' or 'Bundle' and received '%T'", v)
		}
	}
	return NewBundle(merged), nil
}
