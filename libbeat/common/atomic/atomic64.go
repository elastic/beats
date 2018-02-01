// +build amd64 arm64 ppc64 ppc64le mips64 mips64le s390x

package atomic

// atomic Uint/Int for 64bit systems

// Uint provides an architecture specific atomic uint.
type Uint struct{ a Uint64 }

// Int provides an architecture specific atomic uint.
type Int struct{ a Int64 }

func MakeUint(v uint) Uint             { return Uint{MakeUint64(uint64(v))} }
func NewUint(v uint) *Uint             { return &Uint{MakeUint64(uint64(v))} }
func (u *Uint) Load() uint             { return uint(u.a.Load()) }
func (u *Uint) Store(v uint)           { u.a.Store(uint64(v)) }
func (u *Uint) Swap(new uint) uint     { return uint(u.a.Swap(uint64(new))) }
func (u *Uint) Add(delta uint) uint    { return uint(u.a.Add(uint64(delta))) }
func (u *Uint) Sub(delta uint) uint    { return uint(u.a.Add(uint64(-delta))) }
func (u *Uint) Inc() uint              { return uint(u.a.Inc()) }
func (u *Uint) Dec() uint              { return uint(u.a.Dec()) }
func (u *Uint) CAS(old, new uint) bool { return u.a.CAS(uint64(old), uint64(new)) }

func MakeInt(v int) Int              { return Int{MakeInt64(int64(v))} }
func NewInt(v int) *Int              { return &Int{MakeInt64(int64(v))} }
func (i *Int) Load() int             { return int(i.a.Load()) }
func (i *Int) Store(v int)           { i.a.Store(int64(v)) }
func (i *Int) Swap(new int) int      { return int(i.a.Swap(int64(new))) }
func (i *Int) Add(delta int) int     { return int(i.a.Add(int64(delta))) }
func (i *Int) Sub(delta int) int     { return int(i.a.Add(int64(-delta))) }
func (i *Int) Inc() int              { return int(i.a.Inc()) }
func (i *Int) Dec() int              { return int(i.a.Dec()) }
func (i *Int) CAS(old, new int) bool { return i.a.CAS(int64(old), int64(new)) }
