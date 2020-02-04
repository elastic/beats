package ccall

/*
#cgo CFLAGS: -DGVLIBDIR=graphviz
#cgo CFLAGS: -Icdt
#cgo CFLAGS: -Icommon
#cgo CFLAGS: -Igvc
#cgo CFLAGS: -Ipathplan
#cgo CFLAGS: -Icgraph
#cgo CFLAGS: -Ifdpgen
#cgo CFLAGS: -Isfdpgen
#cgo CFLAGS: -Ixdot
#cgo CFLAGS: -Ilabel
#cgo CFLAGS: -Ipack
#cgo CFLAGS: -Iortho
#cgo CFLAGS: -Iosage
#cgo CFLAGS: -Ineatogen
#cgo CFLAGS: -Isparse
#cgo CFLAGS: -Icircogen
#cgo CFLAGS: -Irbtree
#cgo CFLAGS: -Ipatchwork
#cgo CFLAGS: -Itwopigen
#cgo CFLAGS: -I../
#cgo CFLAGS: -I../libltdl
#cgo CFLAGS: -Wno-unused-result -Wno-format
#include "cdt.h"

extern void *call_searchf(Dtsearch_f searchf, Dt_t *a0, void *a1, int a2);
extern void *call_memoryf(Dtmemory_f memoryf, Dt_t *a0, void *a1, size_t a2, Dtdisc_t *a3);
extern void *call_makef(Dtmake_f makef, Dt_t *a0, void *a1, Dtdisc_t *a2);
extern int call_comparf(Dtcompar_f comparf, Dt_t *a0, void *a1, void *a2, Dtdisc_t *a3);
extern void call_freef(Dtfree_f freef, Dt_t *a0, void *a1, Dtdisc_t *a2);
extern unsigned int call_hashf(Dthash_f hashf, Dt_t *a0, void *a1, Dtdisc_t *a2);
extern int call_eventf(Dtevent_f eventf, Dt_t *a0, int a1, void *a2, Dtdisc_t *a3);
extern int call_dtwalk(Dt_t *a0, void *a1);

*/
import "C"
import (
	"reflect"
	"unsafe"
)

type Dtlink struct {
	c *C.Dtlink_t
}

type Dthold struct {
	c *C.Dthold_t
}

type Dtdisc struct {
	c        *C.Dtdisc_t
	makef    Dtmake
	freef    Dtfree
	comparef Dtcompare
	hashf    Dthash
	memoryf  Dtmemory
	eventf   Dtevent
}

type Dtmethod struct {
	c      *C.Dtmethod_t
	search Dtsearch
}

type Dtdata struct {
	c *C.Dtdata_t
}

type Dict struct {
	c      *C.Dict_t
	search Dtsearch
	memory Dtmemory
}

type Dtstat struct {
	c *C.Dtstat_t
}

type Dtsearch func(*Dict, unsafe.Pointer, int) unsafe.Pointer
type Dtmake func(*Dict, unsafe.Pointer, *Dtdisc) unsafe.Pointer
type Dtmemory func(*Dict, unsafe.Pointer, uint, *Dtdisc) unsafe.Pointer
type Dtfree func(*Dict, unsafe.Pointer, *Dtdisc)
type Dtcompare func(*Dict, unsafe.Pointer, unsafe.Pointer, *Dtdisc) int
type Dthash func(*Dict, unsafe.Pointer, *Dtdisc) uint
type Dtevent func(*Dict, int, unsafe.Pointer, *Dtdisc) int

func ToDtlink(c *C.Dtlink_t) *Dtlink {
	if c == nil {
		return nil
	}
	return &Dtlink{c: c}
}

func (g *Dtlink) C() *C.Dtlink_t {
	if g == nil {
		return nil
	}
	return g.c
}

func (g *Dtlink) Right() *Dtlink {
	return ToDtlink(g.c.right)
}

func (g *Dtlink) SetRight(v *Dtlink) {
	g.c.right = v.c
}

func (g *Dtlink) Hash() uint {
	hash := (*C.uint)(unsafe.Pointer(&g.c.hl))
	return uint(*hash)
}

func (g *Dtlink) SetHash(v uint) {
	hash := (*C.uint)(unsafe.Pointer(&g.c.hl))
	*hash = C.uint(v)
}

func (g *Dtlink) Left() *Dtlink {
	link := (*C.Dtlink_t)(unsafe.Pointer(&g.c.hl))
	return ToDtlink(link)
}

func (g *Dtlink) SetLeft(v *Dtlink) {
	g.c.hl = *(*[8]byte)(unsafe.Pointer(v.c))
}

func ToDthold(c *C.Dthold_t) *Dthold {
	if c == nil {
		return nil
	}
	return &Dthold{c: c}
}

func (g *Dthold) C() *C.Dthold_t {
	if g == nil {
		return nil
	}
	return g.c
}

func (g *Dthold) Hdr() *Dtlink {
	return ToDtlink(&g.c.hdr)
}

func (g *Dthold) SetHdr(v *Dtlink) {
	c := v.c
	if c == nil {
		return
	}
	g.c.hdr = *c
}

func (g *Dthold) Obj() unsafe.Pointer {
	return g.c.obj
}

func (g *Dthold) SetObj(v unsafe.Pointer) {
	g.c.obj = v
}

func ToDtmethod(c *C.Dtmethod_t) *Dtmethod {
	return &Dtmethod{c: c}
}

func (g *Dtmethod) C() *C.Dtmethod_t {
	if g == nil {
		return nil
	}
	return g.c
}

func (g *Dtmethod) Search() Dtsearch {
	if g.search != nil {
		return g.search
	}
	return func(d *Dict, o unsafe.Pointer, opt int) unsafe.Pointer {
		return C.call_searchf(g.c.searchf, d.c, o, C.int(opt))
	}
}

func (g *Dtmethod) SetSearch(v Dtsearch) {
	g.search = v
}

func (g *Dtmethod) Type() int {
	return int(g.c._type)
}

func (g *Dtmethod) SetType(v int) {
	g.c._type = C.int(v)
}

func ToDtdata(c *C.Dtdata_t) *Dtdata {
	if c == nil {
		return nil
	}
	return &Dtdata{c: c}
}

func (g *Dtdata) C() *C.Dtdata_t {
	if g == nil {
		return nil
	}
	return g.c
}

func (g *Dtdata) Type() int {
	return int(g.c._type)
}

func (g *Dtdata) SetType(v int) {
	g.c._type = C.int(v)
}

func (g *Dtdata) Here() *Dtlink {
	return ToDtlink(g.c.here)
}

func (g *Dtdata) SetHere(v *Dtlink) {
	g.c.here = v.c
}

func (g *Dtdata) Htab() []*Dtlink {
	var htab []*Dtlink
	p := (*reflect.SliceHeader)(unsafe.Pointer(&htab))
	p.Cap = int(g.c.ntab)
	p.Len = int(g.c.ntab)
	p.Data = uintptr(unsafe.Pointer(&g.c.hh))
	return htab
}

func (g *Dtdata) SetHtab(v []*Dtlink) {
	header := (*reflect.SliceHeader)(unsafe.Pointer(&v))
	g.c.hh = *(*[8]byte)(unsafe.Pointer(header.Data))
}

func (g *Dtdata) Head() *Dtlink {
	link := (*C.Dtlink_t)(unsafe.Pointer(&g.c.hh))
	return ToDtlink(link)
}

func (g *Dtdata) SetHead(v *Dtlink) {
	g.c.hh = *(*[8]byte)(unsafe.Pointer(v.c))
}

func (g *Dtdata) Ntab() int {
	return int(g.c.ntab)
}

func (g *Dtdata) SetNtab(v int) {
	g.c.ntab = C.int(v)
}

func (g *Dtdata) Size() int {
	return int(g.c.size)
}

func (g *Dtdata) SetSize(v int) {
	g.c.size = C.int(v)
}

func (g *Dtdata) Loop() int {
	return int(g.c.loop)
}

func (g *Dtdata) SetLoop(v int) {
	g.c.loop = C.int(v)
}

func (g *Dtdata) Minp() int {
	return int(g.c.minp)
}

func (g *Dtdata) SetMinp(v int) {
	g.c.minp = C.int(v)
}

func ToDtdisc(c *C.Dtdisc_t) *Dtdisc {
	if c == nil {
		return nil
	}
	return &Dtdisc{c: c}
}

func (g *Dtdisc) C() *C.Dtdisc_t {
	if g == nil {
		return nil
	}
	return g.c
}

func (g *Dtdisc) Key() int {
	return int(g.c.key)
}

func (g *Dtdisc) SetKey(v int) {
	g.c.key = C.int(v)
}

func (g *Dtdisc) Size() int {
	return int(g.c.size)
}

func (g *Dtdisc) SetSize(v int) {
	g.c.size = C.int(v)
}

func (g *Dtdisc) Link() int {
	return int(g.c.link)
}

func (g *Dtdisc) SetLink(v int) {
	g.c.link = C.int(v)
}

func (g *Dtdisc) Make() Dtmake {
	if g.makef != nil {
		return g.makef
	}
	return func(a0 *Dict, a1 unsafe.Pointer, a2 *Dtdisc) unsafe.Pointer {
		return C.call_makef(g.c.makef, a0.c, a1, a2.c)
	}
}

func (g *Dtdisc) Memory() Dtmemory {
	if g.memoryf != nil {
		return g.memoryf
	}
	return func(a0 *Dict, a1 unsafe.Pointer, a2 uint, a3 *Dtdisc) unsafe.Pointer {
		return C.call_memoryf(g.c.memoryf, a0.c, a1, C.ulong(a2), a3.c)
	}
}

func (g *Dtdisc) Event() Dtevent {
	if g.eventf != nil {
		return g.eventf
	}
	return func(a0 *Dict, a1 int, a2 unsafe.Pointer, a3 *Dtdisc) int {
		return int(C.call_eventf(g.c.eventf, a0.c, C.int(a1), a2, a3.c))
	}
}

func (g *Dtdisc) Free() Dtfree {
	if g.freef != nil {
		return g.freef
	}
	return func(a0 *Dict, a1 unsafe.Pointer, a2 *Dtdisc) {
		C.call_freef(g.c.freef, a0.c, a1, a2.c)
	}
}

func (g *Dtdisc) Compare() Dtcompare {
	if g.comparef != nil {
		return g.comparef
	}
	return func(a0 *Dict, a1 unsafe.Pointer, a2 unsafe.Pointer, a3 *Dtdisc) int {
		return int(C.call_comparf(g.c.comparf, a0.c, a1, a2, a3.c))
	}
}

func (g *Dtdisc) Hash() Dthash {
	if g.hashf != nil {
		return g.hashf
	}
	return func(a0 *Dict, a1 unsafe.Pointer, a2 *Dtdisc) uint {
		return uint(C.call_hashf(g.c.hashf, a0.c, a1, a2.c))
	}
}

func (g *Dict) C() *C.Dict_t {
	if g == nil {
		return nil
	}
	return g.c
}

func (g *Dict) Search() Dtsearch {
	if g.search != nil {
		return g.search
	}
	return func(d *Dict, o unsafe.Pointer, opt int) unsafe.Pointer {
		return C.call_searchf(g.c.searchf, d.c, o, C.int(opt))
	}
}

func (g *Dict) Disc() *Dtdisc {
	return ToDtdisc(g.c.disc)
}

func (g *Dict) SetDisc(v *Dtdisc) {
	g.c.disc = v.c
}

func (g *Dict) Data() *Dtdata {
	return ToDtdata(g.c.data)
}

func (g *Dict) SetData(v *Dtdata) {
	g.c.data = v.c
}

func (g *Dict) Memory() Dtmemory {
	if g.memory != nil {
		return g.memory
	}
	return func(a0 *Dict, a1 unsafe.Pointer, a2 uint, a3 *Dtdisc) unsafe.Pointer {
		return C.call_memoryf(g.c.memoryf, a0.c, a1, C.ulong(a2), a3.c)
	}
}

func (g *Dict) Meth() *Dtmethod {
	return ToDtmethod(g.c.meth)
}

func (g *Dict) SetMeth(v *Dtmethod) {
	g.c.meth = v.c
}

func (g *Dict) Type() int {
	return int(g.c._type)
}

func (g *Dict) SetType(v int) {
	g.c._type = C.int(v)
}

func (g *Dict) Nview() int {
	return int(g.c.nview)
}

func (g *Dict) SetNview(v int) {
	g.c.nview = C.int(v)
}

func (g *Dict) View() *Dict {
	return ToDict(g.c.view)
}

func (g *Dict) SetView(v *Dict) {
	g.c.view = v.c
}

func (g *Dict) Walk() *Dict {
	return ToDict(g.c.walk)
}

func (g *Dict) SetWalk(v *Dict) {
	g.c.walk = v.c
}

func (g *Dict) User() unsafe.Pointer {
	return g.c.user
}

func (g *Dict) SetUser(v unsafe.Pointer) {
	g.c.user = v
}

func ToDict(c *C.Dict_t) *Dict {
	if c == nil {
		return nil
	}
	return &Dict{c: c}
}

func ToDtstat(c *C.Dtstat_t) *Dtstat {
	if c == nil {
		return nil
	}
	return &Dtstat{c: c}
}

func (g *Dtstat) C() *C.Dtstat_t {
	if g == nil {
		return nil
	}
	return g.c
}

func (g *Dtstat) DtMeth() int {
	return int(g.c.dt_meth)
}

func (g *Dtstat) SetDtMeth(v int) {
	g.c.dt_meth = C.int(v)
}

func (g *Dtstat) DtSize() int {
	return int(g.c.dt_size)
}

func (g *Dtstat) DtSetSize(v int) {
	g.c.dt_size = C.int(v)
}

func (g *Dtstat) DtN() int {
	return int(g.c.dt_n)
}

func (g *Dtstat) SetDtN(v int) {
	g.c.dt_n = C.int(v)
}

func (g *Dtstat) DtMax() int {
	return int(g.c.dt_max)
}

func (g *Dtstat) SetDtMax(v int) {
	g.c.dt_max = C.int(v)
}

func (g *Dtstat) DtCount() []int {
	var count []int
	p := (*reflect.SliceHeader)(unsafe.Pointer(&count))
	p.Cap = int(g.c.dt_size)
	p.Len = int(g.c.dt_size)
	p.Data = uintptr(unsafe.Pointer(g.c.dt_count))
	return count
}

func (g *Dtstat) SetDtCount(v []int) {
	header := (*reflect.SliceHeader)(unsafe.Pointer(&v))
	g.c.dt_count = (*C.int)(unsafe.Pointer(header.Data))
}

func Dtopen(a0 *Dtdisc, a1 *Dtmethod) *Dict {
	return ToDict(C.dtopen(a0.c, a1.c))
}

func Dtclose(a0 *Dict) int {
	return int(C.dtclose(a0.c))
}

func Dtview(a0 *Dict, a1 *Dict) *Dict {
	return ToDict(C.dtview(a0.c, a1.c))
}

func Dtdiscf(a0 *Dict, a1 *Dtdisc, a2 int) *Dtdisc {
	return ToDtdisc(C.dtdisc(a0.c, a1.c, C.int(a2)))
}

func Dtmethodf(a0 *Dict, a1 *Dtmethod) *Dtmethod {
	return ToDtmethod(C.dtmethod(a0.c, a1.c))
}

func Dtflatten(a0 *Dict) *Dtlink {
	return ToDtlink(C.dtflatten(a0.c))
}

func Dtextract(a0 *Dict) *Dtlink {
	return ToDtlink(C.dtextract(a0.c))
}

func Dtrestore(a0 *Dict, a1 *Dtlink) int {
	return int(C.dtrestore(a0.c, a1.c))
}

func Dttreeset(a0 *Dict, a1 int, a2 int) int {
	return int(C.dttreeset(a0.c, C.int(a1), C.int(a2)))
}

//export GoDtwalkCallback
func GoDtwalkCallback(a0 *C.Dict_t, a1 unsafe.Pointer, a2 unsafe.Pointer) int {
	callback := *(*func(a0 *Dict, a1 unsafe.Pointer, a2 unsafe.Pointer) int)(a0.user)
	return callback(ToDict(a0), a1, a2)
}

func Dtwalk(a0 *Dict, a1 func(a0 *Dict, a1 unsafe.Pointer, a2 unsafe.Pointer) int, a2 unsafe.Pointer) int {
	a0.SetUser(unsafe.Pointer(&a1))
	return int(C.call_dtwalk(a0.c, a2))
}

func Dtrenew(a0 *Dict, a1 unsafe.Pointer) unsafe.Pointer {
	return C.dtrenew(a0.c, a1)
}

func Dtsize(a0 *Dict) int {
	return int(C.dtsize(a0.c))
}

func Dtstatf(a0 *Dict, a1 *Dtstat, a2 int) int {
	return int(C.dtstat(a0.c, a1.c, C.int(a2)))
}

func Dtstrhash(a0 uint, a1 unsafe.Pointer, a2 int) uint {
	return uint(C.dtstrhash(C.uint(a0), a1, C.int(a2)))
}
