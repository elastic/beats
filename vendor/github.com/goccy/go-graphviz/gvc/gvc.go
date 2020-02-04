package gvc

import (
	"image"
	"io"
	"unsafe"

	"github.com/goccy/go-graphviz/cgraph"
	"github.com/goccy/go-graphviz/internal/ccall"
)

type Context struct {
	*ccall.GVC
}

type Job struct {
	*ccall.GVJ
}

func New() *Context {
	return &Context{GVC: ccall.GvContext()}
}

func (c *Context) Close() error {
	return ccall.GvFreeContext(c.GVC)
}

func (c *Context) Layout(g *cgraph.Graph, engine string) error {
	return ccall.GvLayout(c.GVC, g.Agraph, engine)
}

func (c *Context) RenderData(g *cgraph.Graph, format string, w io.Writer) error {
	if err := ccall.GvRenderData(c.GVC, g.Agraph, format, w); err != nil {
		return err
	}
	return nil
}

func (c *Context) RenderImage(g *cgraph.Graph, format string) (image.Image, error) {
	var img image.Image
	if err := ccall.GvRenderContext(c.GVC, g.Agraph, format, unsafe.Pointer(&img)); err != nil {
		return nil, err
	}
	return img, nil
}

func (c *Context) RenderFilename(g *cgraph.Graph, format, filename string) error {
	return ccall.GvRenderFilename(c.GVC, g.Agraph, format, filename)
}

func (c *Context) FreeLayout(g *cgraph.Graph) error {
	return ccall.GvFreeLayout(c.GVC, g.Agraph)
}
