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
#cgo LDFLAGS: -lm
#include "gvc.h"
#include "color.h"
#include "gvcjob.h"
*/
import "C"

var (
	BeginJob     func(*GVJ)
	EndJob       func(*GVJ)
	BeginGraph   func(*GVJ)
	EndGraph     func(*GVJ)
	BeginLayer   func(*GVJ, string, int, int)
	EndLayer     func(*GVJ)
	BeginPage    func(*GVJ)
	EndPage      func(*GVJ)
	BeginCluster func(*GVJ)
	EndCluster   func(*GVJ)
	BeginNodes   func(*GVJ)
	EndNodes     func(*GVJ)
	BeginEdges   func(*GVJ)
	EndEdges     func(*GVJ)
	BeginNode    func(*GVJ)
	EndNode      func(*GVJ)
	BeginEdge    func(*GVJ)
	EndEdge      func(*GVJ)
	BeginAnchor  func(*GVJ, string, string, string, string)
	EndAnchor    func(*GVJ)
	BeginLabel   func(*GVJ, int)
	EndLabel     func(*GVJ)
	Textspan     func(*GVJ, Pointf, *TextSpan)
	ResolveColor func(*GVJ, uint, uint, uint, uint)
	Ellipse      func(*GVJ, Pointf, Pointf, int)
	Polygon      func(*GVJ, []Pointf, int)
	Beziercurve  func(*GVJ, []Pointf, int, int, int)
	Polyline     func(*GVJ, []Pointf)
	Comment      func(*GVJ, string)
	LibraryShape func(*GVJ, string, []Pointf, int)
)

//export GoBeginJob
func GoBeginJob(job *C.GVJ_t) {
	if BeginJob != nil {
		BeginJob(ToGVJ(job))
	}
}

//export GoEndJob
func GoEndJob(job *C.GVJ_t) {
	if EndJob != nil {
		EndJob(ToGVJ(job))
	}
}

//export GoBeginGraph
func GoBeginGraph(job *C.GVJ_t) {
	if BeginGraph != nil {
		BeginGraph(ToGVJ(job))
	}
}

//export GoEndGraph
func GoEndGraph(job *C.GVJ_t) {
	if EndGraph != nil {
		EndGraph(ToGVJ(job))
	}
}

//export GoBeginLayer
func GoBeginLayer(job *C.GVJ_t, layername *C.char, layerNum C.int, numLayers C.int) {
	if BeginLayer != nil {
		BeginLayer(ToGVJ(job), C.GoString(layername), int(layerNum), int(numLayers))
	}
}

//export GoEndLayer
func GoEndLayer(job *C.GVJ_t) {
	if EndLayer != nil {
		EndLayer(ToGVJ(job))
	}
}

//export GoBeginPage
func GoBeginPage(job *C.GVJ_t) {
	if BeginPage != nil {
		BeginPage(ToGVJ(job))
	}
}

//export GoEndPage
func GoEndPage(job *C.GVJ_t) {
	if EndPage != nil {
		EndPage(ToGVJ(job))
	}
}

//export GoBeginCluster
func GoBeginCluster(job *C.GVJ_t) {
	if BeginCluster != nil {
		BeginCluster(ToGVJ(job))
	}
}

//export GoEndCluster
func GoEndCluster(job *C.GVJ_t) {
	if EndCluster != nil {
		EndCluster(ToGVJ(job))
	}
}

//export GoBeginNodes
func GoBeginNodes(job *C.GVJ_t) {
	if BeginNodes != nil {
		BeginNodes(ToGVJ(job))
	}
}

//export GoEndNodes
func GoEndNodes(job *C.GVJ_t) {
	if EndNodes != nil {
		EndNodes(ToGVJ(job))
	}
}

//export GoBeginEdges
func GoBeginEdges(job *C.GVJ_t) {
	if BeginEdges != nil {
		BeginEdges(ToGVJ(job))
	}
}

//export GoEndEdges
func GoEndEdges(job *C.GVJ_t) {
	if EndEdges != nil {
		EndEdges(ToGVJ(job))
	}
}

//export GoBeginNode
func GoBeginNode(job *C.GVJ_t) {
	if BeginNode != nil {
		BeginNode(ToGVJ(job))
	}
}

//export GoEndNode
func GoEndNode(job *C.GVJ_t) {
	if EndNode != nil {
		EndNode(ToGVJ(job))
	}
}

//export GoBeginEdge
func GoBeginEdge(job *C.GVJ_t) {
	if BeginEdge != nil {
		BeginEdge(ToGVJ(job))
	}
}

//export GoEndEdge
func GoEndEdge(job *C.GVJ_t) {
	if EndEdge != nil {
		EndEdge(ToGVJ(job))
	}
}

//export GoBeginAnchor
func GoBeginAnchor(job *C.GVJ_t, href, tooltip, target, id *C.char) {
	if BeginAnchor != nil {
		BeginAnchor(ToGVJ(job), C.GoString(href), C.GoString(tooltip), C.GoString(target), C.GoString(id))
	}
}

//export GoEndAnchor
func GoEndAnchor(job *C.GVJ_t) {
	if EndAnchor != nil {
		EndAnchor(ToGVJ(job))
	}
}

//export GoBeginLabel
func GoBeginLabel(job *C.GVJ_t, typ C.int) {
	if BeginLabel != nil {
		BeginLabel(ToGVJ(job), int(typ))
	}
}

//export GoEndLabel
func GoEndLabel(job *C.GVJ_t) {
	if EndLabel != nil {
		EndLabel(ToGVJ(job))
	}
}

//export GoTextspan
func GoTextspan(job *C.GVJ_t, p C.pointf, span *C.textspan_t) {
	if Textspan != nil {
		Textspan(ToGVJ(job), ToPointf(p), ToTextSpan(span))
	}
}

//export GoResolveColor
func GoResolveColor(job *C.GVJ_t, r, g, b, a C.uint) {
	if ResolveColor != nil {
		ResolveColor(ToGVJ(job), uint(r), uint(g), uint(b), uint(a))
	}
}

//export GoEllipse
func GoEllipse(job *C.GVJ_t, a0, a1 C.pointf, filled C.int) {
	if Ellipse != nil {
		Ellipse(ToGVJ(job), ToPointf(a0), ToPointf(a1), int(filled))
	}
}

//export GoPolygon
func GoPolygon(job *C.GVJ_t, a *C.pointf, n, filled C.int) {
	if Polygon != nil {
		Polygon(ToGVJ(job), ToPointsf(a, n), int(filled))
	}
}

//export GoBeziercurve
func GoBeziercurve(job *C.GVJ_t, a *C.pointf, n, arrowAtStart, arrowAtEnd, ext C.int) {
	if Beziercurve != nil {
		Beziercurve(ToGVJ(job), ToPointsf(a, n), int(arrowAtStart), int(arrowAtEnd), int(ext))
	}
}

//export GoPolyline
func GoPolyline(job *C.GVJ_t, a *C.pointf, n C.int) {
	if Polyline != nil {
		Polyline(ToGVJ(job), ToPointsf(a, n))
	}
}

//export GoComment
func GoComment(job *C.GVJ_t, comment *C.char) {
	if Comment != nil {
		Comment(ToGVJ(job), C.GoString(comment))
	}
}

//export GoLibraryShape
func GoLibraryShape(job *C.GVJ_t, name *C.char, a *C.pointf, n, filled C.int) {
	if LibraryShape != nil {
		LibraryShape(ToGVJ(job), C.GoString(name), ToPointsf(a, n), int(filled))
	}
}
