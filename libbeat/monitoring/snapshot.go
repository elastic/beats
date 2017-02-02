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
func CollectFlatSnapshot(r *Registry, mode Mode, expvar bool) FlatSnapshot {
	vs := newSnapshotVisitor()
	r.Visit(mode, vs)
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

func (vs *snapshotVisitor) OnRegistryStart() {}

func (vs *snapshotVisitor) OnRegistryFinished() {
	if len(vs.level) > 0 {
		vs.dropName()
	}
}

func (vs *snapshotVisitor) OnKey(name string) {
	vs.level = append(vs.level, name)
}

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

func (vs *snapshotVisitor) OnString(s string) {
	vs.snapshot.Strings[vs.getName()] = s
}

func (vs *snapshotVisitor) OnBool(b bool) {
	vs.snapshot.Bools[vs.getName()] = b
}

func (vs *snapshotVisitor) OnInt(i int64) {
	vs.snapshot.Ints[vs.getName()] = i
}

func (vs *snapshotVisitor) OnFloat(f float64) {
	vs.snapshot.Floats[vs.getName()] = f
}
