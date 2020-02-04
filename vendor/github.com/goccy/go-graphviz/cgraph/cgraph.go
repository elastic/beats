package cgraph

import (
	"io/ioutil"
	"unsafe"

	"github.com/goccy/go-graphviz/cdt"
	"github.com/goccy/go-graphviz/internal/ccall"
)

type Graph struct {
	*ccall.Agraph
}

type Node struct {
	*ccall.Agnode
}

type SubNode struct {
	*ccall.Agsubnode
}

type Edge struct {
	*ccall.Agedge
}

type Desc struct {
	*ccall.Agdesc
}

type Disc struct {
	*ccall.Agdisc
}

// Symbol symbol in one of the above dictionaries
type Symbol struct {
	*ccall.Agsym
}

// Record generic runtime record
type Record struct {
	*ccall.Agrec
}

type Tag struct {
	*ccall.Agtag
}

type Object struct {
	*ccall.Agobj
}

type Clos struct {
	*ccall.Agclos
}

type State struct {
	*ccall.Agdstate
}

type CallbackStack struct {
	*ccall.Agcbstack
}

type Attr struct {
	*ccall.Agattr
}

type DataDict struct {
	*ccall.Agdatadict
}

type IDTYPE uint64

var (
	Directed         = &Desc{Agdesc: ccall.Agdirected}
	StrictDirected   = &Desc{Agdesc: ccall.Agstrictdirected}
	UnDirected       = &Desc{Agdesc: ccall.Agundirected}
	StrictUnDirected = &Desc{Agdesc: ccall.Agstrictundirected}
)

func toGraph(g *ccall.Agraph) *Graph {
	if g == nil {
		return nil
	}
	return &Graph{Agraph: g}
}

func toNode(n *ccall.Agnode) *Node {
	if n == nil {
		return nil
	}
	return &Node{Agnode: n}
}

func toEdge(e *ccall.Agedge) *Edge {
	if e == nil {
		return nil
	}
	return &Edge{Agedge: e}
}

func ParseBytes(bytes []byte) (*Graph, error) {
	graph, err := ccall.Agmemread(string(bytes))
	if err != nil {
		return nil, err
	}
	return toGraph(graph), nil
}

func ParseFile(path string) (*Graph, error) {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	graph, err := ccall.Agmemread(string(file))
	if err != nil {
		return nil, err
	}
	return toGraph(graph), nil
}

func Open(name string, desc *Desc, disc *Disc) (*Graph, error) {
	var (
		agdesc *ccall.Agdesc
		agdisc *ccall.Agdisc
	)
	if desc != nil {
		agdesc = desc.Agdesc
	}
	if disc != nil {
		agdisc = disc.Agdisc
	}
	graph, err := ccall.Agopen(name, agdesc, agdisc)
	if err != nil {
		return nil, err
	}
	return toGraph(graph), nil
}

type OBJECTKIND int

const (
	GRAPH   OBJECTKIND = 0
	NODE    OBJECTKIND = 1
	OUTEDGE OBJECTKIND = 2
	INEDGE  OBJECTKIND = 3
	EDGE    OBJECTKIND = OUTEDGE
)

func ObjectKind(obj *Object) OBJECTKIND {
	return OBJECTKIND(ccall.Agobjkind(unsafe.Pointer(obj.Agobj.C())))
}

func HTMLStr(s string) int {
	return ccall.Aghtmlstr(s)
}

func Canon(s string, i int) string {
	return ccall.Agcanon(s, i)
}

func StrCanon(a0 string, a1 string) string {
	return ccall.Agstrcanon(a0, a1)
}

func CanonStr(str string) string {
	return ccall.AgcanonStr(str)
}

func AttrSym(obj *Object, name string) *Symbol {
	sym := ccall.Agattrsym(unsafe.Pointer(obj.Agobj.C()), name)
	if sym == nil {
		return nil
	}
	return &Symbol{Agsym: sym}
}

func (r *Record) Name() string {
	return r.Agrec.Name()
}

func (r *Record) SetName(v string) {
	r.Agrec.SetName(v)
}

func (r *Record) Next() *Record {
	v := r.Agrec.Next()
	if v == nil {
		return nil
	}
	return &Record{Agrec: v}
}

func (r *Record) SetNext(v *Record) {
	if v == nil || v.Agrec == nil {
		return
	}
	r.Agrec.SetNext(v.Agrec)
}

func (t *Tag) ID() IDTYPE {
	return IDTYPE(t.Agtag.ID())
}

func (t *Tag) SetID(v IDTYPE) {
	t.Agtag.SetID(uint64(v))
}

func (o *Object) Tag() *Tag {
	v := o.Agobj.Tag()
	if v == nil {
		return nil
	}
	return &Tag{Agtag: v}
}

func (o *Object) SetTag(v *Tag) {
	if v == nil || v.Agtag == nil {
		return
	}
	o.Agobj.SetTag(v.Agtag)
}

func (o *Object) Data() *Record {
	v := o.Agobj.Data()
	if v == nil {
		return nil
	}
	return &Record{Agrec: v}
}

func (o *Object) SetData(v *Record) {
	if v == nil || v.Agrec == nil {
		return
	}
	o.Agobj.SetData(v.Agrec)
}

func (o *Object) SafeSet(name, value, def string) int {
	return ccall.Agsafeset(unsafe.Pointer(o.Agobj.C()), name, value, def)
}

func (n *SubNode) SeqLink() *cdt.Link {
	v := n.Agsubnode.SeqLink()
	if v == nil {
		return nil
	}
	return &cdt.Link{Dtlink: v}
}

func (n *SubNode) SetSeqLink(v *cdt.Link) {
	if v == nil || v.Dtlink == nil {
		return
	}
	n.Agsubnode.SetSeqLink(v.Dtlink)
}

func (n *SubNode) IDLink() *cdt.Link {
	v := n.Agsubnode.IDLink()
	if v == nil {
		return nil
	}
	return &cdt.Link{Dtlink: v}
}

func (n *SubNode) SetIDLink(v *cdt.Link) {
	if v == nil || v.Dtlink == nil {
		return
	}
	n.Agsubnode.SetIDLink(v.Dtlink)
}

func (n *SubNode) Node() *Node {
	v := n.Agsubnode.Node()
	if v == nil {
		return nil
	}
	return &Node{Agnode: v}
}

func (n *SubNode) SetNode(v *Node) {
	if v == nil || v.Agnode == nil {
		return
	}
	n.Agsubnode.SetNode(v.Agnode)
}

func (n *SubNode) InID() *cdt.Link {
	v := n.Agsubnode.InID()
	if v == nil {
		return nil
	}
	return &cdt.Link{Dtlink: v}
}

func (n *SubNode) SetInID(v *cdt.Link) {
	if v == nil || v.Dtlink == nil {
		return
	}
	n.Agsubnode.SetInID(v.Dtlink)
}

func (n *SubNode) OutID() *cdt.Link {
	v := n.Agsubnode.OutID()
	if v == nil {
		return nil
	}
	return &cdt.Link{Dtlink: v}
}

func (n *SubNode) SetOutID(v *cdt.Link) {
	if v == nil || v.Dtlink == nil {
		return
	}
	n.Agsubnode.SetOutID(v.Dtlink)
}

func (n *SubNode) InSeq() *cdt.Link {
	v := n.Agsubnode.InSeq()
	if v == nil {
		return nil
	}
	return &cdt.Link{Dtlink: v}
}

func (n *SubNode) SetInSeq(v *cdt.Link) {
	if v == nil || v.Dtlink == nil {
		return
	}
	n.Agsubnode.SetInSeq(v.Dtlink)
}

func (n *SubNode) OutSeq() *cdt.Link {
	v := n.Agsubnode.OutSeq()
	if v == nil {
		return nil
	}
	return &cdt.Link{Dtlink: v}
}

func (n *SubNode) SetOutSeq(v *cdt.Link) {
	if v == nil || v.Dtlink == nil {
		return
	}
	n.Agsubnode.SetOutSeq(v.Dtlink)
}

func (n *Node) Base() *Object {
	v := n.Agnode.Base()
	if v == nil {
		return nil
	}
	return &Object{Agobj: v}
}

func (n *Node) SetBase(v *Object) {
	if v == nil || v.Agobj == nil {
		return
	}
	n.Agnode.SetBase(v.Agobj)
}

func (n *Node) Root() *Graph {
	v := n.Agnode.Root()
	if v == nil {
		return nil
	}
	return &Graph{Agraph: v}
}

func (n *Node) SetRootGraph(v *Graph) {
	if v == nil || v.Agraph == nil {
		return
	}
	n.Agnode.SetRoot(v.Agraph)
}

func (n *Node) MainSub() *SubNode {
	v := n.Agnode.Mainsub()
	if v == nil {
		return nil
	}
	return &SubNode{Agsubnode: v}
}

func (n *Node) SetMainSub(v *SubNode) {
	if v == nil || v.Agsubnode == nil {
		return
	}
	n.Agnode.SetMainsub(v.Agsubnode)
}

func (e *Edge) Base() *Object {
	v := e.Agedge.Base()
	if v == nil {
		return nil
	}
	return &Object{Agobj: v}
}

func (e *Edge) SetBase(v *Object) {
	if v == nil || v.Agobj == nil {
		return
	}
	e.Agedge.SetBase(v.Agobj)
}

func (e *Edge) SeqLink() *cdt.Link {
	v := e.Agedge.SeqLink()
	if v == nil {
		return nil
	}
	return &cdt.Link{Dtlink: v}
}

func (e *Edge) SetSeqLink(v *cdt.Link) {
	if v == nil || v.Dtlink == nil {
		return
	}
	e.Agedge.SetSeqLink(v.Dtlink)
}

func (e *Edge) IDLink() *cdt.Link {
	v := e.Agedge.IDLink()
	if v == nil {
		return nil
	}
	return &cdt.Link{Dtlink: v}
}

func (e *Edge) SetIDLink(v *cdt.Link) {
	if v == nil || v.Dtlink == nil {
		return
	}
	e.Agedge.SetIDLink(v.Dtlink)
}

func (e *Edge) Node() *Node {
	v := e.Agedge.Node()
	if v == nil {
		return nil
	}
	return &Node{Agnode: v}
}

func (e *Edge) SetNode(v *Node) {
	if v == nil || v.Agnode == nil {
		return
	}
	e.Agedge.SetNode(v.Agnode)
}

func (c *Clos) Disc() *Disc {
	v := c.Agclos.Disc()
	if v == nil {
		return nil
	}
	return &Disc{Agdisc: v}
}

func (c *Clos) SetDisc(v *Disc) {
	if v == nil || v.Agdisc == nil {
		return
	}
	c.Agclos.SetDisc(v.Agdisc)
}

func (c *Clos) State() *State {
	v := c.Agclos.State()
	if v == nil {
		return nil
	}
	return &State{Agdstate: v}
}

func (c *Clos) SetState(v *State) {
	if v == nil || v.Agdstate == nil {
		return
	}
	c.Agclos.SetState(v.Agdstate)
}

func (c *Clos) StrDict() *cdt.Dict {
	v := c.Agclos.Strdict()
	if v == nil {
		return nil
	}
	return &cdt.Dict{Dict: v}
}

func (c *Clos) SetStrDict(v *cdt.Dict) {
	if v == nil || v.Dict == nil {
		return
	}
	c.Agclos.SetStrdict(v.Dict)
}

func (c *Clos) Seq() [3]uint64 {
	return c.Agclos.Seq()
}

func (c *Clos) SetSeq(v []uint64) {
	c.Agclos.SetSeq(v)
}

func (c *Clos) Callback() *CallbackStack {
	v := c.Agclos.Cb()
	if v == nil {
		return nil
	}
	return &CallbackStack{Agcbstack: v}
}

func (c *Clos) SetCallback(v *CallbackStack) {
	if v == nil || v.Agcbstack == nil {
		return
	}
	c.Agclos.SetCb(v.Agcbstack)
}

func (c *Clos) CallbacksEnabled() bool {
	return c.Agclos.CallbacksEnabled()
}

func (c *Clos) SetCallbacskEnabled(v bool) {
	c.Agclos.SetCallbacksEnabled(v)
}

func (c *Clos) LookupByName() [3]*cdt.Dict {
	v := c.Agclos.LookupByName()
	r := [3]*cdt.Dict{}
	r[0] = &cdt.Dict{Dict: v[0]}
	r[1] = &cdt.Dict{Dict: v[1]}
	r[2] = &cdt.Dict{Dict: v[2]}
	return r
}

func (c *Clos) LookupByID() [3]*cdt.Dict {
	v := c.Agclos.LookupByID()
	r := [3]*cdt.Dict{}
	r[0] = &cdt.Dict{Dict: v[0]}
	r[1] = &cdt.Dict{Dict: v[1]}
	r[2] = &cdt.Dict{Dict: v[2]}
	return r
}

func (a *Attr) H() *Record {
	v := a.Agattr.H()
	if v == nil {
		return nil
	}
	return &Record{Agrec: v}
}

func (a *Attr) SetH(v *Record) {
	if v == nil || v.Agrec == nil {
		return
	}
	a.Agattr.SetH(v.Agrec)
}

func (a *Attr) Dict() *cdt.Dict {
	v := a.Agattr.Dict()
	if v == nil {
		return nil
	}
	return &cdt.Dict{Dict: v}
}

func (a *Attr) SetDict(v *cdt.Dict) {
	if v == nil || v.Dict == nil {
		return
	}
	a.Agattr.SetDict(v.Dict)
}

func (s *Symbol) Link() *cdt.Link {
	v := s.Agsym.Link()
	if v == nil {
		return nil
	}
	return &cdt.Link{Dtlink: v}
}

func (s *Symbol) SetLink(v *cdt.Link) {
	if v == nil || v.Dtlink == nil {
		return
	}
	s.Agsym.SetLink(v.Dtlink)
}

func (s *Symbol) Name() string {
	return s.Agsym.Name()
}

func (s *Symbol) SetName(v string) {
	s.Agsym.SetName(v)
}

func (s *Symbol) Defval() string {
	return s.Agsym.Defval()
}

func (s *Symbol) SetDefval(v string) {
	s.Agsym.SetDefval(v)
}

func (s *Symbol) ID() int {
	return s.Agsym.ID()
}

func (s *Symbol) SetID(v int) {
	s.Agsym.SetID(v)
}

func (s *Symbol) Kind() uint {
	return s.Agsym.Kind()
}

func (s *Symbol) SetKind(v uint) {
	s.Agsym.SetKind(v)
}

func (s *Symbol) Fixed() uint {
	return s.Agsym.Fixed()
}

func (s *Symbol) SetFixed(v uint) {
	s.Agsym.SetFixed(v)
}

func (s *Symbol) Print() uint {
	return s.Agsym.Print()
}

func (s *Symbol) SetPrint(v uint) {
	s.Agsym.SetPrint(v)
}

func (d *DataDict) H() *Record {
	v := d.Agdatadict.H()
	if v == nil {
		return nil
	}
	return &Record{Agrec: v}
}

func (d *DataDict) SetH(v *Record) {
	if v == nil || v.Agrec == nil {
		return
	}
	d.Agdatadict.SetH(v.Agrec)
}

func (d *DataDict) DictN() *cdt.Dict {
	v := d.Agdatadict.DictN()
	if v == nil {
		return nil
	}
	return &cdt.Dict{Dict: v}
}

func (d *DataDict) SetDictN(v *cdt.Dict) {
	if v == nil || v.Dict == nil {
		return
	}
	d.Agdatadict.SetDictN(v.Dict)
}

func (d *DataDict) DictE() *cdt.Dict {
	v := d.Agdatadict.DictE()
	if v == nil {
		return nil
	}
	return &cdt.Dict{Dict: v}
}

func (d *DataDict) SetDictE(v *cdt.Dict) {
	if v == nil || v.Dict == nil {
		return
	}
	d.Agdatadict.SetDictE(v.Dict)
}

func (d *DataDict) DictG() *cdt.Dict {
	v := d.Agdatadict.DictG()
	if v == nil {
		return nil
	}
	return &cdt.Dict{Dict: v}
}

func (d *DataDict) SetDictG(v *cdt.Dict) {
	if v == nil || v.Dict == nil {
		return
	}
	d.Agdatadict.SetDictG(v.Dict)
}

func (g *Graph) Base() *Object {
	v := g.Agraph.Base()
	if v == nil {
		return nil
	}
	return &Object{Agobj: v}
}

func (g *Graph) SetBase(v *Object) {
	if v == nil || v.Agobj == nil {
		return
	}
	g.Agraph.SetBase(v.Agobj)
}

func (g *Graph) Desc() *Desc {
	v := g.Agraph.Desc()
	if v == nil {
		return nil
	}
	return &Desc{Agdesc: v}
}

func (g *Graph) SetDesc(v *Desc) {
	if v == nil || v.Agdesc == nil {
		return
	}
	g.Agraph.SetDesc(v.Agdesc)
}

func (g *Graph) Link() *cdt.Link {
	v := g.Agraph.Link()
	if v == nil {
		return nil
	}
	return &cdt.Link{Dtlink: v}
}

func (g *Graph) SetLink(v *cdt.Link) {
	if v == nil || v.Dtlink == nil {
		return
	}
	g.Agraph.SetLink(v.Dtlink)
}

func (g *Graph) NSeq() *cdt.Dict {
	v := g.Agraph.NSeq()
	if v == nil {
		return nil
	}
	return &cdt.Dict{Dict: v}
}

func (g *Graph) SetNSeq(v *cdt.Dict) {
	if v == nil || v.Dict == nil {
		return
	}
	g.Agraph.SetNSeq(v.Dict)
}

func (g *Graph) NID() *cdt.Dict {
	v := g.Agraph.NID()
	if v == nil {
		return nil
	}
	return &cdt.Dict{Dict: v}
}

func (g *Graph) SetNID(v *cdt.Dict) {
	if v == nil || v.Dict == nil {
		return
	}
	g.Agraph.SetNID(v.Dict)
}

func (g *Graph) ESeq() *cdt.Dict {
	v := g.Agraph.ESeq()
	if v == nil {
		return nil
	}
	return &cdt.Dict{Dict: v}
}

func (g *Graph) SetESeq(v *cdt.Dict) {
	if v == nil || v.Dict == nil {
		return
	}
	g.Agraph.SetESeq(v.Dict)
}

func (g *Graph) EID() *cdt.Dict {
	v := g.Agraph.EID()
	if v == nil {
		return nil
	}
	return &cdt.Dict{Dict: v}
}

func (g *Graph) SetEID(v *cdt.Dict) {
	if v == nil || v.Dict == nil {
		return
	}
	g.Agraph.SetEID(v.Dict)
}

func (g *Graph) GDict() *cdt.Dict {
	v := g.Agraph.GDict()
	if v == nil {
		return nil
	}
	return &cdt.Dict{Dict: v}
}

func (g *Graph) SetGDict(v *cdt.Dict) {
	if v == nil || v.Dict == nil {
		return
	}
	g.Agraph.SetGDict(v.Dict)
}

func (g *Graph) SetParent(v *Graph) {
	if v == nil || v.Agraph == nil {
		return
	}
	g.Agraph.SetParent(v.Agraph)
}

func (g *Graph) Root() *Graph {
	v := g.Agraph.Root()
	if v == nil {
		return nil
	}
	return &Graph{Agraph: v}
}

func (g *Graph) SetRootGraph(v *Graph) {
	if v == nil || v.Agraph == nil {
		return
	}
	g.Agraph.SetRoot(v.Agraph)
}

func (g *Graph) Clos() *Clos {
	v := g.Agraph.Clos()
	if v == nil {
		return nil
	}
	return &Clos{Agclos: v}
}

func (g *Graph) SetClos(v *Clos) {
	if v == nil || v.Agclos == nil {
		return
	}
	g.Agraph.SetClos(v.Agclos)
}

func (g *Graph) CopyAttr(t *Graph) int {
	return ccall.Agcopyattr(unsafe.Pointer(g.Agraph.C()), unsafe.Pointer(t.Agraph.C()))
}

func (g *Graph) BindRecord(name string, size uint, moveToFront int) {
	ccall.Agbindrec(unsafe.Pointer(g.Agraph.C()), name, size, moveToFront)
}

func (g *Graph) Record(name string, moveToFront int) *Record {
	rec := ccall.Aggetrec(unsafe.Pointer(g.Agraph.C()), name, moveToFront)
	if rec == nil {
		return nil
	}
	return &Record{Agrec: rec}
}

func (g *Graph) DeleteRecord(name string) int {
	return ccall.Agdelrec(unsafe.Pointer(g.Agraph.C()), name)
}

func (g *Graph) Get(name string) string {
	return ccall.Agget(unsafe.Pointer(g.Agraph.C()), name)
}

func (g *Graph) XGet(sym *Symbol) string {
	return ccall.Agxget(unsafe.Pointer(g.Agraph.C()), sym.Agsym)
}

func (g *Graph) Set(name, value string) int {
	return ccall.Agset(unsafe.Pointer(g.Agraph.C()), name, value)
}

func (g *Graph) XSet(sym *Symbol, value string) int {
	return ccall.Agxset(unsafe.Pointer(g.Agraph.C()), sym.Agsym, value)
}

func (g *Graph) SafeSet(name, value, def string) int {
	return ccall.Agsafeset(unsafe.Pointer(g.Agraph.C()), name, value, def)
}

func (g *Graph) Close() error {
	return ccall.Agclose(g.Agraph)
}

func (g *Graph) IsSimple() bool {
	return ccall.Agissimple(g.Agraph)
}

func (g *Graph) CreateNode(name string) (*Node, error) {
	node, err := ccall.Agnodef(g.Agraph, name, 1)
	if err != nil {
		return nil, err
	}
	return toNode(node), nil
}

func (g *Graph) Node(name string) (*Node, error) {
	node, err := ccall.Agnodef(g.Agraph, name, 0)
	if err != nil {
		return nil, err
	}
	return toNode(node), nil
}

func (g *Graph) IDNode(id IDTYPE, createFlag int) (*Node, error) {
	node, err := ccall.Agidnode(g.Agraph, uint64(id), createFlag)
	if err != nil {
		return nil, err
	}
	return toNode(node), nil
}

func (g *Graph) SubNode(n *Node, createFlag int) (*Node, error) {
	node, err := ccall.Agsubnodef(g.Agraph, n.Agnode, createFlag)
	if err != nil {
		return nil, err
	}
	return toNode(node), nil
}

func (g *Graph) FirstNode() *Node {
	return toNode(ccall.Agfstnode(g.Agraph))
}

func (g *Graph) NextNode(n *Node) *Node {
	return toNode(ccall.Agnxtnode(g.Agraph, n.Agnode))
}

func (g *Graph) LastNode() *Node {
	return toNode(ccall.Aglstnode(g.Agraph))
}

func (g *Graph) PreviousNode(n *Node) *Node {
	return toNode(ccall.Agprvnode(g.Agraph, n.Agnode))
}

func (g *Graph) SubRep(n *Node) *SubNode {
	return &SubNode{
		Agsubnode: ccall.Agsubrep(g.Agraph, n.Agnode),
	}
}

func (g *Graph) CreateEdge(name string, start *Node, end *Node) (*Edge, error) {
	edge, err := ccall.Agedgef(g.Agraph, start.Agnode, end.Agnode, name, 1)
	if err != nil {
		return nil, err
	}
	return toEdge(edge), nil
}

func (g *Graph) IDEdge(t *Node, h *Node, id IDTYPE, createFlag int) (*Edge, error) {
	edge, err := ccall.Agidedge(g.Agraph, t.Agnode, h.Agnode, uint64(id), createFlag)
	if err != nil {
		return nil, err
	}
	return toEdge(edge), nil
}

func (g *Graph) SubEdge(e *Edge, createFlag int) (*Edge, error) {
	edge, err := ccall.Agsubedge(g.Agraph, e.Agedge, createFlag)
	if err != nil {
		return nil, err
	}
	return toEdge(edge), nil
}

func (g *Graph) FirstIn(n *Node) *Edge {
	return toEdge(ccall.Agfstin(g.Agraph, n.Agnode))
}

func (g *Graph) NextIn(n *Edge) *Edge {
	return toEdge(ccall.Agnxtin(g.Agraph, n.Agedge))
}

func (g *Graph) FirstOut(n *Node) *Edge {
	return toEdge(ccall.Agfstout(g.Agraph, n.Agnode))
}

func (g *Graph) NextOut(e *Edge) *Edge {
	return toEdge(ccall.Agnxtout(g.Agraph, e.Agedge))
}

func (g *Graph) FirstEdge(n *Node) *Edge {
	return toEdge(ccall.Agfstedge(g.Agraph, n.Agnode))
}

func (g *Graph) NextEdge(e *Edge, n *Node) *Edge {
	return toEdge(ccall.Agnxtedge(g.Agraph, e.Agedge, n.Agnode))
}

func (g *Graph) Contains(o interface{}) bool {
	switch t := o.(type) {
	case *Graph:
		return ccall.Agcontains(g.Agraph, unsafe.Pointer(t.Agraph.C()))
	case *Node:
		return ccall.Agcontains(g.Agraph, unsafe.Pointer(t.Agnode.C()))
	case *Edge:
		return ccall.Agcontains(g.Agraph, unsafe.Pointer(t.Agedge.C()))
	}
	return false
}

func (g *Graph) Name() string {
	return ccall.Agnameof(unsafe.Pointer(g.Agraph.C()))
}

func (g *Graph) Delete(obj unsafe.Pointer) error {
	return ccall.Agdelete(g.Agraph, obj)
}

func (g *Graph) DeleteSubGraph(sub *Graph) int32 {
	return ccall.Agdelsubg(g.Agraph, sub.Agraph)
}

func (g *Graph) DeleteNode(n *Node) bool {
	return ccall.Agdelnode(g.Agraph, n.Agnode) == 1
}

func (g *Graph) DeleteEdge(e *Edge) bool {
	return ccall.Agdeledge(g.Agraph, e.Agedge) == 1
}

func (g *Graph) Strdup(s string) string {
	return ccall.Agstrdup(g.Agraph, s)
}

func (g *Graph) StrdupHTML(s string) string {
	return ccall.AgstrdupHTML(g.Agraph, s)
}

func (g *Graph) StrBind(s string) string {
	return ccall.Agstrbind(g.Agraph, s)
}

func (g *Graph) StrFree(s string) int {
	return ccall.Agstrfree(g.Agraph, s)
}

func (g *Graph) Attr(kind int, name, value string) *Symbol {
	return &Symbol{
		Agsym: ccall.Agattrf(g.Agraph, kind, name, value),
	}
}

func (g *Graph) NextAttr(kind int, attr *Symbol) *Symbol {
	return &Symbol{
		Agsym: ccall.Agnxtattr(g.Agraph, kind, attr.Agsym),
	}
}

func (g *Graph) Init(kind int, recName string, recSize int, moveToFront int) {
	ccall.Aginit(g.Agraph, kind, recName, recSize, moveToFront)
}

func (g *Graph) Clean(kind int, recName string) {
	ccall.Agclean(g.Agraph, kind, recName)
}

func (g *Graph) SubGraph(name string, cflag int) *Graph {
	return &Graph{
		Agraph: ccall.Agsubg(g.Agraph, name, cflag),
	}
}

func (g *Graph) IDSubGraph(id IDTYPE, cflag int) *Graph {
	return &Graph{
		Agraph: ccall.Agidsubg(g.Agraph, uint64(id), cflag),
	}
}

func (g *Graph) FirstSubGraph() *Graph {
	return &Graph{
		Agraph: ccall.Agfstsubg(g.Agraph),
	}
}

func (g *Graph) NextSubGraph() *Graph {
	return &Graph{
		Agraph: ccall.Agnxtsubg(g.Agraph),
	}
}

func (g *Graph) Parent() *Graph {
	return &Graph{
		Agraph: ccall.Agparent(g.Agraph),
	}
}

func (g *Graph) NumberNodes() int {
	return ccall.Agnnodes(g.Agraph)
}

func (g *Graph) NumberEdges() int {
	return ccall.Agnedges(g.Agraph)
}

func (g *Graph) NumberSubGraph() int {
	return ccall.Agnsubg(g.Agraph)
}

func (g *Graph) Degree(n *Node, in, out int) int {
	return ccall.Agdegree(g.Agraph, n.Agnode, in, out)
}

func (g *Graph) CountUniqueEdges(n *Node, in, out int) int {
	return ccall.Agcountuniqedges(g.Agraph, n.Agnode, in, out)
}

func (g *Graph) InternalMapClearLocalNames() {
	ccall.Aginternalmapclearlocalnames(g.Agraph)
}

func (g *Graph) Flatten(flag int) {
	ccall.Agflatten(g.Agraph, flag)
}

func (n *Node) Name() string {
	return ccall.Agnameof(unsafe.Pointer(n.Agnode.C()))
}

func (n *Node) CopyAttr(t *Node) int {
	return ccall.Agcopyattr(unsafe.Pointer(n.Agnode.C()), unsafe.Pointer(t.Agnode.C()))
}

func (n *Node) BindRecord(name string, size uint, moveToFront int) {
	ccall.Agbindrec(unsafe.Pointer(n.Agnode.C()), name, size, moveToFront)
}

func (n *Node) Record(name string, moveToFront int) *Record {
	rec := ccall.Aggetrec(unsafe.Pointer(n.Agnode.C()), name, moveToFront)
	if rec == nil {
		return nil
	}
	return &Record{Agrec: rec}
}

func (n *Node) DeleteRecord(name string) int {
	return ccall.Agdelrec(unsafe.Pointer(n.Agnode.C()), name)
}

func (n *Node) Get(name string) string {
	return ccall.Agget(unsafe.Pointer(n.Agnode.C()), name)
}

func (n *Node) XGet(sym *Symbol) string {
	return ccall.Agxget(unsafe.Pointer(n.Agnode.C()), sym.Agsym)
}

func (n *Node) Set(name, value string) int {
	return ccall.Agset(unsafe.Pointer(n.Agnode.C()), name, value)
}

func (n *Node) XSet(sym *Symbol, value string) int {
	return ccall.Agxset(unsafe.Pointer(n.Agnode.C()), sym.Agsym, value)
}

func (n *Node) SafeSet(name, value, def string) int {
	return ccall.Agsafeset(unsafe.Pointer(n.Agnode.C()), name, value, def)
}

func (n *Node) ReLabel(newname string) error {
	return ccall.AgrelabelNode(n.Agnode, newname)
}

func (n *Node) Before(v *Node) error {
	return ccall.Agnodebefore(n.Agnode, v.Agnode)
}

func (e *Edge) Name() string {
	return ccall.Agnameof(unsafe.Pointer(e.Agedge.C()))
}

func (e *Edge) CopyAttr(t *Edge) int {
	return ccall.Agcopyattr(unsafe.Pointer(e.Agedge.C()), unsafe.Pointer(t.Agedge.C()))
}

func (e *Edge) BindRecord(name string, size uint, moveToFront int) {
	ccall.Agbindrec(unsafe.Pointer(e.Agedge.C()), name, size, moveToFront)
}

func (e *Edge) Record(name string, moveToFront int) *Record {
	rec := ccall.Aggetrec(unsafe.Pointer(e.Agedge.C()), name, moveToFront)
	if rec == nil {
		return nil
	}
	return &Record{Agrec: rec}
}

func (e *Edge) DeleteRecord(name string) int {
	return ccall.Agdelrec(unsafe.Pointer(e.Agedge.C()), name)
}

func (e *Edge) Get(name string) string {
	return ccall.Agget(unsafe.Pointer(e.Agedge.C()), name)
}

func (e *Edge) XGet(sym *Symbol) string {
	return ccall.Agxget(unsafe.Pointer(e.Agedge.C()), sym.Agsym)
}

func (e *Edge) Set(name, value string) int {
	return ccall.Agset(unsafe.Pointer(e.Agedge.C()), name, value)
}

func (e *Edge) XSet(sym *Symbol, value string) int {
	return ccall.Agxset(unsafe.Pointer(e.Agedge.C()), sym.Agsym, value)
}

func (e *Edge) SafeSet(name, value, def string) int {
	return ccall.Agsafeset(unsafe.Pointer(e.Agedge.C()), name, value, def)
}
