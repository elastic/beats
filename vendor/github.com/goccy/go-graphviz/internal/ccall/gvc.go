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
#include "gvc.h"
#include "gvcjob.h"
#include "cgraph.h"
#include <stdlib.h>
*/
import "C"
import (
	"io"
	"os"
	"reflect"
	"unsafe"
)

type GVC struct {
	c *C.GVC_t
}

type GVJ struct {
	c *C.GVJ_t
}

type GVCommon struct {
	c *C.GVCOMMON_t
}

type ObjState struct {
	c *C.obj_state_t
}

type Pointf struct {
	X float64
	Y float64
}

type Point struct {
	X int
	Y int
}

type Box struct {
	LL Point
	UR Point
}

type Boxf struct {
	LL Pointf
	UR Pointf
}

type GVColor struct {
	R uint
	G uint
	B uint
	A uint
}

func ToGVC(c *C.GVC_t) *GVC {
	if c == nil {
		return nil
	}
	return &GVC{c: c}
}

func (g *GVC) C() *C.GVC_t {
	if g == nil {
		return nil
	}
	return g.c
}

func ToGVJ(c *C.GVJ_t) *GVJ {
	if c == nil {
		return nil
	}
	return &GVJ{c: c}
}

func (g *GVJ) C() *C.GVJ_t {
	if g == nil {
		return nil
	}
	return g.c
}

func ToGVCommon(c *C.GVCOMMON_t) *GVCommon {
	if c == nil {
		return nil
	}
	return &GVCommon{c: c}
}

func ToObjState(c *C.obj_state_t) *ObjState {
	if c == nil {
		return nil
	}
	return &ObjState{c: c}
}

func ToPointf(c C.pointf) Pointf {
	return Pointf{X: float64(c.x), Y: float64(c.y)}
}

func ToPointsf(c *C.pointf, n C.int) []Pointf {
	var p []C.pointf
	v := (*reflect.SliceHeader)(unsafe.Pointer(&p))
	v.Cap = int(n)
	v.Len = int(n)
	v.Data = uintptr(unsafe.Pointer(c))
	points := make([]Pointf, 0, int(n))
	for _, point := range p {
		points = append(points, ToPointf(point))
	}
	return points
}

func ToPoint(c C.point) Point {
	return Point{X: int(c.x), Y: int(c.y)}
}

func ToBoxf(c C.boxf) Boxf {
	return Boxf{
		LL: ToPointf(c.LL),
		UR: ToPointf(c.UR),
	}
}

func ToBox(c C.box) Box {
	return Box{
		LL: ToPoint(c.LL),
		UR: ToPoint(c.UR),
	}
}

func ToGVColor(c C.gvcolor_t) GVColor {
	var rgba []byte
	p := (*reflect.SliceHeader)(unsafe.Pointer(&rgba))
	p.Cap = 4
	p.Len = 4
	p.Data = uintptr(unsafe.Pointer(&c.u))
	return GVColor{
		R: uint(rgba[0]),
		G: uint(rgba[1]),
		B: uint(rgba[2]),
		A: uint(rgba[3]),
	}
}

func (g *ObjState) Parent() *ObjState {
	v := g.c.parent
	if v == nil {
		return nil
	}
	return &ObjState{c: v}
}

type ObjType int

const (
	ROOTGRAPH_OBJTYPE ObjType = iota
	CLUSTER_OBJTYPE
	NODE_OBJTYPE
	EDGE_OBJTYPE
)

func (g *ObjState) Type() ObjType {
	return ObjType(g.c._type)
}

func (g *ObjState) Graph() *Agraph {
	v := (*C.Agraph_t)(unsafe.Pointer(&g.c.u))
	if v == nil {
		return nil
	}
	return &Agraph{c: v}
}

func (g *ObjState) SubGraph() *Agraph {
	v := (*C.Agraph_t)(unsafe.Pointer(&g.c.u))
	if v == nil {
		return nil
	}
	return &Agraph{c: v}
}

func (g *ObjState) Node() *Agnode {
	v := (*C.Agnode_t)(unsafe.Pointer(&g.c.u))
	if v == nil {
		return nil
	}
	return &Agnode{c: v}
}

func (g *ObjState) Edge() *Agedge {
	v := (*C.Agedge_t)(unsafe.Pointer(&g.c.u))
	if v == nil {
		return nil
	}
	return &Agedge{c: v}
}

type EmitState int

const (
	EMIT_GDRAW EmitState = iota
	EMIT_CDRAW
	EMIT_TDRAW
	EMIT_HDRAW
	EMIT_GLABEL
	EMIT_CLABEL
	EMIT_TLABEL
	EMIT_HLABEL
	EMIT_NDRAW
	EMIT_EDRAW
	EMIT_NLABEL
	EMIT_ELABEL
)

func (g *ObjState) EmitState() EmitState {
	return EmitState(g.c.emit_state)
}

func (g *ObjState) PenColor() GVColor {
	return ToGVColor(g.c.pencolor)
}

func (g *ObjState) FillColor() GVColor {
	return ToGVColor(g.c.fillcolor)
}

func (g *ObjState) StopColor() GVColor {
	return ToGVColor(g.c.stopcolor)
}

func (g *ObjState) GradientAngle() int {
	return int(g.c.gradient_angle)
}

func (g *ObjState) GradientFrac() float32 {
	return float32(g.c.gradient_frac)
}

type PenType int

const (
	PEN_NONE PenType = iota
	PEN_DASHED
	PEN_DOTTED
	PEN_SOLID
)

type FillType int

const (
	FILL_NONE FillType = iota
	FILL_SOLID
	FILL_LINEAR
	FILL_RADIAL
)

func (g *ObjState) Pen() PenType {
	return PenType(g.c.pen)
}

func (g *ObjState) Fill() FillType {
	return FillType(g.c.fill)
}

func (g *ObjState) PenWidth() float64 {
	return float64(g.c.penwidth)
}

func (g *ObjState) RawStyle() []string {
	return []string{}
}

func (g *ObjState) Z() float64 {
	return float64(g.c.z)
}

func (g *ObjState) TailZ() float64 {
	return float64(g.c.tail_z)
}

func (g *ObjState) HeadZ() float64 {
	return float64(g.c.head_z)
}

func (g *ObjState) Label() string {
	return C.GoString(g.c.label)
}

func (g *ObjState) XLabel() string {
	return C.GoString(g.c.xlabel)
}

func (g *ObjState) TailLabel() string {
	return C.GoString(g.c.taillabel)
}

func (g *ObjState) HeadLabel() string {
	return C.GoString(g.c.headlabel)
}

func (g *ObjState) URL() string {
	return C.GoString(g.c.url)
}

func (g *ObjState) ID() string {
	return C.GoString(g.c.id)
}

func (g *ObjState) LabelURL() string {
	return C.GoString(g.c.labelurl)
}

func (g *ObjState) TailURL() string {
	return C.GoString(g.c.tailurl)
}

func (g *ObjState) HeadURL() string {
	return C.GoString(g.c.headurl)
}

func (g *ObjState) Tooltip() string {
	return C.GoString(g.c.tooltip)
}

func (g *ObjState) LabelTooltip() string {
	return C.GoString(g.c.labeltooltip)
}

func (g *ObjState) TailTooltip() string {
	return C.GoString(g.c.tailtooltip)
}

func (g *ObjState) HeadTooltip() string {
	return C.GoString(g.c.headtooltip)
}

func (g *ObjState) Target() string {
	return C.GoString(g.c.target)
}

func (g *ObjState) LabelTarget() string {
	return C.GoString(g.c.labeltarget)
}

func (g *ObjState) TailTarget() string {
	return C.GoString(g.c.tailtarget)
}

func (g *ObjState) HeadTarget() string {
	return C.GoString(g.c.headtarget)
}

type MapShape int

const (
	MAP_RECTANGLE MapShape = iota
	MAP_CIRCLE
	MAP_POLYGON
)

func (g *ObjState) URLMapShape() MapShape {
	return MapShape(g.c.url_map_shape)
}

func (g *ObjState) URLMapN() int {
	return int(g.c.url_map_n)
}

func (g *ObjState) URLMapP() []Pointf {
	return ToPointsf(g.c.url_map_p, g.c.url_map_n)
}

func (g *ObjState) URLBsplinemapPolyN() int {
	return int(g.c.url_bsplinemap_poly_n)
}

func (g *ObjState) URLBsplinemapN() []int {
	var p []C.int
	v := (*reflect.SliceHeader)(unsafe.Pointer(&p))
	v.Cap = int(g.c.url_bsplinemap_poly_n)
	v.Len = int(g.c.url_bsplinemap_poly_n)
	v.Data = uintptr(unsafe.Pointer(g.c.url_bsplinemap_n))
	n := make([]int, 0, int(g.c.url_bsplinemap_poly_n))
	for _, pp := range p {
		n = append(n, int(pp))
	}
	return n
}

func (g *ObjState) URLBsplinemapP() []Pointf {
	return ToPointsf(g.c.url_bsplinemap_p, g.c.url_bsplinemap_poly_n)
}

func (g *ObjState) TailEndURLMapN() int {
	return int(g.c.tailendurl_map_n)
}

func (g *ObjState) TailEndURLMapP() []Pointf {
	return ToPointsf(g.c.tailendurl_map_p, g.c.tailendurl_map_n)
}

func (g *ObjState) HeadEndURLMapN() int {
	return int(g.c.headendurl_map_n)
}

func (g *ObjState) HeadEndURLMapP() []Pointf {
	return ToPointsf(g.c.headendurl_map_p, g.c.headendurl_map_n)
}

func (g *GVJ) GVC() *GVC {
	return ToGVC(g.c.gvc)
}

func (g *GVJ) Next() *GVJ {
	return ToGVJ(g.c.next)
}

func (g *GVJ) NextActive() *GVJ {
	return ToGVJ(g.c.next_active)
}

func (g *GVJ) Common() *GVCommon {
	return ToGVCommon(g.c.common)
}

func (g *GVJ) Obj() *ObjState {
	return ToObjState(g.c.obj)
}

func (g *GVJ) InputFilename() string {
	return C.GoString(g.c.input_filename)
}

func (g *GVJ) GraphIndex() int {
	return int(g.c.graph_index)
}

func (g *GVJ) LayoutType() string {
	return C.GoString(g.c.layout_type)
}

func (g *GVJ) OutputFilename() string {
	return C.GoString(g.c.output_filename)
}

func (g *GVJ) OutputFile() *os.File {
	fd := C.fileno(g.c.output_file)
	return os.NewFile(uintptr(fd), g.OutputFilename())
}

func (g *GVJ) OutputData() []byte {
	if g.c.output_data == nil {
		return nil
	}
	return []byte(C.GoString(g.c.output_data))
}

func (g *GVJ) SetOutputData(v []byte) {
	length := len(v)
	g.c.output_data = (*C.char)(C.realloc(unsafe.Pointer(g.c.output_data), C.ulong(length)))
	header := (*reflect.SliceHeader)(unsafe.Pointer(&v))
	C.memcpy(unsafe.Pointer(g.c.output_data), unsafe.Pointer(header.Data), C.ulong(length))
	g.c.output_data_position = C.uint(length)
}

func (g *GVJ) OutputDataAllocated() uint {
	return uint(g.c.output_data_allocated)
}

func (g *GVJ) OutputDataPosition() uint {
	return uint(g.c.output_data_position)
}

func (g *GVJ) OutputLangname() string {
	return C.GoString(g.c.output_langname)
}

func (g *GVJ) OutputLang() int {
	return int(g.c.output_lang)
}

func (g *GVJ) DeviceDPI() Pointf {
	return ToPointf(g.c.device_dpi)
}

func (g *GVJ) DeviceSetsDPI() bool {
	return g.c.device_sets_dpi == 1
}

func (g *GVJ) Display() unsafe.Pointer {
	return g.c.display
}

func (g *GVJ) Screen() int {
	return int(g.c.screen)
}

func (g *GVJ) Context() unsafe.Pointer {
	return g.c.context
}

func (g *GVJ) ExternalContext() bool {
	return g.c.external_context == 1
}

func (g *GVJ) ImageData() []byte {
	return []byte(C.GoString(g.c.imagedata))
}

func (g *GVJ) Flags() int {
	return int(g.c.flags)
}

func (g *GVJ) NumLayers() int {
	return int(g.c.numLayers)
}

func (g *GVJ) LayerNum() int {
	return int(g.c.layerNum)
}

func (g *GVJ) PagesArraySize() Point {
	return ToPoint(g.c.pagesArraySize)
}

func (g *GVJ) PagesArrayFirst() Point {
	return ToPoint(g.c.pagesArrayFirst)
}

func (g *GVJ) PagesArrayMajor() Point {
	return ToPoint(g.c.pagesArrayMajor)
}

func (g *GVJ) PagesArrayMinor() Point {
	return ToPoint(g.c.pagesArrayMinor)
}

func (g *GVJ) PagesArrayElem() Point {
	return ToPoint(g.c.pagesArrayElem)
}

func (g *GVJ) NumPages() int {
	return int(g.c.numPages)
}

func (g *GVJ) BB() Boxf {
	return ToBoxf(g.c.bb)
}

func (g *GVJ) Pad() Pointf {
	return ToPointf(g.c.pad)
}

func (g *GVJ) Clip() Boxf {
	return ToBoxf(g.c.clip)
}

func (g *GVJ) PageBox() Boxf {
	return ToBoxf(g.c.pageBox)
}

func (g *GVJ) PageSize() Pointf {
	return ToPointf(g.c.pageSize)
}

func (g *GVJ) Focus() Pointf {
	return ToPointf(g.c.focus)
}

func (g *GVJ) Zoom() float64 {
	return float64(g.c.zoom)
}

func (g *GVJ) Rotation() int {
	return int(g.c.rotation)
}

func (g *GVJ) View() Pointf {
	return ToPointf(g.c.view)
}

func (g *GVJ) CanvasBox() Boxf {
	return ToBoxf(g.c.canvasBox)
}

func (g *GVJ) Margin() Pointf {
	return ToPointf(g.c.margin)
}

func (g *GVJ) DPI() Pointf {
	return ToPointf(g.c.dpi)
}

func (g *GVJ) Width() uint {
	return uint(g.c.width)
}

func (g *GVJ) Height() uint {
	return uint(g.c.height)
}

func (g *GVJ) PageBoundingBox() Box {
	return ToBox(g.c.pageBoundingBox)
}

func (g *GVJ) BoundingBox() Box {
	return ToBox(g.c.boundingBox)
}

func (g *GVJ) Scale() Pointf {
	return ToPointf(g.c.scale)
}

func (g *GVJ) Translation() Pointf {
	return ToPointf(g.c.translation)
}

func (g *GVJ) DevScale() Pointf {
	return ToPointf(g.c.devscale)
}

func (g *GVJ) FitMode() bool {
	return g.c.fit_mode == 1
}

func (g *GVJ) NeedsRefresh() bool {
	return g.c.needs_refresh == 1
}

func (g *GVJ) Click() bool {
	return g.c.click == 1
}

func (g *GVJ) HasGrown() bool {
	return g.c.has_grown == 1
}

func (g *GVJ) HasBeenRendered() bool {
	return g.c.has_been_rendered == 1
}

func (g *GVJ) Button() uint {
	return uint(g.c.button)
}

func (g *GVJ) Pointer() Pointf {
	return ToPointf(g.c.pointer)
}

func (g *GVJ) OldPointer() Pointf {
	return ToPointf(g.c.oldpointer)
}

func (g *GVJ) CurrentObj() unsafe.Pointer {
	return g.c.current_obj
}

func (g *GVJ) SelectedObj() unsafe.Pointer {
	return g.c.selected_obj
}

func (g *GVJ) ActiveTooltip() []byte {
	return []byte(C.GoString(g.c.active_tooltip))
}

func (g *GVJ) SelectedHref() []byte {
	return []byte(C.GoString(g.c.selected_href))
}

func (g *GVJ) Window() unsafe.Pointer {
	return g.c.window
}

func (g *GVJ) NumKeys() int {
	return int(g.c.numkeys)
}

func (g *GVJ) KeyCodes() unsafe.Pointer {
	return g.c.keycodes
}

func GvContext() *GVC {
	v := C.gvContext()
	if v == nil {
		return nil
	}
	return &GVC{c: v}
}

func GvcVersion(gvc *GVC) string {
	return C.GoString(C.gvcVersion(gvc.C()))
}

func GvcBuildDate(gvc *GVC) string {
	return C.GoString(C.gvcBuildDate(gvc.C()))
}

func GvNextInputGraph(gvc *GVC) *Agraph {
	return ToAgraph(C.gvNextInputGraph(gvc.C()))
}

func GvPluginsGraph(gvc *GVC) *Agraph {
	return ToAgraph(C.gvPluginsGraph(gvc.C()))
}

func GvLayout(gvc *GVC, g *Agraph, engine string) error {
	C.gvLayout(gvc.C(), g.C(), C.CString(engine))
	return Aglasterr()
}

func GvLayoutJobs(gvc *GVC, g *Agraph) error {
	C.gvLayoutJobs(gvc.C(), g.C())
	return Aglasterr()
}

func AttachAttrs(g *Agraph) {
	C.attach_attrs(g.C())
}

func GvRenderData(gvc *GVC, g *Agraph, format string, w io.Writer) error {
	var (
		buf    *C.char
		length C.uint
	)
	C.gvRenderData(gvc.C(), g.C(), C.CString(format), &buf, &length)
	if err := Aglasterr(); err != nil {
		return err
	}
	defer C.gvFreeRenderData(buf)
	var gobuf []byte
	header := (*reflect.SliceHeader)(unsafe.Pointer(&gobuf))
	header.Cap = int(length)
	header.Len = int(length)
	header.Data = uintptr(unsafe.Pointer(buf))
	if _, err := w.Write(gobuf); err != nil {
		return err
	}
	return nil
}

func GvRenderFilename(gvc *GVC, g *Agraph, format, filename string) error {
	C.gvRenderFilename(gvc.C(), g.C(), C.CString(format), C.CString(filename))
	return Aglasterr()
}

func GvRenderContext(gvc *GVC, g *Agraph, format string, context unsafe.Pointer) error {
	C.gvRenderContext(gvc.C(), g.C(), C.CString(format), context)
	return Aglasterr()
}

func GvRenderJobs(gvc *GVC, g *Agraph) error {
	C.gvRenderJobs(gvc.C(), g.C())
	return Aglasterr()
}

func GvFreeLayout(gvc *GVC, g *Agraph) error {
	C.gvFreeLayout(gvc.C(), g.C())
	return Aglasterr()
}

func GvFinalize(gvc *GVC) {
	C.gvFinalize(gvc.C())
}

func GvFreeContext(gvc *GVC) error {
	C.gvFreeContext(gvc.C())
	return Aglasterr()
}

func GvToolTred(g *Agraph) int {
	return int(C.gvToolTred(g.C()))
}
