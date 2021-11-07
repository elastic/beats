package hepa

func WithFunc(p Purifier, fn FilterFunc) Purifier {
	c := New()
	c.parent = &p
	c.filter = fn
	return c
}

func With(p Purifier, f Filter) Purifier {
	c := New()
	c.parent = &p
	c.filter = f
	return c
}
