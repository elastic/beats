package graphviz

import (
	"image"
	"io"

	"github.com/goccy/go-graphviz/cgraph"
	"github.com/goccy/go-graphviz/gvc"
)

type Graphviz struct {
	ctx    *gvc.Context
	name   string
	dir    *cgraph.Desc
	layout Layout
}

type Layout string

const (
	CIRCO     Layout = "circo"
	DOT       Layout = "dot"
	FDP       Layout = "fdp"
	NEATO     Layout = "neato"
	OSAGE     Layout = "osage"
	PATCHWORK Layout = "patchwork"
	SFDP      Layout = "sfdp"
	TWOPI     Layout = "twopi"
)

type Format string

const (
	XDOT Format = "dot"
	SVG  Format = "svg"
	PNG  Format = "png"
	JPG  Format = "jpg"
)

func ParseFile(path string) (*cgraph.Graph, error) {
	graph, err := cgraph.ParseFile(path)
	if err != nil {
		return nil, err
	}
	return graph, nil
}

func ParseBytes(bytes []byte) (*cgraph.Graph, error) {
	graph, err := cgraph.ParseBytes(bytes)
	if err != nil {
		return nil, err
	}
	return graph, nil
}

func New() *Graphviz {
	return &Graphviz{
		ctx:    gvc.New(),
		dir:    cgraph.Directed,
		layout: DOT,
	}
}

func (g *Graphviz) Close() {
	g.ctx.Close()
}

func (g *Graphviz) SetLayout(layout Layout) *Graphviz {
	g.layout = layout
	return g
}

func (g *Graphviz) Render(graph *cgraph.Graph, format Format, w io.Writer) (e error) {
	if err := g.ctx.Layout(graph, string(g.layout)); err != nil {
		return err
	}
	defer func() {
		if err := g.ctx.FreeLayout(graph); err != nil {
			e = err
		}
	}()

	if err := g.ctx.RenderData(graph, string(format), w); err != nil {
		return err
	}
	return nil
}

func (g *Graphviz) RenderImage(graph *cgraph.Graph, format Format) (img image.Image, e error) {
	if err := g.ctx.Layout(graph, string(g.layout)); err != nil {
		return nil, err
	}
	defer func() {
		if err := g.ctx.FreeLayout(graph); err != nil {
			e = err
		}
	}()
	image, err := g.ctx.RenderImage(graph, string(format))
	if err != nil {
		return nil, err
	}
	return image, nil
}

func (g *Graphviz) RenderFilename(graph *cgraph.Graph, format Format, path string) (e error) {
	if err := g.ctx.Layout(graph, string(g.layout)); err != nil {
		return err
	}
	defer func() {
		if err := g.ctx.FreeLayout(graph); err != nil {
			e = err
		}
	}()

	if err := g.ctx.RenderFilename(graph, string(format), path); err != nil {
		return err
	}
	return nil
}

func (g *Graphviz) Graph(option ...GraphOption) (*cgraph.Graph, error) {
	for _, opt := range option {
		opt(g)
	}
	graph, err := cgraph.Open(g.name, g.dir, nil)
	if err != nil {
		return nil, err
	}
	return graph, nil
}
