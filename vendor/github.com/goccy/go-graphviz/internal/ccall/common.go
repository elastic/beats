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
#include "textspan.h"
*/
import "C"
import "unsafe"

type PostscriptAlias struct {
	c *C.PostscriptAlias
}

func ToPostscriptAlias(c *C.PostscriptAlias) *PostscriptAlias {
	if c == nil {
		return nil
	}
	return &PostscriptAlias{c: c}
}

func (g *PostscriptAlias) C() *C.PostscriptAlias {
	if g == nil {
		return nil
	}
	return g.c
}

func (g *PostscriptAlias) Name() string {
	return C.GoString(g.c.name)
}

func (g *PostscriptAlias) Family() string {
	return C.GoString(g.c.family)
}

func (g *PostscriptAlias) Weight() string {
	return C.GoString(g.c.weight)
}

func (g *PostscriptAlias) Stretch() string {
	return C.GoString(g.c.stretch)
}

func (g *PostscriptAlias) Style() string {
	return C.GoString(g.c.style)
}

func (g *PostscriptAlias) XFigCode() int {
	return int(g.c.xfig_code)
}

func (g *PostscriptAlias) SVGFontFamily() string {
	return C.GoString(g.c.svg_font_family)
}

func (g *PostscriptAlias) SVGFontWeight() string {
	return C.GoString(g.c.svg_font_weight)
}

func (g *PostscriptAlias) SVGFontStyle() string {
	return C.GoString(g.c.svg_font_style)
}

type TextFont struct {
	c *C.textfont_t
}

func ToTextFont(c *C.textfont_t) *TextFont {
	if c == nil {
		return nil
	}
	return &TextFont{c: c}
}

func (g *TextFont) Name() string {
	return C.GoString(g.c.name)
}

func (g *TextFont) Color() string {
	return C.GoString(g.c.color)
}

func (g *TextFont) PostscriptAlias() *PostscriptAlias {
	v := g.c.postscript_alias
	if v == nil {
		return nil
	}
	return &PostscriptAlias{c: v}
}

func (g *TextFont) Size() float64 {
	return float64(g.c.size)
}

type TextSpan struct {
	c *C.textspan_t
}

func ToTextSpan(c *C.textspan_t) *TextSpan {
	if c == nil {
		return nil
	}
	return &TextSpan{c: c}
}

func (t *TextSpan) C() *C.textspan_t {
	if t == nil {
		return nil
	}
	return t.c
}

func (t *TextSpan) Str() string {
	return C.GoString(t.c.str)
}

func (t *TextSpan) Font() *TextFont {
	return ToTextFont(t.c.font)
}

func (t *TextSpan) Layout() unsafe.Pointer {
	return t.c.layout
}

func (t *TextSpan) YOffsetLayout() float64 {
	return float64(t.c.yoffset_layout)
}

func (t *TextSpan) YOffsetCenterLine() float64 {
	return float64(t.c.yoffset_centerline)
}

func (t *TextSpan) Size() Pointf {
	return ToPointf(t.c.size)
}

func (t *TextSpan) Just() byte {
	return byte(t.c.just)
}
