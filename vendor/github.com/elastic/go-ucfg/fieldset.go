package ucfg

type fieldSet struct {
	fields map[string]struct{}
	parent *fieldSet
}

func NewFieldSet(parent *fieldSet) *fieldSet {
	return &fieldSet{
		fields: map[string]struct{}{},
		parent: parent,
	}
}

func (s *fieldSet) Has(name string) (exists bool) {
	if _, exists = s.fields[name]; !exists && s.parent != nil {
		exists = s.parent.Has(name)
	}
	return
}

func (s *fieldSet) Add(name string) {
	s.fields[name] = struct{}{}
}

func (s *fieldSet) AddNew(name string) (ok bool) {
	if ok = !s.Has(name); ok {
		s.Add(name)
	}
	return
}

func (s *fieldSet) Names() []string {
	var names []string
	for k := range s.fields {
		names = append(names, k)
	}

	if s.parent != nil {
		names = append(names, s.parent.Names()...)
	}
	return names
}
