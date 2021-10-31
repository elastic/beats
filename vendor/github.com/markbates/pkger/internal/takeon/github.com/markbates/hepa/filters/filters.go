package filters

type FilterFunc func([]byte) ([]byte, error)

func (f FilterFunc) Filter(b []byte) ([]byte, error) {
	return f(b)
}

type dir struct {
	Dir string
	Err error
}
