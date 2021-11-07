package hepa

import "bytes"

type Filter interface {
	Filter([]byte) ([]byte, error)
}

type FilterFunc func([]byte) ([]byte, error)

func (f FilterFunc) Filter(b []byte) ([]byte, error) {
	return f(b)
}

func Rinse(p Purifier, s, r []byte) Purifier {
	return WithFunc(p, func(b []byte) ([]byte, error) {
		b = bytes.ReplaceAll(b, s, r)
		return b, nil
	})
}

func Clean(p Purifier, s []byte) Purifier {
	return WithFunc(p, func(b []byte) ([]byte, error) {
		if bytes.Contains(b, s) {
			return []byte{}, nil
		}
		return b, nil
	})
}

func Noop() FilterFunc {
	return func(b []byte) ([]byte, error) {
		return b, nil
	}
}
