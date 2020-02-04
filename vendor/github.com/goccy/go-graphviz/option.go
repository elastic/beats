package graphviz

import "github.com/goccy/go-graphviz/cgraph"

type GraphOption func(g *Graphviz)

var (
	Directed         = func(g *Graphviz) { g.dir = cgraph.Directed }
	StrictDirected   = func(g *Graphviz) { g.dir = cgraph.StrictDirected }
	UnDirected       = func(g *Graphviz) { g.dir = cgraph.UnDirected }
	StrictUnDirected = func(g *Graphviz) { g.dir = cgraph.StrictUnDirected }
)

func Name(name string) GraphOption {
	return func(g *Graphviz) {
		g.name = name
	}
}
