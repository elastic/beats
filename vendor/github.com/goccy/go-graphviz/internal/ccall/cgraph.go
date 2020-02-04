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
#include "cgraph.h"
#include <stdlib.h>

void seterr(char *msg)
{
  agerr(AGERR, msg);
}
*/
import "C"
import (
	"errors"
	"unsafe"
)

var (
	Agdirected         = ToAgdesc(&C.Agdirected)
	Agstrictdirected   = ToAgdesc(&C.Agstrictdirected)
	Agundirected       = ToAgdesc(&C.Agundirected)
	Agstrictundirected = ToAgdesc(&C.Agstrictundirected)
)

type Agrec struct {
	c *C.Agrec_t
}

func (g *Agrec) C() *C.Agrec_t {
	if g == nil {
		return nil
	}
	return g.c
}

func ToAgrec(c *C.Agrec_t) *Agrec {
	if c == nil {
		return nil
	}
	return &Agrec{c: c}
}

func (g *Agrec) Name() string {
	return C.GoString(g.c.name)
}

func (g *Agrec) SetName(v string) {
	g.c.name = C.CString(v)
}

func (g *Agrec) Next() *Agrec {
	return ToAgrec(g.c.next)
}

func (g *Agrec) SetNext(v *Agrec) {
	g.c.next = v.c.next
}

type Agtag struct {
	c *C.Agtag_t
}

func ToAgtag(c *C.Agtag_t) *Agtag {
	if c == nil {
		return nil
	}
	return &Agtag{c: c}
}

func (g *Agtag) C() *C.Agtag_t {
	if g == nil {
		return nil
	}
	return g.c
}

func (g *Agtag) ID() uint64 {
	return uint64(g.c.id)
}

func (g *Agtag) SetID(v uint64) {
	g.c.id = C.IDTYPE(v)
}

type Agobj struct {
	c *C.Agobj_t
}

func ToAgobj(c *C.Agobj_t) *Agobj {
	if c == nil {
		return nil
	}
	return &Agobj{c: c}
}

func (g *Agobj) C() *C.Agobj_t {
	if g == nil {
		return nil
	}
	return g.c
}

func (g *Agobj) Tag() *Agtag {
	return ToAgtag(&g.c.tag)
}

func (g *Agobj) SetTag(v *Agtag) {
	c := v.c
	if c == nil {
		return
	}
	g.c.tag = *c
}

func (g *Agobj) Data() *Agrec {
	return ToAgrec(g.c.data)
}

func (g *Agobj) SetData(v *Agrec) {
	g.c.data = v.c
}

type Agsubnode struct {
	c *C.Agsubnode_t
}

func ToAgsubnode(c *C.Agsubnode_t) *Agsubnode {
	if c == nil {
		return nil
	}
	return &Agsubnode{c: c}
}

func (g *Agsubnode) C() *C.Agsubnode_t {
	if g == nil {
		return nil
	}
	return g.c
}

func (g *Agsubnode) SeqLink() *Dtlink {
	return ToDtlink(&g.c.seq_link)
}

func (g *Agsubnode) SetSeqLink(v *Dtlink) {
	c := v.c
	if c == nil {
		return
	}
	g.c.seq_link = *c
}

func (g *Agsubnode) IDLink() *Dtlink {
	return ToDtlink(&g.c.id_link)
}

func (g *Agsubnode) SetIDLink(v *Dtlink) {
	c := v.c
	if c == nil {
		return
	}
	g.c.id_link = *c
}

func (g *Agsubnode) Node() *Agnode {
	return ToAgnode(g.c.node)
}

func (g *Agsubnode) SetNode(v *Agnode) {
	g.c.node = v.c
}

func (g *Agsubnode) InID() *Dtlink {
	return ToDtlink(g.c.in_id)
}

func (g *Agsubnode) SetInID(v *Dtlink) {
	g.c.in_id = v.c
}

func (g *Agsubnode) OutID() *Dtlink {
	return ToDtlink(g.c.out_id)
}

func (g *Agsubnode) SetOutID(v *Dtlink) {
	g.c.out_id = v.c
}

func (g *Agsubnode) InSeq() *Dtlink {
	return ToDtlink(g.c.in_seq)
}

func (g *Agsubnode) SetInSeq(v *Dtlink) {
	g.c.in_seq = v.c
}

func (g *Agsubnode) OutSeq() *Dtlink {
	return ToDtlink(g.c.out_seq)
}

func (g *Agsubnode) SetOutSeq(v *Dtlink) {
	g.c.out_seq = v.c
}

type Agnode struct {
	c *C.Agnode_t
}

func ToAgnode(c *C.Agnode_t) *Agnode {
	if c == nil {
		return nil
	}
	return &Agnode{c: c}
}

func (g *Agnode) C() *C.Agnode_t {
	if g == nil {
		return nil
	}
	return g.c
}

func (g *Agnode) Base() *Agobj {
	return ToAgobj(&g.c.base)
}

func (g *Agnode) SetBase(v *Agobj) {
	c := v.c
	if c == nil {
		return
	}
	g.c.base = *c
}

func (g *Agnode) Root() *Agraph {
	return ToAgraph(g.c.root)
}

func (g *Agnode) SetRoot(v *Agraph) {
	g.c.root = v.c
}

func (g *Agnode) Mainsub() *Agsubnode {
	return ToAgsubnode(&g.c.mainsub)
}

func (g *Agnode) SetMainsub(v *Agsubnode) {
	c := v.c
	if c == nil {
		return
	}
	g.c.mainsub = *c
}

type Agedge struct {
	c *C.Agedge_t
}

func ToAgedge(c *C.Agedge_t) *Agedge {
	if c == nil {
		return nil
	}
	return &Agedge{c: c}
}

func (g *Agedge) C() *C.Agedge_t {
	if g == nil {
		return nil
	}
	return g.c
}

func (g *Agedge) Base() *Agobj {
	return ToAgobj(&g.c.base)
}

func (g *Agedge) SetBase(v *Agobj) {
	c := v.c
	if c == nil {
		return
	}
	g.c.base = *c
}

func (g *Agedge) IDLink() *Dtlink {
	return ToDtlink(&g.c.id_link)
}

func (g *Agedge) SetIDLink(v *Dtlink) {
	c := v.c
	if c == nil {
		return
	}
	g.c.id_link = *c
}

func (g *Agedge) SeqLink() *Dtlink {
	return ToDtlink(&g.c.seq_link)
}

func (g *Agedge) SetSeqLink(v *Dtlink) {
	c := v.c
	if c == nil {
		return
	}
	g.c.seq_link = *c
}

func (g *Agedge) Node() *Agnode {
	return ToAgnode(g.c.node)
}

func (g *Agedge) SetNode(v *Agnode) {
	g.c.node = v.c
}

type Agedgepair struct {
	c *C.Agedgepair_t
}

func ToAgedgepair(c *C.Agedgepair_t) *Agedgepair {
	if c == nil {
		return nil
	}
	return &Agedgepair{c: c}
}

func (g *Agedgepair) C() *C.Agedgepair_t {
	if g == nil {
		return nil
	}
	return g.c
}

func (g *Agedgepair) Out() *Agedge {
	return ToAgedge(&g.c.out)
}

func (g *Agedgepair) SetOut(v *Agedge) {
	c := v.c
	if c == nil {
		return
	}
	g.c.out = *c
}

func (g *Agedgepair) In() *Agedge {
	return ToAgedge(&g.c.in)
}

func (g *Agedgepair) SetIn(v *Agedge) {
	c := v.c
	if c == nil {
		return
	}
	g.c.in = *c
}

type Agdesc struct {
	c *C.Agdesc_t
}

func ToAgdesc(c *C.Agdesc_t) *Agdesc {
	if c == nil {
		return nil
	}
	return &Agdesc{c: c}
}

func (g *Agdesc) C() *C.Agdesc_t {
	if g == nil {
		return nil
	}
	return g.c
}

type Agdisc struct {
	c *C.Agdisc_t
}

func ToAgdisc(c *C.Agdisc_t) *Agdisc {
	if c == nil {
		return nil
	}
	return &Agdisc{c: c}
}

func (g *Agdisc) C() *C.Agdisc_t {
	if g == nil {
		return nil
	}
	return g.c
}

type Agcbdisc struct {
	c *C.Agcbdisc_t
}

func ToAgcbdisc(c *C.Agcbdisc_t) *Agcbdisc {
	if c == nil {
		return nil
	}
	return &Agcbdisc{c: c}
}

type Agcbstack struct {
	c *C.Agcbstack_t
}

func ToAgcbstack(c *C.Agcbstack_t) *Agcbstack {
	if c == nil {
		return nil
	}
	return &Agcbstack{c: c}
}

func (g *Agcbstack) Prev() *Agcbstack {
	return ToAgcbstack(g.c.prev)
}

type Agdstate struct {
	c *C.Agdstate_t
}

func ToAgdstate(c *C.Agdstate_t) *Agdstate {
	if c == nil {
		return nil
	}
	return &Agdstate{c: c}
}

func (g *Agdstate) Mem() unsafe.Pointer {
	return g.c.mem
}

func (g *Agdstate) ID() unsafe.Pointer {
	return g.c.id
}

type Agclos struct {
	c *C.Agclos_t
}

func ToAgclos(c *C.Agclos_t) *Agclos {
	if c == nil {
		return nil
	}
	return &Agclos{c: c}
}

func (g *Agclos) Disc() *Agdisc {
	return ToAgdisc(&g.c.disc)
}

func (g *Agclos) SetDisc(v *Agdisc) {
	c := v.c
	if c == nil {
		return
	}
	g.c.disc = *c
}

func (g *Agclos) State() *Agdstate {
	return ToAgdstate(&g.c.state)
}

func (g *Agclos) SetState(v *Agdstate) {
	c := v.c
	if c == nil {
		return
	}
	g.c.state = *c
}

func (g *Agclos) Strdict() *Dict {
	return ToDict(g.c.strdict)
}

func (g *Agclos) SetStrdict(v *Dict) {
	g.c.strdict = v.c
}

func (g *Agclos) Seq() [3]uint64 {
	seq := [3]uint64{}
	seq[0] = uint64(g.c.seq[0])
	seq[1] = uint64(g.c.seq[1])
	seq[2] = uint64(g.c.seq[2])
	return seq
}

func (g *Agclos) SetSeq(v []uint64) {
	g.c.seq[0] = C.uint64_t(v[0])
	g.c.seq[1] = C.uint64_t(v[1])
	g.c.seq[2] = C.uint64_t(v[2])
}

func (g *Agclos) Cb() *Agcbstack {
	return ToAgcbstack(g.c.cb)
}

func (g *Agclos) SetCb(v *Agcbstack) {
	g.c.cb = v.c
}

func (g *Agclos) CallbacksEnabled() bool {
	return uint(g.c.callbacks_enabled) == 1
}

func (g *Agclos) SetCallbacksEnabled(v bool) {
	if v {
		g.c.callbacks_enabled = 1
	} else {
		g.c.callbacks_enabled = 0
	}
}

func (g *Agclos) LookupByName() [3]*Dict {
	v := [3]*Dict{}
	v[0] = ToDict(g.c.lookup_by_name[0])
	v[1] = ToDict(g.c.lookup_by_name[1])
	v[2] = ToDict(g.c.lookup_by_name[2])
	return v
}

func (g *Agclos) LookupByID() [3]*Dict {
	v := [3]*Dict{}
	v[0] = ToDict(g.c.lookup_by_id[0])
	v[1] = ToDict(g.c.lookup_by_id[1])
	v[2] = ToDict(g.c.lookup_by_id[2])
	return v
}

type Agraph struct {
	c *C.Agraph_t
}

func ToAgraph(c *C.Agraph_t) *Agraph {
	if c == nil {
		return nil
	}
	return &Agraph{c: c}
}

func (g *Agraph) C() *C.Agraph_t {
	if g == nil {
		return nil
	}
	return g.c
}

func (g *Agraph) Base() *Agobj {
	return ToAgobj(&g.c.base)
}

func (g *Agraph) SetBase(v *Agobj) {
	c := v.c
	if c == nil {
		return
	}
	g.c.base = *c
}

func (g *Agraph) Desc() *Agdesc {
	return ToAgdesc(&g.c.desc)
}

func (g *Agraph) SetDesc(v *Agdesc) {
	c := v.c
	if c == nil {
		return
	}
	g.c.desc = *c
}

func (g *Agraph) Link() *Dtlink {
	return ToDtlink(&g.c.link)
}

func (g *Agraph) SetLink(v *Dtlink) {
	c := v.c
	if c == nil {
		return
	}
	g.c.link = *c
}

func (g *Agraph) NSeq() *Dict {
	return ToDict(g.c.n_seq)
}

func (g *Agraph) SetNSeq(v *Dict) {
	g.c.n_seq = v.c
}

func (g *Agraph) NID() *Dict {
	return ToDict(g.c.n_id)
}

func (g *Agraph) SetNID(v *Dict) {
	g.c.n_id = v.c
}

func (g *Agraph) ESeq() *Dict {
	return ToDict(g.c.e_seq)
}

func (g *Agraph) SetESeq(v *Dict) {
	g.c.e_seq = v.c
}

func (g *Agraph) EID() *Dict {
	return ToDict(g.c.e_id)
}

func (g *Agraph) SetEID(v *Dict) {
	g.c.e_id = v.c
}

func (g *Agraph) GDict() *Dict {
	return ToDict(g.c.g_dict)
}

func (g *Agraph) SetGDict(v *Dict) {
	g.c.g_dict = v.c
}

func (g *Agraph) Parent() *Agraph {
	return ToAgraph(g.c.parent)
}

func (g *Agraph) SetParent(v *Agraph) {
	g.c.parent = v.c
}

func (g *Agraph) Root() *Agraph {
	return ToAgraph(g.c.root)
}

func (g *Agraph) SetRoot(v *Agraph) {
	g.c.root = v.c
}

func (g *Agraph) Clos() *Agclos {
	return ToAgclos(g.c.clos)
}

func (g *Agraph) SetClos(v *Agclos) {
	g.c.clos = v.c
}

type Agattr struct {
	c *C.Agattr_t
}

func ToAgattr(c *C.Agattr_t) *Agattr {
	if c == nil {
		return nil
	}
	return &Agattr{c: c}
}

func (g *Agattr) C() *C.Agattr_t {
	if g == nil {
		return nil
	}
	return g.c
}

func (g *Agattr) H() *Agrec {
	return ToAgrec(&g.c.h)
}

func (g *Agattr) SetH(v *Agrec) {
	c := v.c
	if c == nil {
		return
	}
	g.c.h = *c
}

func (g *Agattr) Dict() *Dict {
	return ToDict(g.c.dict)
}

func (g *Agattr) SetDict(v *Dict) {
	g.c.dict = v.c
}

func (g *Agattr) Str() []string {
	v := []string{}
	/*
		i := 0
		for {
			if g.c.str[i] == nil {
				break
			}
			v = append(v, C.GoString(g.c.str[i]))
		}
	*/
	return v
}

func (g *Agattr) SetStr(v []string) {

}

type Agsym struct {
	c *C.Agsym_t
}

func ToAgsym(c *C.Agsym_t) *Agsym {
	if c == nil {
		return nil
	}
	return &Agsym{c: c}
}

func (g *Agsym) C() *C.Agsym_t {
	if g == nil {
		return nil
	}
	return g.c
}

func (g *Agsym) Link() *Dtlink {
	return ToDtlink(&g.c.link)
}

func (g *Agsym) SetLink(v *Dtlink) {
	c := v.c
	if c == nil {
		return
	}
	g.c.link = *c
}

func (g *Agsym) Name() string {
	return C.GoString(g.c.name)
}

func (g *Agsym) SetName(v string) {
	g.c.name = C.CString(v)
}

func (g *Agsym) Defval() string {
	return C.GoString(g.c.defval)
}

func (g *Agsym) SetDefval(v string) {
	g.c.defval = C.CString(v)
}

func (g *Agsym) ID() int {
	return int(g.c.id)
}

func (g *Agsym) SetID(v int) {
	g.c.id = C.int(v)
}

func (g *Agsym) Kind() uint {
	return uint(g.c.kind)
}

func (g *Agsym) SetKind(v uint) {
	g.c.kind = C.uchar(v)
}

func (g *Agsym) Fixed() uint {
	return uint(g.c.fixed)
}

func (g *Agsym) SetFixed(v uint) {
	g.c.fixed = C.uchar(v)
}

func (g *Agsym) Print() uint {
	return uint(g.c.print)
}

func (g *Agsym) SetPrint(v uint) {
	g.c.print = C.uchar(v)
}

type Agdatadict struct {
	c *C.Agdatadict_t
}

func ToAgdatadict(c *C.Agdatadict_t) *Agdatadict {
	if c == nil {
		return nil
	}
	return &Agdatadict{c: c}
}

func (g *Agdatadict) H() *Agrec {
	return ToAgrec(&g.c.h)
}

func (g *Agdatadict) SetH(v *Agrec) {
	c := v.c
	if c == nil {
		return
	}
	g.c.h = *c
}

func (g *Agdatadict) DictN() *Dict {
	return ToDict(g.c.dict.n)
}

func (g *Agdatadict) SetDictN(v *Dict) {
	g.c.dict.n = v.c
}

func (g *Agdatadict) DictE() *Dict {
	return ToDict(g.c.dict.e)
}

func (g *Agdatadict) SetDictE(v *Dict) {
	g.c.dict.e = v.c
}

func (g *Agdatadict) DictG() *Dict {
	return ToDict(g.c.dict.g)
}

func (g *Agdatadict) SetDictG(v *Dict) {
	g.c.dict.g = v.c
}

func Agpushdisc(g *Agraph, disc *Agcbdisc, state unsafe.Pointer) {
	C.agpushdisc(g.c, disc.c, state)
}

func Agpopdisc(g *Agraph, disc *Agcbdisc) int {
	return int(C.agpopdisc(g.c, disc.c))
}

func Agcallbacks(g *Agraph, flag int) int {
	return int(C.agcallbacks(g.c, C.int(flag)))
}

func Agopen(name string, desc *Agdesc, disc *Agdisc) (*Agraph, error) {
	graph := ToAgraph(C.agopen(C.CString(name), *desc.C(), disc.C()))
	return graph, Aglasterr()
}

func Agclose(g *Agraph) error {
	C.agclose(g.c)
	return Aglasterr()
}

func Agread(ch unsafe.Pointer, disc *Agdisc) (*Agraph, error) {
	graph := ToAgraph(C.agread(ch, disc.c))
	return graph, Aglasterr()
}

func Agmemread(cp string) (*Agraph, error) {
	graph := ToAgraph(C.agmemread(C.CString(cp)))
	return graph, Aglasterr()
}

func Agsetfile(file string) {
	C.agsetfile(C.CString(file))
}

func Agwrite(g *Agraph, ch unsafe.Pointer) error {
	C.agwrite(g.c, ch)
	return Aglasterr()
}

func Agisdirected(g *Agraph) bool {
	return C.agisdirected(g.c) == 1
}

func Agisundirected(g *Agraph) bool {
	return C.agisundirected(g.c) == 1
}

func Agisstrict(g *Agraph) bool {
	return C.agisstrict(g.c) == 1
}

func Agissimple(g *Agraph) bool {
	return C.agissimple(g.c) == 1
}

func Agnodef(g *Agraph, name string, createFlag int) (*Agnode, error) {
	node := ToAgnode(C.agnode(g.c, C.CString(name), C.int(createFlag)))
	return node, Aglasterr()
}

func Agidnode(g *Agraph, id uint64, createFlag int) (*Agnode, error) {
	node := ToAgnode(C.agidnode(g.c, C.IDTYPE(id), C.int(createFlag)))
	return node, Aglasterr()
}

func Agsubnodef(g *Agraph, n *Agnode, createFlag int) (*Agnode, error) {
	node := ToAgnode(C.agsubnode(g.c, n.c, C.int(createFlag)))
	return node, Aglasterr()
}

func Agfstnode(g *Agraph) *Agnode {
	return ToAgnode(C.agfstnode(g.c))
}

func Agnxtnode(g *Agraph, n *Agnode) *Agnode {
	return ToAgnode(C.agnxtnode(g.c, n.c))
}

func Aglstnode(g *Agraph) *Agnode {
	return ToAgnode(C.aglstnode(g.c))
}

func Agprvnode(g *Agraph, n *Agnode) *Agnode {
	return ToAgnode(C.agprvnode(g.c, n.c))
}

func Agsubrep(g *Agraph, n *Agnode) *Agsubnode {
	return ToAgsubnode(C.agsubrep(g.c, n.c))
}

func Agnodebefore(u *Agnode, v *Agnode) error {
	C.agnodebefore(u.c, v.c)
	return Aglasterr()
}

func Agedgef(g *Agraph, t *Agnode, h *Agnode, name string, createFlag int) (*Agedge, error) {
	edge := ToAgedge(C.agedge(g.c, t.c, h.c, C.CString(name), C.int(createFlag)))
	return edge, Aglasterr()
}

func Agidedge(g *Agraph, t *Agnode, h *Agnode, id uint64, createFlag int) (*Agedge, error) {
	edge := ToAgedge(C.agidedge(g.c, t.c, h.c, C.IDTYPE(id), C.int(createFlag)))
	return edge, Aglasterr()
}

func Agsubedge(g *Agraph, e *Agedge, createFlag int) (*Agedge, error) {
	edge := ToAgedge(C.agsubedge(g.c, e.c, C.int(createFlag)))
	return edge, Aglasterr()
}

func Agfstin(g *Agraph, n *Agnode) *Agedge {
	return ToAgedge(C.agfstin(g.c, n.c))
}

func Agnxtin(g *Agraph, n *Agedge) *Agedge {
	return ToAgedge(C.agnxtin(g.c, n.c))
}

func Agfstout(g *Agraph, n *Agnode) *Agedge {
	return ToAgedge(C.agfstout(g.c, n.c))
}

func Agnxtout(g *Agraph, e *Agedge) *Agedge {
	return ToAgedge(C.agnxtout(g.c, e.c))
}

func Agfstedge(g *Agraph, n *Agnode) *Agedge {
	return ToAgedge(C.agfstedge(g.c, n.c))
}

func Agnxtedge(g *Agraph, e *Agedge, n *Agnode) *Agedge {
	return ToAgedge(C.agnxtedge(g.c, e.c, n.c))
}

func Agcontains(g *Agraph, p unsafe.Pointer) bool {
	return C.agcontains(g.c, p) == 1
}

func Agnameof(p unsafe.Pointer) string {
	return C.GoString(C.agnameof(p))
}

func AgrelabelNode(n *Agnode, newname string) error {
	C.agrelabel_node(n.c, C.CString(newname))
	return Aglasterr()
}

func Agdelete(g *Agraph, obj unsafe.Pointer) error {
	C.agdelete(g.c, obj)
	return Aglasterr()
}

func Agdelsubg(g *Agraph, sub *Agraph) int32 {
	return int32(C.agdelsubg(g.c, sub.c))
}

func Agdelnode(g *Agraph, argN *Agnode) int {
	return int(C.agdelnode(g.c, argN.c))
}

func Agdeledge(g *Agraph, argE *Agedge) int {
	return int(C.agdeledge(g.c, argE.c))
}

func Agobjkind(obj unsafe.Pointer) int {
	return int(C.agobjkind(obj))
}

func Agstrdup(g *Agraph, s string) string {
	return C.GoString(C.agstrdup(g.c, C.CString(s)))
}

func AgstrdupHTML(g *Agraph, s string) string {
	return C.GoString(C.agstrdup_html(g.c, C.CString(s)))
}

func Aghtmlstr(s string) int {
	return int(C.aghtmlstr(C.CString(s)))
}

func Agstrbind(g *Agraph, s string) string {
	return C.GoString(C.agstrbind(g.c, C.CString(s)))
}

func Agstrfree(g *Agraph, s string) int {
	return int(C.agstrfree(g.c, C.CString(s)))
}

func Agcanon(s string, i int) string {
	return C.GoString(C.agcanon(C.CString(s), C.int(i)))
}

func Agstrcanon(a0 string, a1 string) string {
	return C.GoString(C.agstrcanon(C.CString(a0), C.CString(a1)))
}

func AgcanonStr(str string) string {
	return C.GoString(C.agcanonStr(C.CString(str)))
}

func Agattrf(g *Agraph, kind int, name string, value string) *Agsym {
	return ToAgsym(C.agattr(g.c, C.int(kind), C.CString(name), C.CString(value)))
}

func Agattrsym(obj unsafe.Pointer, name string) *Agsym {
	return ToAgsym(C.agattrsym(obj, C.CString(name)))
}

func Agnxtattr(g *Agraph, kind int, attr *Agsym) *Agsym {
	return ToAgsym(C.agnxtattr(g.c, C.int(kind), attr.c))
}

func Agcopyattr(oldobj unsafe.Pointer, newobj unsafe.Pointer) int {
	return int(C.agcopyattr(oldobj, newobj))
}

func Agbindrec(obj unsafe.Pointer, name string, size uint, moveToFront int) unsafe.Pointer {
	return C.agbindrec(obj, C.CString(name), C.uint(size), C.int(moveToFront))
}

func Aggetrec(obj unsafe.Pointer, name string, moveToFront int) *Agrec {
	return ToAgrec(C.aggetrec(obj, C.CString(name), C.int(moveToFront)))
}

func Agdelrec(obj unsafe.Pointer, name string) int {
	return int(C.agdelrec(obj, C.CString(name)))
}

func Aginit(g *Agraph, kind int, recName string, recSize int, moveToFront int) {
	C.aginit(g.c, C.int(kind), C.CString(recName), C.int(recSize), C.int(moveToFront))
}

func Agclean(g *Agraph, kind int, recName string) {
	C.agclean(g.c, C.int(kind), C.CString(recName))
}

func Agget(obj unsafe.Pointer, name string) string {
	return C.GoString(C.agget(obj, C.CString(name)))
}

func Agxget(obj unsafe.Pointer, sym *Agsym) string {
	return C.GoString(C.agxget(obj, sym.c))
}

func Agset(obj unsafe.Pointer, name string, value string) int {
	return int(C.agset(obj, C.CString(name), C.CString(value)))
}

func Agxset(obj unsafe.Pointer, sym *Agsym, value string) int {
	return int(C.agxset(obj, sym.c, C.CString(value)))
}

func Agsafeset(obj unsafe.Pointer, name string, value string, def string) int {
	return int(C.agsafeset(obj, C.CString(name), C.CString(value), C.CString(def)))
}

func Agsubg(g *Agraph, name string, cflag int) *Agraph {
	return ToAgraph(C.agsubg(g.c, C.CString(name), C.int(cflag)))
}

func Agidsubg(g *Agraph, id uint64, cflag int) *Agraph {
	return ToAgraph(C.agidsubg(g.c, C.IDTYPE(id), C.int(cflag)))
}

func Agfstsubg(g *Agraph) *Agraph {
	return ToAgraph(C.agfstsubg(g.c))
}

func Agnxtsubg(subg *Agraph) *Agraph {
	return ToAgraph(C.agnxtsubg(subg.c))
}

func Agparent(g *Agraph) *Agraph {
	return ToAgraph(C.agparent(g.c))
}

func Agnnodes(g *Agraph) int {
	return int(C.agnnodes(g.c))
}

func Agnedges(g *Agraph) int {
	return int(C.agnedges(g.c))
}

func Agnsubg(g *Agraph) int {
	return int(C.agnsubg(g.c))
}

func Agdegree(g *Agraph, n *Agnode, in int, out int) int {
	return int(C.agdegree(g.c, n.c, C.int(in), C.int(out)))
}

func Agcountuniqedges(g *Agraph, n *Agnode, in int, out int) int {
	return int(C.agcountuniqedges(g.c, n.c, C.int(in), C.int(out)))
}

func Agalloc(g *Agraph, size uint) unsafe.Pointer {
	return C.agalloc(g.c, C.ulong(size))
}

func Agfree(g *Agraph, ptr unsafe.Pointer) {
	C.agfree(g.c, ptr)
}

func Agflatten(g *Agraph, flag int) {
	C.agflatten(g.c, C.int(flag))
}

func Aginternalmapclearlocalnames(g *Agraph) {
	C.aginternalmapclearlocalnames(g.c)
}

func Aglasterr() error {
	s := C.aglasterr()
	if s == nil {
		return nil
	}
	v := C.GoString(s)
	C.free(unsafe.Pointer(s))
	return errors.New(v)
}

func Agerr(msg string) {
	s := C.CString(msg)
	C.seterr(s)
	C.free(unsafe.Pointer(s))
}

func init() {
	C.agseterr(C.AGMAX)
}
