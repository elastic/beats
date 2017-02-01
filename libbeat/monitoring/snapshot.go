package monitoring

import "strings"

// FlatSnapshot represents a flatten snapshot of all metrics.
// Names in the tree will be joined with `.` .
type FlatSnapshot struct {
	Bools   map[string]bool
	Ints    map[string]int64
	Floats  map[string]float64
	Strings map[string]string
}

type snapshotVisitor struct {
	snapshot FlatSnapshot
	level    []string
}

// CollectFlatSnapshot collects a flattened snapshot of
// a metrics tree start with the given registry.
func CollectFlatSnapshot(r *Registry, expvar bool) FlatSnapshot {
	vs := newSnapshotVisitor()
	r.Visit(vs)
	if expvar {
		VisitExpvars(vs)
	}
	return vs.snapshot
}

func MakeFlatSnapshot() FlatSnapshot {
	return FlatSnapshot{
		Bools:   map[string]bool{},
		Ints:    map[string]int64{},
		Floats:  map[string]float64{},
		Strings: map[string]string{},
	}
}

func newSnapshotVisitor() *snapshotVisitor {
	return &snapshotVisitor{snapshot: MakeFlatSnapshot()}
}

func (vs *snapshotVisitor) OnRegistryStart() error {
	return nil
}

func (vs *snapshotVisitor) OnRegistryFinished() error {
	if len(vs.level) > 0 {
		vs.dropName()
	}
	return nil
}

func (vs *snapshotVisitor) OnKey(name string) error {
	vs.level = append(vs.level, name)
	return nil
}

func (vs *snapshotVisitor) OnKeyNext() error { return nil }

func (vs *snapshotVisitor) getName() string {
	defer vs.dropName()
	if len(vs.level) == 1 {
		return vs.level[0]
	}
	return strings.Join(vs.level, ".")
}

func (vs *snapshotVisitor) dropName() {
	vs.level = vs.level[:len(vs.level)-1]
}

func (vs *snapshotVisitor) OnString(s string) error {
	vs.snapshot.Strings[vs.getName()] = s
	return nil
}

func (vs *snapshotVisitor) OnBool(b bool) error {
	vs.snapshot.Bools[vs.getName()] = b
	return nil
}

func (vs *snapshotVisitor) OnInt(i int64) error {
	vs.snapshot.Ints[vs.getName()] = i
	return nil
}

func (vs *snapshotVisitor) OnFloat(f float64) error {
	vs.snapshot.Floats[vs.getName()] = f
	return nil
}
