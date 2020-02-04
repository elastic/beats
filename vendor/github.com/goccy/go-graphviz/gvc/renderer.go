package gvc

import (
	"sync"

	"github.com/goccy/go-graphviz/internal/ccall"
	"github.com/pkg/errors"
)

type Renderer interface {
	BeginJob(*Job) error
	EndJob(*Job) error
	BeginGraph(*Job) error
	EndGraph(*Job) error
	BeginLayer(*Job, string, int, int) error
	EndLayer(*Job) error
	BeginPage(*Job) error
	EndPage(*Job) error
	BeginCluster(*Job) error
	EndCluster(*Job) error
	BeginNodes(*Job) error
	EndNodes(*Job) error
	BeginEdges(*Job) error
	EndEdges(*Job) error
	BeginNode(*Job) error
	EndNode(*Job) error
	BeginEdge(*Job) error
	EndEdge(*Job) error
	BeginAnchor(*Job, string, string, string, string) error
	EndAnchor(*Job) error
	BeginLabel(*Job, int) error
	EndLabel(*Job) error
	TextSpan(*Job, Pointf, *TextSpan) error
	ResolveColor(*Job, Color) error
	Ellipse(*Job, Pointf, Pointf, int) error
	Polygon(*Job, []Pointf, int) error
	BezierCurve(*Job, []Pointf, int, int) error
	Polyline(*Job, []Pointf) error
	Comment(*Job, string) error
	LibraryShape(*Job, string, []Pointf, int) error
}

type DefaultRenderer struct{}

func (*DefaultRenderer) BeginJob(job *Job) error   { return nil }
func (*DefaultRenderer) EndJob(job *Job) error     { return nil }
func (*DefaultRenderer) BeginGraph(job *Job) error { return nil }
func (*DefaultRenderer) EndGraph(job *Job) error   { return nil }
func (*DefaultRenderer) BeginLayer(job *Job, layerName string, layerNum int, numLayers int) error {
	return nil
}
func (*DefaultRenderer) EndLayer(job *Job) error                                      { return nil }
func (*DefaultRenderer) BeginPage(job *Job) error                                     { return nil }
func (*DefaultRenderer) EndPage(job *Job) error                                       { return nil }
func (*DefaultRenderer) BeginCluster(job *Job) error                                  { return nil }
func (*DefaultRenderer) EndCluster(job *Job) error                                    { return nil }
func (*DefaultRenderer) BeginNodes(job *Job) error                                    { return nil }
func (*DefaultRenderer) EndNodes(job *Job) error                                      { return nil }
func (*DefaultRenderer) BeginEdges(job *Job) error                                    { return nil }
func (*DefaultRenderer) EndEdges(job *Job) error                                      { return nil }
func (*DefaultRenderer) BeginNode(job *Job) error                                     { return nil }
func (*DefaultRenderer) EndNode(job *Job) error                                       { return nil }
func (*DefaultRenderer) BeginEdge(job *Job) error                                     { return nil }
func (*DefaultRenderer) EndEdge(job *Job) error                                       { return nil }
func (*DefaultRenderer) BeginAnchor(job *Job, href, tooltip, target, id string) error { return nil }
func (*DefaultRenderer) EndAnchor(job *Job) error                                     { return nil }
func (*DefaultRenderer) BeginLabel(job *Job, typ int) error                           { return nil }
func (*DefaultRenderer) EndLabel(job *Job) error                                      { return nil }
func (*DefaultRenderer) TextSpan(job *Job, p Pointf, span *TextSpan) error            { return nil }
func (*DefaultRenderer) ResolveColor(job *Job, c Color) error                         { return nil }
func (*DefaultRenderer) Ellipse(job *Job, a0, a1 Pointf, filled int) error            { return nil }
func (*DefaultRenderer) Polygon(job *Job, a []Pointf, filled int) error               { return nil }
func (*DefaultRenderer) BezierCurve(job *Job, a []Pointf, arrowAtStart, arrowAtEnd int) error {
	return nil
}
func (*DefaultRenderer) Polyline(job *Job, a []Pointf) error    { return nil }
func (*DefaultRenderer) Comment(job *Job, comment string) error { return nil }
func (*DefaultRenderer) LibraryShape(job *Job, name string, a []Pointf, filled int) error {
	return nil
}

type rendererWithError struct {
	renderer Renderer
	err      error
}

func (r *rendererWithError) setError(err error) {
	r.err = err
	ccall.Agerr(err.Error())
}

var (
	renderers = map[string]*rendererWithError{}
	mu        sync.Mutex
)

func RegisterRenderer(name string, renderer Renderer) {
	mu.Lock()
	defer mu.Unlock()
	renderers[name] = &rendererWithError{renderer: renderer}
}

type Point struct {
	X int
	Y int
}

type Pointf struct {
	X float64
	Y float64
}

type Box struct {
	LL Point
	UR Point
}

type Boxf struct {
	LL Pointf
	UR Pointf
}

type Color struct {
	R uint
	G uint
	B uint
	A uint
}

type TextSpan struct {
	*ccall.TextSpan
}

func dispatchRenderer(job *ccall.GVJ) *rendererWithError {
	name := job.OutputLangname()
	r, exists := renderers[name]
	if !exists {
		r := &rendererWithError{}
		r.setError(errors.Errorf("could not find renderer for %s", name))
		renderers[name] = r
		return r
	}
	return r
}

func beginJob(job *ccall.GVJ) {
	r := dispatchRenderer(job)
	if r.err != nil {
		return
	}
	if err := r.renderer.BeginJob(&Job{GVJ: job}); err != nil {
		r.setError(err)
	}
}

func endJob(job *ccall.GVJ) {
	r := dispatchRenderer(job)
	if r.err != nil {
		return
	}
	if err := r.renderer.EndJob(&Job{GVJ: job}); err != nil {
		r.setError(err)
	}
}

func beginGraph(job *ccall.GVJ) {
	r := dispatchRenderer(job)
	if r.err != nil {
		return
	}
	if err := r.renderer.BeginGraph(&Job{GVJ: job}); err != nil {
		r.setError(err)
	}
}

func endGraph(job *ccall.GVJ) {
	r := dispatchRenderer(job)
	if r.err != nil {
		return
	}
	if err := r.renderer.EndGraph(&Job{GVJ: job}); err != nil {
		r.setError(err)
	}
}

func beginLayer(job *ccall.GVJ, layerName string, layerNum int, numLayers int) {
	r := dispatchRenderer(job)
	if r.err != nil {
		return
	}
	if err := r.renderer.BeginLayer(&Job{GVJ: job}, layerName, layerNum, numLayers); err != nil {
		r.setError(err)
	}
}

func endLayer(job *ccall.GVJ) {
	r := dispatchRenderer(job)
	if r.err != nil {
		return
	}
	if err := r.renderer.EndLayer(&Job{GVJ: job}); err != nil {
		r.setError(err)
	}
}

func beginPage(job *ccall.GVJ) {
	r := dispatchRenderer(job)
	if r.err != nil {
		return
	}
	if err := r.renderer.BeginPage(&Job{GVJ: job}); err != nil {
		r.setError(err)
	}
}

func endPage(job *ccall.GVJ) {
	r := dispatchRenderer(job)
	if r.err != nil {
		return
	}
	if err := r.renderer.EndPage(&Job{GVJ: job}); err != nil {
		r.setError(err)
	}
}

func beginCluster(job *ccall.GVJ) {
	r := dispatchRenderer(job)
	if r.err != nil {
		return
	}
	if err := r.renderer.BeginCluster(&Job{GVJ: job}); err != nil {
		r.setError(err)
	}
}

func endCluster(job *ccall.GVJ) {
	r := dispatchRenderer(job)
	if r.err != nil {
		return
	}
	if err := r.renderer.EndCluster(&Job{GVJ: job}); err != nil {
		r.setError(err)
	}
}

func beginNodes(job *ccall.GVJ) {
	r := dispatchRenderer(job)
	if r.err != nil {
		return
	}
	if err := r.renderer.BeginNodes(&Job{GVJ: job}); err != nil {
		r.setError(err)
	}
}

func endNodes(job *ccall.GVJ) {
	r := dispatchRenderer(job)
	if r.err != nil {
		return
	}
	if err := r.renderer.EndNodes(&Job{GVJ: job}); err != nil {
		r.setError(err)
	}
}

func beginEdges(job *ccall.GVJ) {
	r := dispatchRenderer(job)
	if r.err != nil {
		return
	}
	if err := r.renderer.BeginEdges(&Job{GVJ: job}); err != nil {
		r.setError(err)
	}
}

func endEdges(job *ccall.GVJ) {
	r := dispatchRenderer(job)
	if r.err != nil {
		return
	}
	if err := r.renderer.EndEdges(&Job{GVJ: job}); err != nil {
		r.setError(err)
	}
}

func beginNode(job *ccall.GVJ) {
	r := dispatchRenderer(job)
	if r.err != nil {
		return
	}
	if err := r.renderer.BeginNode(&Job{GVJ: job}); err != nil {
		r.setError(err)
	}
}

func endNode(job *ccall.GVJ) {
	r := dispatchRenderer(job)
	if r.err != nil {
		return
	}
	if err := r.renderer.EndNode(&Job{GVJ: job}); err != nil {
		r.setError(err)
	}
}

func beginEdge(job *ccall.GVJ) {
	r := dispatchRenderer(job)
	if r.err != nil {
		return
	}
	if err := r.renderer.BeginEdge(&Job{GVJ: job}); err != nil {
		r.setError(err)
	}
}

func endEdge(job *ccall.GVJ) {
	r := dispatchRenderer(job)
	if r.err != nil {
		return
	}
	if err := r.renderer.EndEdge(&Job{GVJ: job}); err != nil {
		r.setError(err)
	}
}

func beginAnchor(job *ccall.GVJ, href, tooltip, target, id string) {
	r := dispatchRenderer(job)
	if r.err != nil {
		return
	}
	if err := r.renderer.BeginAnchor(&Job{GVJ: job}, href, tooltip, target, id); err != nil {
		r.setError(err)
	}
}

func endAnchor(job *ccall.GVJ) {
	r := dispatchRenderer(job)
	if r.err != nil {
		return
	}
	if err := r.renderer.EndAnchor(&Job{GVJ: job}); err != nil {
		r.setError(err)
	}
}

func beginLabel(job *ccall.GVJ, typ int) {
	r := dispatchRenderer(job)
	if r.err != nil {
		return
	}
	if err := r.renderer.BeginLabel(&Job{GVJ: job}, typ); err != nil {
		r.setError(err)
	}
}

func endLabel(job *ccall.GVJ) {
	r := dispatchRenderer(job)
	if r.err != nil {
		return
	}
	if err := r.renderer.EndLabel(&Job{GVJ: job}); err != nil {
		r.setError(err)
	}
}

func textspan(job *ccall.GVJ, p ccall.Pointf, span *ccall.TextSpan) {
	r := dispatchRenderer(job)
	if r.err != nil {
		return
	}
	if err := r.renderer.TextSpan(&Job{GVJ: job}, Pointf{X: p.X, Y: p.Y}, &TextSpan{TextSpan: span}); err != nil {
		r.setError(err)
	}
}

func resolveColor(job *ccall.GVJ, r, g, b, a uint) {
	renderer := dispatchRenderer(job)
	if renderer.err != nil {
		return
	}
	if err := renderer.renderer.ResolveColor(&Job{GVJ: job}, Color{R: r, G: g, B: b, A: a}); err != nil {
		renderer.setError(err)
	}
}

func ellipse(job *ccall.GVJ, a0, a1 ccall.Pointf, filled int) {
	r := dispatchRenderer(job)
	if r.err != nil {
		return
	}
	if err := r.renderer.Ellipse(&Job{GVJ: job}, Pointf{X: a0.X, Y: a0.Y}, Pointf{X: a1.X, Y: a1.Y}, filled); err != nil {
		r.setError(err)
	}
}

func polygon(job *ccall.GVJ, a []ccall.Pointf, filled int) {
	r := dispatchRenderer(job)
	if r.err != nil {
		return
	}
	v := make([]Pointf, len(a))
	for idx, aa := range a {
		v[idx] = Pointf{X: aa.X, Y: aa.Y}
	}
	if err := r.renderer.Polygon(&Job{GVJ: job}, v, filled); err != nil {
		r.setError(err)
	}
}

func beziercurve(job *ccall.GVJ, a []ccall.Pointf, arrowAtStart, arrowAtEnd, ext int) {
	r := dispatchRenderer(job)
	if r.err != nil {
		return
	}
	v := make([]Pointf, len(a))
	for idx, aa := range a {
		v[idx] = Pointf{X: aa.X, Y: aa.Y}
	}
	if err := r.renderer.BezierCurve(&Job{GVJ: job}, v, arrowAtStart, arrowAtEnd); err != nil {
		r.setError(err)
	}
}

func polyline(job *ccall.GVJ, a []ccall.Pointf) {
	r := dispatchRenderer(job)
	if r.err != nil {
		return
	}
	v := make([]Pointf, len(a))
	for idx, aa := range a {
		v[idx] = Pointf{X: aa.X, Y: aa.Y}
	}
	if err := r.renderer.Polyline(&Job{GVJ: job}, v); err != nil {
		r.setError(err)
	}
}

func comment(job *ccall.GVJ, comment string) {
	r := dispatchRenderer(job)
	if r.err != nil {
		return
	}
	if err := r.renderer.Comment(&Job{GVJ: job}, comment); err != nil {
		r.setError(err)
	}
}

func libraryShape(job *ccall.GVJ, name string, a []ccall.Pointf, filled int) {
	r := dispatchRenderer(job)
	if r.err != nil {
		return
	}
	v := make([]Pointf, len(a))
	for idx, aa := range a {
		v[idx] = Pointf{X: aa.X, Y: aa.Y}
	}
	if err := r.renderer.LibraryShape(&Job{GVJ: job}, name, v, filled); err != nil {
		r.setError(err)
	}
}

func init() {
	ccall.BeginJob = beginJob
	ccall.EndJob = endJob
	ccall.BeginGraph = beginGraph
	ccall.EndGraph = endGraph
	ccall.BeginLayer = beginLayer
	ccall.EndLayer = endLayer
	ccall.BeginPage = beginPage
	ccall.EndPage = endPage
	ccall.BeginCluster = beginCluster
	ccall.EndCluster = endCluster
	ccall.BeginNodes = beginNodes
	ccall.EndNodes = endNodes
	ccall.BeginEdges = beginEdges
	ccall.EndEdges = endEdges
	ccall.BeginNode = beginNode
	ccall.EndNode = endNode
	ccall.BeginEdge = beginEdge
	ccall.EndEdge = endEdge
	ccall.BeginAnchor = beginAnchor
	ccall.EndAnchor = endAnchor
	ccall.BeginLabel = beginLabel
	ccall.EndLabel = endLabel
	ccall.Textspan = textspan
	ccall.ResolveColor = resolveColor
	ccall.Ellipse = ellipse
	ccall.Polygon = polygon
	ccall.Beziercurve = beziercurve
	ccall.Polyline = polyline
	ccall.Comment = comment
	ccall.LibraryShape = libraryShape
}
