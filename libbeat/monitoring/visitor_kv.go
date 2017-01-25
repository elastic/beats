package monitoring

import "strings"

type KeyValueVisitor struct {
	cb    func(key string, value interface{}) error
	level []string
}

func NewKeyValueVisitor(cb func(string, interface{}) error) *KeyValueVisitor {
	return &KeyValueVisitor{cb: cb}
}

func (vs *KeyValueVisitor) OnRegistryStart() error {
	return nil
}

func (vs *KeyValueVisitor) OnRegistryFinished() error {
	if len(vs.level) > 0 {
		vs.dropName()
	}
	return nil
}

func (vs *KeyValueVisitor) OnKey(name string) error {
	vs.level = append(vs.level, name)
	return nil
}

func (vs *KeyValueVisitor) OnKeyNext() error { return nil }

func (vs *KeyValueVisitor) getName() string {
	defer vs.dropName()
	if len(vs.level) == 1 {
		return vs.level[0]
	}
	return strings.Join(vs.level, ".")
}

func (vs *KeyValueVisitor) dropName() {
	vs.level = vs.level[:len(vs.level)-1]
}

func (vs *KeyValueVisitor) OnString(s string) error {
	return vs.cb(vs.getName(), s)
}

func (vs *KeyValueVisitor) OnBool(b bool) error {
	return vs.cb(vs.getName(), b)
}

func (vs *KeyValueVisitor) OnNil() error {
	return vs.cb(vs.getName(), nil)
}

func (vs *KeyValueVisitor) OnInt(i int64) error {
	return vs.cb(vs.getName(), i)
}

func (vs *KeyValueVisitor) OnFloat(f float64) error {
	return vs.cb(vs.getName(), f)
}
