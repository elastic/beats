package cgraph

import "fmt"

type attribute string

const (
	dampingAttr            attribute = "Damping"
	kAttr                  attribute = "K"
	urlAttr                attribute = "URL"
	backgroundAttr         attribute = "_background"
	areaAttr               attribute = "area"
	arrowHeadAttr          attribute = "arrowhead"
	arrowSizeAttr          attribute = "arrowsize"
	arrowTailAttr          attribute = "arrowtail"
	bbAttr                 attribute = "bb"
	bgcolorAttr            attribute = "bgcolor"
	centerAttr             attribute = "center"
	charsetAttr            attribute = "charset"
	clusterRankAttr        attribute = "clusterrank"
	colorAttr              attribute = "color"
	colorSchemeAttr        attribute = "colorscheme"
	commentAttr            attribute = "comment"
	compoundAttr           attribute = "compound"
	concentrateAttr        attribute = "concentrate"
	constraintAttr         attribute = "constraint"
	decorateAttr           attribute = "decorate"
	defaultDistAttr        attribute = "defaultdist"
	dimAttr                attribute = "dim"
	dimenAttr              attribute = "dimen"
	dirAttr                attribute = "dir"
	dirEdgeConstraintsAttr attribute = "diredgeconstraints"
	distortionAttr         attribute = "distortion"
	dpiAttr                attribute = "dpi"
	edgeURLAttr            attribute = "edgeURL"
	edgeHrefAttr           attribute = "edgehref"
	edgeTargetAttr         attribute = "edgetarget"
	edgeTooltipAttr        attribute = "edgetooltip"
	epsilonAttr            attribute = "epsilon"
	esepAttr               attribute = "esep"
	fillColorAttr          attribute = "fillcolor"
	fixedSizeAttr          attribute = "fixedsize"
	fontColorAttr          attribute = "fontcolor"
	fontNameAttr           attribute = "fontname"
	fontNamesAttr          attribute = "fontnames"
	fontPathAttr           attribute = "fontpath"
	fontSizeAttr           attribute = "fontsize"
	forceLabelsAttr        attribute = "forcelabels"
	gradientAngleAttr      attribute = "gradientangle"
	groupAttr              attribute = "group"
	headURLAttr            attribute = "headURL"
	headLpAttr             attribute = "head_lp"
	headClipAttr           attribute = "headclip"
	headHrefAttr           attribute = "headhref"
	headLabelAttr          attribute = "headlabel"
	headPortAttr           attribute = "headport"
	headTargetAttr         attribute = "headtarget"
	headTooltipAttr        attribute = "headtooltip"
	heightAttr             attribute = "height"
	hrefAttr               attribute = "href"
	idAttr                 attribute = "id"
	imageAttr              attribute = "image"
	imagePathAttr          attribute = "imagepath"
	imagePosAttr           attribute = "imagepos"
	imageScaleAttr         attribute = "imagescale"
	inputScaleAttr         attribute = "inputscale"
	labelAttr              attribute = "label"
	labelURLAttr           attribute = "labelURL"
	labelSchemeAttr        attribute = "label_scheme"
	labelAngleAttr         attribute = "labelangle"
	labelDistanceAttr      attribute = "labeldistance"
	labelFloatAttr         attribute = "labelfloat"
	labelFontColorAttr     attribute = "labelfontcolor"
	labelFontNameAttr      attribute = "labelfontname"
	labelFontSizeAttr      attribute = "labelfontsize"
	labelHrefAttr          attribute = "labelhref"
	labelJustAttr          attribute = "labeljust"
	labelLocAttr           attribute = "labelloc"
	labelTargetAttr        attribute = "labeltarget"
	labelTooltipAttr       attribute = "labeltooltip"
	landscapeAttr          attribute = "landscape"
	layerAttr              attribute = "layer"
	layerListSepAttr       attribute = "layerlistsep"
	layersAttr             attribute = "layers"
	layerSelectAttr        attribute = "layerselect"
	layerSepAttr           attribute = "layersep"
	layoutAttr             attribute = "layout"
	lenAttr                attribute = "len"
	levelsAttr             attribute = "levels"
	levelsGapAttr          attribute = "levelsgap"
	lHeadAttr              attribute = "lhead"
	lHeightAttr            attribute = "lheight"
	lpAttr                 attribute = "lp"
	lTailAttr              attribute = "ltail"
	lWidthAttr             attribute = "lwidth"
	marginAttr             attribute = "margin"
	maxIterAttr            attribute = "maxiter"
	mcLimitAttr            attribute = "mclimit"
	minDistAttr            attribute = "mindist"
	minLenAttr             attribute = "minlen"
	modeAttr               attribute = "mode"
	modelAttr              attribute = "model"
	mosekAttr              attribute = "mosek"
	newRankAttr            attribute = "newrank"
	nodeSepAttr            attribute = "nodesep"
	noJustifyAttr          attribute = "nojustify"
	normalizeAttr          attribute = "normalize"
	noTranslateAttr        attribute = "notranslate"
	nsLimitAttr            attribute = "nslimit"
	nsLimit1Attr           attribute = "nslimit1"
	orderingAttr           attribute = "ordering"
	orientationAttr        attribute = "orientation"
	outputOrderAttr        attribute = "outputorder"
	overlapAttr            attribute = "overlap"
	overlapScalingAttr     attribute = "overlap_scaling"
	overlapShrinkAttr      attribute = "overlap_shrink"
	packAttr               attribute = "pack"
	packModeAttr           attribute = "packmode"
	padAttr                attribute = "pad"
	pageAttr               attribute = "page"
	pageDirAttr            attribute = "pagedir"
	penColorAttr           attribute = "pencolor"
	penWidthAttr           attribute = "penwidth"
	peripheriesAttr        attribute = "peripheries"
	pinAttr                attribute = "pin"
	posAttr                attribute = "pos"
	quadTreeAttr           attribute = "quadtree"
	quantumAttr            attribute = "quantum"
	rankAttr               attribute = "rank"
	rankDirAttr            attribute = "rankdir"
	rankSepAttr            attribute = "ranksep"
	ratioAttr              attribute = "ratio"
	rectsAttr              attribute = "rects"
	regularAttr            attribute = "regular"
	remincrossAttr         attribute = "remincross"
	repulsiveforceAttr     attribute = "repulsiveforce"
	resolutionAttr         attribute = "resolution"
	rootAttr               attribute = "root"
	rotateAttr             attribute = "rotate"
	rotationAttr           attribute = "rotation"
	sameHeadAttr           attribute = "samehead"
	sameTailAttr           attribute = "sametail"
	samplePointsAttr       attribute = "samplepoints"
	scaleAttr              attribute = "scale"
	searchSizeAttr         attribute = "searchsize"
	sepAttr                attribute = "sep"
	shapeAttr              attribute = "shape"
	shapeFileAttr          attribute = "shapefile"
	showBoxesAttr          attribute = "showboxes"
	sidesAttr              attribute = "sides"
	sizeAttr               attribute = "size"
	skewAttr               attribute = "skew"
	smoothingAttr          attribute = "smoothing"
	sortvAttr              attribute = "sortv"
	splinesAttr            attribute = "splines"
	startAttr              attribute = "start"
	styleAttr              attribute = "style"
	stylesheetAttr         attribute = "stylesheet"
	tailURLAttr            attribute = "tailURL"
	tailLpAttr             attribute = "tail_lp"
	tailClipAttr           attribute = "tailclip"
	tailHrefAttr           attribute = "tailhref"
	tailLabelAttr          attribute = "taillabel"
	tailPortAttr           attribute = "tailport"
	tailTargetAttr         attribute = "tailtarget"
	tailTooltipAttr        attribute = "tailtooltip"
	targetAttr             attribute = "target"
	tooltipAttr            attribute = "tooltip"
	trueColorAttr          attribute = "truecolor"
	verticesAttr           attribute = "vertices"
	viewportAttr           attribute = "viewport"
	voroMarginAttr         attribute = "voro_margin"
	weightAttr             attribute = "weight"
	widthAttr              attribute = "width"
	xdotVersionAttr        attribute = "xdotversion"
	xlabelAttr             attribute = "xlabel"
	xlpAttr                attribute = "xlp"
	zAttr                  attribute = "z"
)

var (
	trueStr  = toBoolString(true)
	falseStr = toBoolString(false)
)

func toBoolString(v bool) string {
	return fmt.Sprintf("%t", v)
}

// SetDamping
// Factor damping force motions.
// On each iteration, a nodes movement is limited to this factor of its potential motion.
// By being less than 1.0, the system tends to ``cool'', thereby preventing cycling.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:Damping
func (g *Graph) SetDamping(v float64) *Graph {
	g.SafeSet(string(dampingAttr), fmt.Sprint(v), "0.99")
	return g
}

// SetK
// Spring constant used in virtual physical model.
// It roughly corresponds to an ideal edge length (in inches), in that increasing K tends to increase the distance between nodes.
// Note that the edge attribute len can be used to override this value for adjacent nodes.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:K
func (g *Graph) SetK(v float64) *Graph {
	g.SafeSet(string(kAttr), fmt.Sprint(v), "0.3")
	return g
}

// SetURL
// Hyperlinks incorporated into device-dependent output.
// At present, used in ps2, cmap, i*map and svg formats.
// For all these formats, URLs can be attached to nodes, edges and clusters.
// URL attributes can also be attached to the root graph in ps2, cmap and i*map formats.
// This serves as the base URL for relative URLs in the former, and as the default image map file in the latter.
// For svg, cmapx and imap output, the active area for a node is its visible image.
// For example, an unfilled node with no drawn boundary will only be active on its label.
// For other output, the active area is its bounding box.
// The active area for a cluster is its bounding box.
// For edges, the active areas are small circles where the edge contacts its head and tail nodes.
// In addition, for svg, cmapx and imap, the active area includes a thin polygon approximating the edge.
// The circles may overlap the related node, and the edge URL dominates.
// If the edge has a label, this will also be active.
// Finally, if the edge has a head or tail label, this will also be active.
//
// Note that, for edges, the attributes headURL, tailURL, labelURL and edgeURL allow control of various parts of an edge.
// Also note that, if active areas of two edges overlap, it is unspecified which area dominates.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:URL
func (g *Graph) SetURL(v string) *Graph {
	g.SafeSet(string(urlAttr), v, "")
	return g
}

// SetURL
// Hyperlinks incorporated into device-dependent output.
// At present, used in ps2, cmap, i*map and svg formats.
// For all these formats, URLs can be attached to nodes, edges and clusters.
// URL attributes can also be attached to the root graph in ps2, cmap and i*map formats.
// This serves as the base URL for relative URLs in the former, and as the default image map file in the latter.
// For svg, cmapx and imap output, the active area for a node is its visible image.
// For example, an unfilled node with no drawn boundary will only be active on its label.
// For other output, the active area is its bounding box.
// The active area for a cluster is its bounding box.
// For edges, the active areas are small circles where the edge contacts its head and tail nodes.
// In addition, for svg, cmapx and imap, the active area includes a thin polygon approximating the edge.
// The circles may overlap the related node, and the edge URL dominates.
// If the edge has a label, this will also be active.
// Finally, if the edge has a head or tail label, this will also be active.
//
// Note that, for edges, the attributes headURL, tailURL, labelURL and edgeURL allow control of various parts of an edge.
// Also note that, if active areas of two edges overlap, it is unspecified which area dominates.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:URL
func (n *Node) SetURL(v string) *Node {
	n.SafeSet(string(urlAttr), v, "")
	return n
}

// SetURL
// Hyperlinks incorporated into device-dependent output.
// At present, used in ps2, cmap, i*map and svg formats.
// For all these formats, URLs can be attached to nodes, edges and clusters.
// URL attributes can also be attached to the root graph in ps2, cmap and i*map formats.
// This serves as the base URL for relative URLs in the former, and as the default image map file in the latter.
// For svg, cmapx and imap output, the active area for a node is its visible image.
// For example, an unfilled node with no drawn boundary will only be active on its label.
// For other output, the active area is its bounding box.
// The active area for a cluster is its bounding box.
// For edges, the active areas are small circles where the edge contacts its head and tail nodes.
// In addition, for svg, cmapx and imap, the active area includes a thin polygon approximating the edge.
// The circles may overlap the related node, and the edge URL dominates.
// If the edge has a label, this will also be active.
// Finally, if the edge has a head or tail label, this will also be active.
//
// Note that, for edges, the attributes headURL, tailURL, labelURL and edgeURL allow control of various parts of an edge.
// Also note that, if active areas of two edges overlap, it is unspecified which area dominates.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:URL
func (e *Edge) SetURL(v string) *Edge {
	e.SafeSet(string(urlAttr), v, "")
	return e
}

// SetBackground
// A string in the xdot format specifying an arbitrary background.
// During rendering, the canvas is first filled as described in the bgcolor attribute.
// Then, if _background is defined, the graphics operations described in the string are performed on the canvas.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:_background
func (g *Graph) SetBackground(v string) *Graph {
	g.SafeSet(string(backgroundAttr), v, "")
	return g
}

// SetArea
// Indicates the preferred area for a node or empty cluster when laid out by patchwork.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:area
func (n *Node) SetArea(v float64) *Node {
	n.SafeSet(string(areaAttr), fmt.Sprint(v), "1.0")
	return n
}

type ArrowType string

const (
	NormalArrow   ArrowType = "normal"
	InvArrow      ArrowType = "inv"
	DotArrow      ArrowType = "dot"
	InvDotArrow   ArrowType = "invdot"
	ODotArrow     ArrowType = "odot"
	InvODotArrow  ArrowType = "invodot"
	NoneArrow     ArrowType = "none"
	TeeArrow      ArrowType = "tee"
	EmptyArrow    ArrowType = "empty"
	InvEmptyArrow ArrowType = "invempty"
	DiamondArrow  ArrowType = "diamond"
	ODiamondArrow ArrowType = "odiamond"
	EDiamondArrow ArrowType = "ediamond"
	CrowArrow     ArrowType = "crow"
	BoxArrow      ArrowType = "box"
	OBoxArrow     ArrowType = "obox"
	OpenArrow     ArrowType = "open"
	HalfOpenArrow ArrowType = "halfopen"
	VeeArrow      ArrowType = "vee"
)

// SetArrowHead
// Style of arrowhead on the head node of an edge.
// This will only appear if the dir attribute is "forward" or "both".
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:arrowhead
func (e *Edge) SetArrowHead(v ArrowType) *Edge {
	e.SafeSet(string(arrowHeadAttr), string(v), string(NormalArrow))
	return e
}

// SetArrowSize
// Multiplicative scale factor for arrowheads.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:arrowsize
func (e *Edge) SetArrowSize(v float64) *Edge {
	e.SafeSet(string(arrowSizeAttr), fmt.Sprint(v), "1.0")
	return e
}

// SetArrowTail
// Style of arrowhead on the tail node of an edge.
// This will only appear if the dir attribute is "back" or "both".
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:arrowtail
func (e *Edge) SetArrowTail(v ArrowType) *Edge {
	e.SafeSet(string(arrowTailAttr), string(v), string(NormalArrow))
	return e
}

// SetBB
// Bounding box of drawing in points.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:bb
func (g *Graph) SetBB(llx, lly, urx, ury float64) *Graph {
	g.SafeSet(string(bbAttr), fmt.Sprintf("%f,%f,%f,%f", llx, lly, urx, ury), "")
	return g
}

// SetBackgroundColor
// When attached to the root graph, this color is used as the background for entire canvas.
// When a cluster attribute, it is used as the initial background for the cluster.
// If a cluster has a filled style, the cluster's fillcolor will overlay the background color.
// If the value is a colorList, a gradient fill is used.
// By default, this is a linear fill; setting style=radial will cause a radial fill.
// At present, only two colors are used.
// If the second color (after a colon) is missing, the default color is used for it.
// See also the gradientangle attribute for setting the gradient angle.
//
// For certain output formats, such as PostScript, no fill is done for the root graph unless bgcolor is explicitly set.
// For bitmap formats, however, the bits need to be initialized to something, so the canvas is filled with white by default.
// This means that if the bitmap output is included in some other document,
// all of the bits within the bitmap's bounding box will be set, overwriting whatever color or graphics were already on the page.
// If this effect is not desired, and you only want to set bits explicitly assigned in drawing the graph, set bgcolor="transparent".
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:bgcolor
func (g *Graph) SetBackgroundColor(v string) *Graph {
	g.SafeSet(string(bgcolorAttr), v, "")
	return g
}

// SetCenter
// If true, the drawing is centered in the output canvas.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:center
func (g *Graph) SetCenter(v bool) *Graph {
	g.SafeSet(string(centerAttr), toBoolString(v), falseStr)
	return g
}

// SetCharset
// Specifies the character encoding used when interpreting string input as a text label.
// The default value is "UTF-8".
// The other legal value is "iso-8859-1" or, equivalently, "Latin1".
// The charset attribute is case-insensitive.
// Note that if the character encoding used in the input does not match the charset value, the resulting output may be very strange.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:charset
func (g *Graph) SetCharset(v string) *Graph {
	g.SafeSet(string(charsetAttr), v, "UTF-8")
	return g
}

type ClusterMode string

const (
	LocalCluster  ClusterMode = "local"
	GlobalCluster ClusterMode = "global"
	NoneCluster   ClusterMode = "none"
)

// SetClusterRank
// Mode used for handling clusters.
// If clusterrank is "local", a subgraph whose name begins with "cluster" is given special treatment.
// The subgraph is laid out separately, and then integrated as a unit into its parent graph, with a bounding rectangle drawn about it.
// If the cluster has a label parameter, this label is displayed within the rectangle.
// Note also that there can be clusters within clusters.
// At present, the modes "global" and "none" appear to be identical, both turning off the special cluster processing.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:clusterrank
func (g *Graph) SetClusterRank(v ClusterMode) *Graph {
	g.SafeSet(string(clusterRankAttr), string(v), string(LocalCluster))
	return g
}

// SetColor
// Basic drawing color for graphics, not text.
// For the latter, use the fontcolor attribute.
// For edges, the value can either be a single color or a colorList.
// In the latter case, if colorList has no fractions,
// the edge is drawn using parallel splines or lines, one for each color in the list, in the order given.
// The head arrow, if any, is drawn using the first color in the list, and the tail arrow, if any, the second color.
// This supports the common case of drawing opposing edges, but using parallel splines instead of separately routed multiedges.
// If any fraction is used, the colors are drawn in series, with each color being given roughly its specified fraction of the edge.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:color
func (n *Node) SetColor(v string) *Node {
	n.SafeSet(string(colorAttr), v, "black")
	return n
}

// SetColor
// Basic drawing color for graphics, not text.
// For the latter, use the fontcolor attribute.
// For edges, the value can either be a single color or a colorList.
// In the latter case, if colorList has no fractions,
// the edge is drawn using parallel splines or lines, one for each color in the list, in the order given.
// The head arrow, if any, is drawn using the first color in the list, and the tail arrow, if any, the second color.
// This supports the common case of drawing opposing edges, but using parallel splines instead of separately routed multiedges.
// If any fraction is used, the colors are drawn in series, with each color being given roughly its specified fraction of the edge.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:color
func (e *Edge) SetColor(v string) *Edge {
	e.SafeSet(string(colorAttr), v, "black")
	return e
}

// SetColorScheme
// This attribute specifies a color scheme namespace.
// If defined, it specifies the context for interpreting color names.
// In particular, if a color value has form "xxx" or "//xxx", then the color xxx will be evaluated according to the current color scheme.
// If no color scheme is set, the standard X11 naming is used.
// For example, if colorscheme=bugn9, then color=7 is interpreted as "/bugn9/7".
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:colorscheme
func (g *Graph) SetColorScheme(v string) *Graph {
	g.SafeSet(string(colorSchemeAttr), v, "")
	return g
}

// SetColorScheme
// This attribute specifies a color scheme namespace.
// If defined, it specifies the context for interpreting color names.
// In particular, if a color value has form "xxx" or "//xxx", then the color xxx will be evaluated according to the current color scheme.
// If no color scheme is set, the standard X11 naming is used.
// For example, if colorscheme=bugn9, then color=7 is interpreted as "/bugn9/7".
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:colorscheme
func (n *Node) SetColorScheme(v string) *Node {
	n.SafeSet(string(colorSchemeAttr), v, "")
	return n
}

// SetColorScheme
// This attribute specifies a color scheme namespace.
// If defined, it specifies the context for interpreting color names.
// In particular, if a color value has form "xxx" or "//xxx", then the color xxx will be evaluated according to the current color scheme.
// If no color scheme is set, the standard X11 naming is used.
// For example, if colorscheme=bugn9, then color=7 is interpreted as "/bugn9/7".
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:colorscheme
func (e *Edge) SetColorScheme(v string) *Edge {
	e.SafeSet(string(colorSchemeAttr), v, "")
	return e
}

// SetComment
// Comments are inserted into output. Device-dependent
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:comment
func (g *Graph) SetComment(v string) *Graph {
	g.SafeSet(string(commentAttr), v, "")
	return g
}

// SetComment
// Comments are inserted into output. Device-dependent
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:comment
func (n *Node) SetComment(v string) *Node {
	n.SafeSet(string(commentAttr), v, "")
	return n
}

// SetComment
// Comments are inserted into output. Device-dependent
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:comment
func (e *Edge) SetComment(v string) *Edge {
	e.SafeSet(string(commentAttr), v, "")
	return e
}

// SetCompound
// If true, allow edges between clusters. (See lhead and ltail below.)
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:compound
func (g *Graph) SetCompound(v bool) *Graph {
	g.SafeSet(string(compoundAttr), toBoolString(v), falseStr)
	return g
}

// SetConcentrate
// If true, use edge concentrators.
// This merges multiedges into a single edge and causes partially parallel edges to share part of their paths.
// The latter feature is not yet available outside of dot.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:concentrate
func (g *Graph) SetConcentrate(v bool) *Graph {
	g.SafeSet(string(concentrateAttr), toBoolString(v), falseStr)
	return g
}

// SetConstraint
// If false, the edge is not used in ranking the nodes.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:constraint
func (e *Edge) SetConstraint(v bool) *Edge {
	e.SafeSet(string(constraintAttr), toBoolString(v), trueStr)
	return e
}

// SetDecorate
// If true, attach edge label to edge by a 2-segment polyline, underlining the label, then going to the closest point of spline.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:decorate
func (e *Edge) SetDecorate(v bool) *Edge {
	e.SafeSet(string(decorateAttr), toBoolString(v), falseStr)
	return e
}

// SetDefaultDist
// This specifies the distance between nodes in separate connected components.
// If set too small, connected components may overlap.
// Only applicable if pack=false.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:defaultdist
func (g *Graph) SetDefaultDist(v float64) *Graph {
	g.SafeSet(string(defaultDistAttr), fmt.Sprint(v), "1.0")
	return g
}

// SetDim
// Set the number of dimensions used for the layout.
// The maximum value allowed is 10.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:dim
func (g *Graph) SetDim(v int) *Graph {
	g.SafeSet(string(dimAttr), fmt.Sprint(v), "2")
	return g
}

// SetDimen
// Set the number of dimensions used for rendering.
// The maximum value allowed is 10.
// If both dimen and dim are set, the latter specifies the dimension used for layout, and the former for rendering.
// If only dimen is set, this is used for both layout and rendering dimensions.
// Note that, at present, all aspects of rendering are 2D.
// This includes the shape and size of nodes, overlap removal, and edge routing.
// Thus, for dimen > 2, the only valid information is the pos attribute of the nodes.
// All other coordinates will be 2D and, at best, will reflect a projection of a higher-dimensional point onto the plane.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:dimen
func (g *Graph) SetDimen(v int) *Graph {
	g.SafeSet(string(dimAttr), fmt.Sprint(v), "2")
	return g
}

type DirType string

const (
	ForwardDir DirType = "forward"
	BackDir    DirType = "back"
	BothDir    DirType = "both"
	NoneDir    DirType = "none"
)

// SetDir
// Set edge type for drawing arrowheads.
// This indicates which ends of the edge should be decorated with an arrowhead.
// The actual style of the arrowhead can be specified using the arrowhead and arrowtail attributes.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:dir
func (e *Edge) SetDir(v DirType) *Edge {
	e.SafeSet(string(dirAttr), string(v), string(ForwardDir))
	return e
}

// SetDirEdgeConstraints
// Only valid when mode="ipsep".
// If true, constraints are generated for each edge in the largest (heuristic) directed acyclic subgraph such that the edge must point downwards.
// If "hier", generates level constraints similar to those used with mode="hier".
// The main difference is that, in the latter case, only these constraints are involved, so a faster solver can be used.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:diredgeconstraints
func (g *Graph) SetDirEdgeConstraints(v string) *Graph {
	g.SafeSet(string(dirEdgeConstraintsAttr), v, falseStr)
	return g
}

// SetDistortion
// Distortion factor for shape=polygon.
// Positive values cause top part to be larger than bottom; negative values do the opposite.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:distortion
func (n *Node) SetDistortion(v float64) *Node {
	n.SafeSet(string(distortionAttr), fmt.Sprint(v), "0.0")
	return n
}

// SetDPI
// This specifies the expected number of pixels per inch on a display device.
// For bitmap output, this guarantees that text rendering will be done more accurately, both in size and in placement.
// For SVG output, it is used to guarantee that the dimensions in the output correspond to the correct number of points or inches.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:dpi
func (g *Graph) SetDPI(v float64) *Graph {
	g.SafeSet(string(dpiAttr), fmt.Sprint(v), "96.0")
	return g
}

// SetEdgeURL
// If edgeURL is defined, this is the link used for the non-label parts of an edge.
// This value overrides any URL defined for the edge.
// Also, this value is used near the head or tail node unless overridden by a headURL or tailURL value, respectively.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:edgeURL
func (e *Edge) SetEdgeURL(v string) *Edge {
	e.SafeSet(string(edgeURLAttr), v, "")
	return e
}

// SetEdgeHref
// Synonym for edgeURL.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:edgehref
func (e *Edge) SetEdgeHref(v string) *Edge {
	e.SafeSet(string(edgeHrefAttr), v, "")
	return e
}

// SetEdgeTarget
// If the edge has a URL or edgeURL attribute,
// this attribute determines which window of the browser is used for the URL attached to the non-label part of the edge.
// Setting it to "_graphviz" will open a new window if it doesn't already exist, or reuse it if it does.
// If undefined, the value of the target is used.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:edgetarget
func (e *Edge) SetEdgeTarget(v string) *Edge {
	e.SafeSet(string(edgeTargetAttr), v, "")
	return e
}

// SetEdgeTooltip
// Tooltip annotation attached to the non-label part of an edge.
// This is used only if the edge has a URL or edgeURL attribute.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:edgetooltip
func (e *Edge) SetEdgeTooltip(v string) *Edge {
	e.SafeSet(string(edgeTooltipAttr), v, "")
	return e
}

// SetEpsilon
// Terminating condition.
// If the length squared of all energy gradients are < epsilon, the algorithm stops.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:epsilon
func (g *Graph) SetEpsilon(v float64) *Graph {
	g.SafeSet(string(epsilonAttr), fmt.Sprint(v), ".0001")
	return g
}

// SetESep
// Margin used around polygons for purposes of spline edge routing.
// The interpretation is the same as given for sep.
// This should normally be strictly less than sep.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:esep
func (g *Graph) SetESep(v float64) *Graph {
	g.SafeSet(string(esepAttr), fmt.Sprintf("+%f", v), "+3")
	return g
}

// SetFillColor
// Color used to fill the background of a node or cluster assuming style=filled, or a filled arrowhead.
// If fillcolor is not defined, color is used. (For clusters, if color is not defined, bgcolor is used.)
// If this is not defined, the default is used, except for shape=point or when the output format is MIF, which use black by default.
// If the value is a colorList, a gradient fill is used. By default, this is a linear fill; setting style=radial will cause a radial fill.
// At present, only two colors are used.
// If the second color (after a colon) is missing, the default color is used for it.
// See also the gradientangle attribute for setting the gradient angle.
//
// Note that a cluster inherits the root graph's attributes if defined.
// Thus, if the root graph has defined a fillcolor, this will override a color or bgcolor attribute set for the cluster.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:fillcolor
func (n *Node) SetFillColor(v string) *Node {
	n.SafeSet(string(fillColorAttr), v, "lightgrey")
	return n
}

// SetFixedSize
// If false, the size of a node is determined by smallest width and height needed to contain its label and image,
// if any, with a margin specified by the margin attribute.
// The width and height must also be at least as large as the sizes specified by the width and height attributes, which specify the minimum values for these parameters.
// If true, the node size is specified by the values of the width and height attributes only and is not expanded to contain the text label.
// There will be a warning if the label (with margin) cannot fit within these limits.
//
// If the fixedsize attribute is set to shape,
// the width and height attributes also determine the size of the node shape, but the label can be much larger.
// Both the label and shape sizes are used when avoiding node overlap,
// but all edges to the node ignore the label and only contact the node shape.
// No warning is given if the label is too large.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:fixedsize
func (n *Node) SetFixedSize(v bool) *Node {
	n.SafeSet(string(fixedSizeAttr), toBoolString(v), falseStr)
	return n
}

// SetFontColor
// Color used for text.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:fontcolor
func (g *Graph) SetFontColor(v string) *Graph {
	g.SafeSet(string(fontColorAttr), v, "black")
	return g
}

// SetFontColor
// Color used for text.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:fontcolor
func (n *Node) SetFontColor(v string) *Node {
	n.SafeSet(string(fontColorAttr), v, "black")
	return n
}

// SetFontColor
// Color used for text.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:fontcolor
func (e *Edge) SetFontColor(v string) *Edge {
	e.SafeSet(string(fontColorAttr), v, "black")
	return e
}

// SetFontSize
// Font size, in points, used for text.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:fontsize
func (g *Graph) SetFontSize(v float64) *Graph {
	g.SafeSet(string(fontSizeAttr), fmt.Sprint(v), "14.0")
	return g
}

// SetFontSize
// Font size, in points, used for text.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:fontsize
func (n *Node) SetFontSize(v float64) *Node {
	n.SafeSet(string(fontSizeAttr), fmt.Sprint(v), "14.0")
	return n
}

// SetFontSize
// Font size, in points, used for text.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:fontsize
func (e *Edge) SetFontSize(v float64) *Edge {
	e.SafeSet(string(fontSizeAttr), fmt.Sprint(v), "14.0")
	return e
}

// SetForceLabels
// If true, all xlabel attributes are placed, even if there is some overlap with nodes or other labels.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:forcelabels
func (g *Graph) SetForceLabels(v bool) *Graph {
	g.SafeSet(string(forceLabelsAttr), toBoolString(v), trueStr)
	return g
}

// SetGradientAngle
// If a gradient fill is being used, this determines the angle of the fill.
// For linear fills, the colors transform along a line specified by the angle and the center of the object.
// For radial fills, a value of zero causes the colors to transform radially from the center;
// for non-zero values, the colors transform from a point near the object's periphery as specified by the value.
// If unset, the default angle is 0.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:gradientangle
func (g *Graph) SetGradientAngle(v int) *Graph {
	g.SafeSet(string(gradientAngleAttr), fmt.Sprint(v), "")
	return g
}

// SetGradientAngle
// If a gradient fill is being used, this determines the angle of the fill.
// For linear fills, the colors transform along a line specified by the angle and the center of the object.
// For radial fills, a value of zero causes the colors to transform radially from the center;
// for non-zero values, the colors transform from a point near the object's periphery as specified by the value.
// If unset, the default angle is 0.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:gradientangle
func (n *Node) SetGradientAngle(v int) *Node {
	n.SafeSet(string(gradientAngleAttr), fmt.Sprint(v), "")
	return n
}

// SetGroup
// If the end points of an edge belong to the same group,
// i.e., have the same group attribute, parameters are set to avoid crossings and keep the edges straight.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:group
func (n *Node) SetGroup(v string) *Node {
	n.SafeSet(string(groupAttr), v, "")
	return n
}

// SetHeadURL
// If headURL is defined, it is output as part of the head label of the edge.
// Also, this value is used near the head node, overriding any URL value.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:headURL
func (e *Edge) SetHeadURL(v string) *Edge {
	e.SafeSet(string(headURLAttr), v, "")
	return e
}

// SetHeadLabelPoint
// Position of an edge's head label, in points.
// The position indicates the center of the label.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:head_lp
func (e *Edge) SetHeadLabelPoint(x, y float64) *Edge {
	e.SafeSet(string(headLpAttr), fmt.Sprintf("%f,%f", x, y), "")
	return e
}

// SetHeadClip
// If true, the head of an edge is clipped to the boundary of the head node;
// otherwise, the end of the edge goes to the center of the node, or the center of a port, if applicable.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:headclip
func (e *Edge) SetHeadClip(v bool) *Edge {
	e.SafeSet(string(headClipAttr), toBoolString(v), trueStr)
	return e
}

// SetHeadHref
// Synonym for headURL.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:headhref
func (e *Edge) SetHeadHref(v string) *Edge {
	e.SafeSet(string(headHrefAttr), v, "")
	return e
}

// SetHeadLabel
// Text label to be placed near head of edge
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:headlabel
func (e *Edge) SetHeadLabel(v string) *Edge {
	e.SafeSet(string(headLabelAttr), v, "")
	return e
}

// SetHeadPort
// Indicates where on the head node to attach the head of the edge.
// In the default case, the edge is aimed towards the center of the node, and then clipped at the node boundary.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:headport
func (e *Edge) SetHeadPort(v string) *Edge {
	e.SafeSet(string(headPortAttr), v, "")
	return e
}

// SetHeadTarget
// If the edge has a headURL, this attribute determines which window of the browser is used for the URL.
// Setting it to "_graphviz" will open a new window if it doesn't already exist,
// or reuse it if it does. If undefined, the value of the target is used.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:headtarget
func (e *Edge) SetHeadTarget(v string) *Edge {
	e.SafeSet(string(headTargetAttr), v, "")
	return e
}

// SetHeadTooltip
// Tooltip annotation attached to the head of an edge.
// This is used only if the edge has a headURL attribute.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:headtooltip
func (e *Edge) SetHeadTooltip(v string) *Edge {
	e.SafeSet(string(headTooltipAttr), v, "")
	return e
}

// SetHeight
// Height of node, in inches.
// This is taken as the initial, minimum height of the node.
// If fixedsize is true, this will be the final height of the node.
// Otherwise, if the node label requires more height to fit, the node's height will be increased to contain the label.
// Note also that, if the output format is dot, the value given to height will be the final value.
// If the node shape is regular, the width and height are made identical.
// In this case, if either the width or the height is set explicitly, that value is used.
// In this case, if both the width or the height are set explicitly, the maximum of the two values is used.
// If neither is set explicitly, the minimum of the two default values is used.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:height
func (n *Node) SetHeight(v float64) *Node {
	n.SafeSet(string(heightAttr), fmt.Sprint(v), "0.5")
	return n
}

// SetHref
// Synonym for URL.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:href
func (g *Graph) SetHref(v string) *Graph {
	g.SafeSet(string(hrefAttr), v, "")
	return g
}

// SetHref
// Synonym for URL.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:href
func (n *Node) SetHref(v string) *Node {
	n.SafeSet(string(hrefAttr), v, "")
	return n
}

// SetHref
// Synonym for URL.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:href
func (e *Edge) SetHref(v string) *Edge {
	e.SafeSet(string(hrefAttr), v, "")
	return e
}

// SetID
// Allows the graph author to provide an id for graph objects which is to be included in the output.
// Normal "\N", "\E", "\G" substitutions are applied.
// If provided, it is the responsibility of the provider to keep its values sufficiently unique for its intended downstream use.
// Note, in particular, that "\E" does not provide a unique id for multi-edges.
// If no id attribute is provided, then a unique internal id is used.
// However, this value is unpredictable by the graph writer.
// An externally provided id is not used internally.
// If the graph provides an id attribute, this will be used as a prefix for internally generated attributes.
// By making these distinct, the user can include multiple image maps in the same document.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:id
func (g *Graph) SetID(v string) *Graph {
	g.SafeSet(string(idAttr), v, "")
	return g
}

// SetID
// Allows the graph author to provide an id for graph objects which is to be included in the output.
// Normal "\N", "\E", "\G" substitutions are applied.
// If provided, it is the responsibility of the provider to keep its values sufficiently unique for its intended downstream use.
// Note, in particular, that "\E" does not provide a unique id for multi-edges.
// If no id attribute is provided, then a unique internal id is used.
// However, this value is unpredictable by the graph writer.
// An externally provided id is not used internally.
// If the graph provides an id attribute, this will be used as a prefix for internally generated attributes.
// By making these distinct, the user can include multiple image maps in the same document.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:id
func (n *Node) SetID(v string) *Node {
	n.SafeSet(string(idAttr), v, "")
	return n
}

// SetID
// Allows the graph author to provide an id for graph objects which is to be included in the output.
// Normal "\N", "\E", "\G" substitutions are applied.
// If provided, it is the responsibility of the provider to keep its values sufficiently unique for its intended downstream use.
// Note, in particular, that "\E" does not provide a unique id for multi-edges.
// If no id attribute is provided, then a unique internal id is used.
// However, this value is unpredictable by the graph writer.
// An externally provided id is not used internally.
// If the graph provides an id attribute, this will be used as a prefix for internally generated attributes.
// By making these distinct, the user can include multiple image maps in the same document.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:id
func (e *Edge) SetID(v string) *Edge {
	e.SafeSet(string(idAttr), v, "")
	return e
}

// SetImage
// Gives the name of a file containing an image to be displayed inside a node.
// The image file must be in one of the recognized formats,
// typically JPEG, PNG, GIF, BMP, SVG or Postscript, and be able to be converted into the desired output format.
// The file must contain the image size information.
// This is usually trivially true for the bitmap formats.
// For PostScript, the file must contain a line starting with %%BoundingBox:
// followed by four integers specifying the lower left x and y coordinates
// and the upper right x and y coordinates of the bounding box for the image, the coordinates being in points.
// An SVG image file must contain width and height attributes, typically as part of the svg element.
// The values for these should have the form of a floating point number, followed by optional units,
// e.g., width="76pt".
// Recognized units are in, px, pc, pt, cm and mm for inches, pixels, picas, points, centimeters and millimeters, respectively.
// The default unit is points.
//
// Unlike with the shapefile attribute, the image is treated as node content rather than the entire node.
// In particular, an image can be contained in a node of any shape, not just a rectangle.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:image
func (n *Node) SetImage(v string) *Node {
	n.SafeSet(string(imageAttr), v, "")
	return n
}

// SetImagePath
// Specifies a list of directories in which to look for image files as specified by the image attribute or using the IMG element in HTML-like labels.
// The string should be a list of (absolute or relative) pathnames, each separated by a semicolon (for Windows) or a colon (all other OS).
// The first directory in which a file of the given name is found will be used to load the image.
// If imagepath is not set, relative pathnames for the image file will be interpreted with respect to the current working directory.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:imagepath
func (g *Graph) SetImagePath(v string) *Graph {
	g.SafeSet(string(imagePathAttr), v, "")
	return g
}

type ImagePos string

const (
	TopLeftPos        ImagePos = "tl"
	TopCenteredPos    ImagePos = "tc"
	TopRightPos       ImagePos = "tr"
	MiddleLeftPos     ImagePos = "ml"
	MiddleCenteredPos ImagePos = "mc"
	BottomLeftPos     ImagePos = "bl"
	BottomCenteredPos ImagePos = "bc"
	BottomRightPos    ImagePos = "br"
)

// SetImagePos
// Attribute controlling how an image is positioned within its containing node.
// This only has an effect when the image is smaller than the containing node.
// The default is to be centered both horizontally and vertically.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:imagepos
func (n *Node) SetImagePos(v ImagePos) *Node {
	n.SafeSet(string(imagePosAttr), string(v), string(MiddleCenteredPos))
	return n
}

// SetImageScale
// Attribute controlling how an image fills its containing node. In general, the image is given its natural size, (cf. dpi), and the node size is made large enough to contain its image, its label, its margin, and its peripheries. Its width and height will also be at least as large as its minimum width and height. If, however, fixedsize=true, the width and height attributes specify the exact size of the node.
// During rendering, in the default case (imagescale=false), the image retains its natural size. If imagescale=true, the image is uniformly scaled (i.e., its aspect ratio is preserved) to fit inside the node. At least one dimension of the image will be as large as possible given the size of the node. When imagescale=width, the width of the image is scaled to fill the node width. The corresponding property holds when imagescale=height. When imagescale=both, both the height and the width are scaled separately to fill the node.
//
// In all cases, if a dimension of the image is larger than the corresponding dimension of the node, that dimension of the image is scaled down to fit the node. As with the case of expansion, if imagescale=true, width and height are scaled uniformly.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:imagescale
func (n *Node) SetImageScale(v bool) *Node {
	n.SafeSet(string(imageScaleAttr), toBoolString(v), falseStr)
	return n
}

// SetInputScale
// For layout algorithms that support initial input positions (specified by the pos attribute),
// this attribute can be used to appropriately scale the values.
// By default, fdp and neato interpret the x and y values of pos as being in inches.
// (NOTE: neato -n(2) treats the coordinates as being in points, being the unit used by the layout algorithms for the pos attribute.)
// Thus, if the graph has pos attributes in points, one should set inputscale=72.
// This can also be set on the command line using the -s flag flag.
// If not set, no scaling is done and the units on input are treated as inches.
// A value of 0 is equivalent to inputscale=72.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:inputscale
func (g *Graph) SetInputScale(v float64) *Graph {
	g.SafeSet(string(inputScaleAttr), fmt.Sprint(v), "")
	return g
}

// SetLabel
// Text label attached to objects.
// If a node's shape is record, then the label can have a special format which describes the record layout.
// Note that a node's default label is "\N", so the node's name or ID becomes its label.
// Technically, a node's name can be an HTML string but this will not mean that the node's label will be interpreted as an HTML-like label.
// This is because the node's actual label is an ordinary string, which will be replaced by the raw bytes stored in the node's name.
// To get an HTML-like label, the label attribute value itself must be an HTML string.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:label
func (g *Graph) SetLabel(v string) *Graph {
	g.SafeSet(string(labelAttr), v, "")
	return g
}

// SetLabel
// Text label attached to objects.
// If a node's shape is record, then the label can have a special format which describes the record layout.
// Note that a node's default label is "\N", so the node's name or ID becomes its label.
// Technically, a node's name can be an HTML string but this will not mean that the node's label will be interpreted as an HTML-like label.
// This is because the node's actual label is an ordinary string, which will be replaced by the raw bytes stored in the node's name.
// To get an HTML-like label, the label attribute value itself must be an HTML string.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:label
func (n *Node) SetLabel(v string) *Node {
	n.SafeSet(string(labelAttr), v, "\\N")
	return n
}

// SetLabel
// Text label attached to objects.
// If a node's shape is record, then the label can have a special format which describes the record layout.
// Note that a node's default label is "\N", so the node's name or ID becomes its label.
// Technically, a node's name can be an HTML string but this will not mean that the node's label will be interpreted as an HTML-like label.
// This is because the node's actual label is an ordinary string, which will be replaced by the raw bytes stored in the node's name.
// To get an HTML-like label, the label attribute value itself must be an HTML string.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:label
func (e *Edge) SetLabel(v string) *Edge {
	e.SafeSet(string(labelAttr), v, "")
	return e
}

// SetLabelURL
// If labelURL is defined, this is the link used for the label of an edge.
// This value overrides any URL defined for the edge.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:labelURL
func (e *Edge) SetLabelURL(v string) *Edge {
	e.SafeSet(string(labelURLAttr), v, "")
	return e
}

// SetLabelScheme
// The value indicates whether to treat a node whose name has the form |edgelabel|* as a special node representing an edge label.
// The default (0) produces no effect.
// If the attribute is set to 1, sfdp uses a penalty-based method to make that kind of node close to the center of its neighbor.
// With a value of 2, sfdp uses a penalty-based method to make that kind of node close to the old center of its neighbor.
// Finally, a value of 3 invokes a two-step process of overlap removal and straightening.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:label_scheme
func (g *Graph) SetLabelScheme(v int) *Graph {
	g.SafeSet(string(labelSchemeAttr), fmt.Sprint(v), "0")
	return g
}

// SetLabelAngle
// This, along with labeldistance, determine where the headlabel (taillabel) are placed with respect to the head (tail) in polar coordinates.
// The origin in the coordinate system is the point where the edge touches the node.
// The ray of 0 degrees goes from the origin back along the edge, parallel to the edge at the origin.
// The angle, in degrees, specifies the rotation from the 0 degree ray,
// with positive angles moving counterclockwise and negative angles moving clockwise.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:labelangle
func (e *Edge) SetLabelAngle(v float64) *Edge {
	e.SafeSet(string(labelAngleAttr), fmt.Sprint(v), "-25.0")
	return e
}

// SetLabelDistance
// Multiplicative scaling factor adjusting the distance that the headlabel(taillabel) is from the head(tail) node.
// The default distance is 10 points. See labelangle for more details.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:labeldistance
func (e *Edge) SetLabelDistance(v float64) *Edge {
	e.SafeSet(string(labelDistanceAttr), fmt.Sprint(v), "1.0")
	return e
}

// SetLabelFloat
// If true, allows edge labels to be less constrained in position.
// In particular, it may appear on top of other edges.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:labelfloat
func (e *Edge) SetLabelFloat(v bool) *Edge {
	e.SafeSet(string(labelFloatAttr), toBoolString(v), falseStr)
	return e
}

// SetLabelFontColor
// Color used for headlabel and taillabel.
// If not set, defaults to edge's fontcolor.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:labelfontcolor
func (e *Edge) SetLabelFontColor(v string) *Edge {
	e.SafeSet(string(labelFontColorAttr), v, "black")
	return e
}

// SetLabelFontSize
// Font size, in points, used for headlabel and taillabel.
// If not set, defaults to edge's fontsize.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:labelfontsize
func (e *Edge) SetLabelFontSize(v float64) *Edge {
	e.SafeSet(string(labelFontSizeAttr), fmt.Sprint(v), "14.0")
	return e
}

// SetLabelHref
// Synonym for labelURL.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:labelhref
func (e *Edge) SetLabelHref(v string) *Edge {
	e.SafeSet(string(labelHrefAttr), v, "")
	return e
}

type JustType string

const (
	LeftJust     JustType = "l"
	CenteredJust JustType = "c"
	RightJust    JustType = "r"
)

// SetLabelJust
// Justification for cluster labels.
// If "r", the label is right-justified within bounding rectangle;
// if "l", left-justified; else the label is centered.
// Note that a subgraph inherits attributes from its parent.
// Thus, if the root graph sets labeljust to "l", the subgraph inherits this value.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:labeljust
func (g *Graph) SetLabelJust(v JustType) *Graph {
	g.SafeSet(string(labelJustAttr), string(v), string(CenteredJust))
	return g
}

type LabelLocation string

const (
	TopLocation      LabelLocation = "t"
	CenteredLocation LabelLocation = "c"
	BottomLocation   LabelLocation = "b"
)

// SetLabelLocation
// Vertical placement of labels for nodes, root graphs and clusters.
// For graphs and clusters, only "t" and "b" are allowed, corresponding to placement at the top and bottom, respectively.
// By default, root graph labels go on the bottom and cluster labels go on the top.
// Note that a subgraph inherits attributes from its parent.
// Thus, if the root graph sets labelloc to "b", the subgraph inherits this value.
//
// For nodes, this attribute is used only when the height of the node is larger than the height of its label.
// If labelloc is set to "t", "c", or "b", the label is aligned with the top, centered, or aligned with the bottom of the node, respectively.
// In the default case, the label is vertically centered.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:labelloc
func (g *Graph) SetLabelLocation(v LabelLocation) *Graph {
	g.SafeSet(string(labelLocAttr), string(v), string(BottomLocation))
	return g
}

// SetLabelLocation
// Vertical placement of labels for nodes, root graphs and clusters.
// For graphs and clusters, only "t" and "b" are allowed, corresponding to placement at the top and bottom, respectively.
// By default, root graph labels go on the bottom and cluster labels go on the top.
// Note that a subgraph inherits attributes from its parent.
// Thus, if the root graph sets labelloc to "b", the subgraph inherits this value.
//
// For nodes, this attribute is used only when the height of the node is larger than the height of its label.
// If labelloc is set to "t", "c", or "b", the label is aligned with the top, centered, or aligned with the bottom of the node, respectively.
// In the default case, the label is vertically centered.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:labelloc
func (n *Node) SetLabelLocation(v LabelLocation) *Node {
	n.SafeSet(string(labelLocAttr), string(v), string(CenteredLocation))
	return n
}

// SetLabelTarget
// If the edge has a URL or labelURL attribute,
// this attribute determines which window of the browser is used for the URL attached to the label.
// Setting it to "_graphviz" will open a new window if it doesn't already exist, or reuse it if it does.
// If undefined, the value of the target is used.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:labeltarget
func (e *Edge) SetLabelTarget(v string) *Edge {
	e.SafeSet(string(labelTargetAttr), v, "")
	return e
}

// SetLabelTooltip
// Tooltip annotation attached to label of an edge.
// This is used only if the edge has a URL or labelURL attribute.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:labeltooltip
func (e *Edge) SetLabelTooltip(v string) *Edge {
	e.SafeSet(string(labelTooltipAttr), v, "")
	return e
}

// SetLandscape
// If true, the graph is rendered in landscape mode.
// Synonymous with rotate=90 or orientation=landscape.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:landscape
func (g *Graph) SetLandscape(v bool) *Graph {
	g.SafeSet(string(landscapeAttr), toBoolString(v), falseStr)
	return g
}

// SetLayer
// Specifies layers in which the node, edge or cluster is present.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:layer
func (n *Node) SetLayer(v string) *Node {
	n.SafeSet(string(layerAttr), v, "")
	return n
}

// SetLayer
// Specifies layers in which the node, edge or cluster is present.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:layer
func (e *Edge) SetLayer(v string) *Edge {
	e.SafeSet(string(layerAttr), v, "")
	return e
}

// SetLayerListSeparator
// Specifies the separator characters used to split an attribute of type layerRange into a list of ranges.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:layerlistsep
func (g *Graph) SetLayerListSeparator(v string) *Graph {
	g.SafeSet(string(layerListSepAttr), v, ",")
	return g
}

// SetLayers
// Specifies a linearly ordered list of layer names attached to the graph The graph is then output in separate layers.
// Only those components belonging to the current output layer appear.
// For more information, see the page How to use drawing layers (overlays).
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:layers
func (g *Graph) SetLayers(v string) *Graph {
	g.SafeSet(string(layersAttr), v, "")
	return g
}

// SetLayerSelect
// Selects a list of layers to be emitted.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:layerselect
func (g *Graph) SetLayerSelect(v string) *Graph {
	g.SafeSet(string(layerSelectAttr), v, "")
	return g
}

// SetLayerSeparator
// Specifies the separator characters used to split the layers attribute into a list of layer names.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:layersep
func (g *Graph) SetLayerSeparator(v string) *Graph {
	g.SafeSet(string(layerSepAttr), v, ":\\t")
	return g
}

// SetLayout
// Specifies the name of the layout algorithm to use, such as "dot" or "neato".
// Normally, graphs should be kept independent of a type of layout.
// In some cases, however, it can be convenient to embed the type of layout desired within the graph.
// For example, a graph containing position information from a layout might want to record what the associated layout algorithm was.
// This attribute takes precedence over the -K flag or the actual command name used.
func (g *Graph) SetLayout(v string) *Graph {
	g.SafeSet(string(layoutAttr), v, "")
	return g
}

// SetLen
// Preferred edge length, in inches.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:len
func (e *Edge) SetLen(v float64) *Edge {
	e.SafeSet(string(lenAttr), fmt.Sprint(v), "1.0")
	return e
}

const (
	maxUint = ^uint(0)
	maxInt  = int(maxUint >> 1)
)

// SetLevels
// Number of levels allowed in the multilevel scheme.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:levels
func (g *Graph) SetLevels(v int) *Graph {
	g.SafeSet(string(levelsAttr), fmt.Sprint(v), fmt.Sprint(maxInt))
	return g
}

// SetLevelsGap
// Specifies strictness of level constraints in neato when mode="ipsep" or "hier".
// Larger positive values mean stricter constraints, which demand more separation between levels.
// On the other hand, negative values will relax the constraints by allowing some overlap between the levels.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:levelsgap
func (g *Graph) SetLevelsGap(v float64) *Graph {
	g.SafeSet(string(levelsGapAttr), fmt.Sprint(v), "0.0")
	return g
}

// SetLogicalHead
// Logical head of an edge.
// When compound is true, if lhead is defined and is the name of a cluster containing the real head,
// the edge is clipped to the boundary of the cluster.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:lhead
func (e *Edge) SetLogicalHead(v string) *Edge {
	e.SafeSet(string(lHeadAttr), v, "")
	return e
}

// SetLabelHeight
// Height of graph or cluster label, in inches.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:lheight
func (e *Edge) SetLabelHeight(v float64) *Edge {
	e.SafeSet(string(lHeightAttr), fmt.Sprint(v), "")
	return e
}

// SetLabelPosition
// Label position, in points. The position indicates the center of the label.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:lp
func (g *Graph) SetLabelPosition(x, y float64) *Graph {
	g.SafeSet(string(lpAttr), fmt.Sprintf("%f,%f", x, y), "")
	return g
}

// SetLabelPosition
// Label position, in points. The position indicates the center of the label.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:lp
func (e *Edge) SetLabelPosition(x, y float64) *Edge {
	e.SafeSet(string(lpAttr), fmt.Sprintf("%f,%f", x, y), "")
	return e
}

// SetLogicalTail
// Logical tail of an edge.
// When compound is true, if ltail is defined and is the name of a cluster containing the real tail,
// the edge is clipped to the boundary of the cluster
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:ltail
func (e *Edge) SetLogicalTail(v string) *Edge {
	e.SafeSet(string(lTailAttr), v, "")
	return e
}

// SetLabelWidth
// Width of graph or cluster label, in inches.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:lwidth
func (g *Graph) SetLabelWidth(v float64) *Graph {
	g.SafeSet(string(lWidthAttr), fmt.Sprint(v), "")
	return g
}

// SetMargin
// For graphs, this sets x and y margins of canvas, in inches.
// If the margin is a single double, both margins are set equal to the given value.
// Note that the margin is not part of the drawing but just empty space left around the drawing.
// It basically corresponds to a translation of drawing, as would be necessary to center a drawing on a page.
// Nothing is actually drawn in the margin.
// To actually extend the background of a drawing, see the pad attribute.
//
// For clusters, this specifies the space between the nodes in the cluster and the cluster bounding box.
// By default, this is 8 points.
//
// For nodes, this attribute specifies space left around the node's label.
// By default, the value is 0.11,0.055.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:margin
func (g *Graph) SetMargin(v float64) *Graph {
	g.SafeSet(string(marginAttr), fmt.Sprint(v), "")
	return g
}

// SetMargin
// For graphs, this sets x and y margins of canvas, in inches.
// If the margin is a single double, both margins are set equal to the given value.
// Note that the margin is not part of the drawing but just empty space left around the drawing.
// It basically corresponds to a translation of drawing, as would be necessary to center a drawing on a page.
// Nothing is actually drawn in the margin.
// To actually extend the background of a drawing, see the pad attribute.
//
// For clusters, this specifies the space between the nodes in the cluster and the cluster bounding box.
// By default, this is 8 points.
//
// For nodes, this attribute specifies space left around the node's label.
// By default, the value is 0.11,0.055.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:margin
func (n *Node) SetMargin(v float64) *Node {
	n.SafeSet(string(marginAttr), fmt.Sprint(v), "")
	return n
}

// SetMaxIterator
// Sets the number of iterations used.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:maxiter
func (g *Graph) SetMaxIterator(v int) *Graph {
	g.SafeSet(string(maxIterAttr), fmt.Sprint(v), "200")
	return g
}

// SetMCLimit
// Multiplicative scale factor used to alter the MinQuit (default = 8) and MaxIter (default = 24) parameters used during crossing minimization.
// These correspond to the number of tries without improvement before quitting and the maximum number of iterations in each pass.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:mclimit
func (g *Graph) SetMCLimit(v float64) *Graph {
	g.SafeSet(string(mcLimitAttr), fmt.Sprint(v), "1.0")
	return g
}

// SetMinDist
// Specifies the minimum separation between all nodes.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:mindist
func (g *Graph) SetMinDist(v float64) *Graph {
	g.SafeSet(string(minDistAttr), fmt.Sprint(v), "1.0")
	return g
}

// SetMinLen
// Minimum edge length (rank difference between head and tail).
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:minlen
func (e *Edge) SetMinLen(v int) *Edge {
	e.SafeSet(string(minLenAttr), fmt.Sprint(v), "1")
	return e
}

type ModeType string

const (
	MajorMode  ModeType = "major"
	KKMode     ModeType = "KK"
	HierMode   ModeType = "hier"
	IpsepMode  ModeType = "ipsep"
	SpringMode ModeType = "spring"
	MaxentMode ModeType = "maxent"
)

// SetMode
// Technique for optimizing the layout.
// For neato, if mode is "major", neato uses stress majorization.
// If mode is "KK", neato uses a version of the gradient descent method.
// The only advantage to the latter technique is that it is sometimes appreciably faster for small (number of nodes < 100) graphs.
// A significant disadvantage is that it may cycle.
// There are two experimental modes in neato, "hier", which adds a top-down directionality similar to the layout used in dot,
// and "ipsep", which allows the graph to specify minimum vertical and horizontal distances between nodes. (See the sep attribute.)
//
// For sfdp, the default mode is "spring", which corresponds to using a spring-electrical model.
// Setting mode to "maxent" causes a similar model to be run but one that also takes into account edge lengths specified by the "len" attribute.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:mode
func (g *Graph) SetMode(v ModeType) *Graph {
	g.SafeSet(string(modeAttr), string(v), string(MajorMode))
	return g
}

type ModelType string

const (
	ShortPathModel ModelType = "shortpath"
	CircuitModel   ModelType = "circuit"
	SubsetModel    ModelType = "subset"
	MdsModel       ModelType = "mds"
)

// SetModel
// This value specifies how the distance matrix is computed for the input graph.
// The distance matrix specifies the ideal distance between every pair of nodes.
// neato attemps to find a layout which best achieves these distances.
// By default, it uses the length of the shortest path, where the length of each edge is given by its len attribute.
// If model is "circuit", neato uses the circuit resistance model to compute the distances.
// This tends to emphasize clusters.
// If model is "subset", neato uses the subset model.
// This sets the edge length to be the number of nodes that are neighbors of exactly one of the end points,
// and then calculates the shortest paths.
// This helps to separate nodes with high degree.
// For more control of distances, one can use model=mds.
// In this case, the len of an edge is used as the ideal distance between its vertices.
// A shortest path calculation is only used for pairs of nodes not connected by an edge.
// Thus, by supplying a complete graph, the input can specify all of the relevant distances.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:model
func (g *Graph) SetModel(v ModelType) *Graph {
	g.SafeSet(string(modelAttr), string(v), string(ShortPathModel))
	return g
}

// SetMosek
// If Graphviz is built with MOSEK defined, mode=ipsep and mosek=true, the Mosek software (www.mosek.com) is use to solve the ipsep constraints.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:mosek
func (g *Graph) SetMosek(v bool) *Graph {
	g.SafeSet(string(mosekAttr), toBoolString(v), falseStr)
	return g
}

// SetNewRank
// The original ranking algorithm in dot is recursive on clusters.
// This can produce fewer ranks and a more compact layout,
// but sometimes at the cost of a head node being place on a higher rank than the tail node.
// It also assumes that a node is not constrained in separate, incompatible subgraphs.
// For example, a node cannot be in a cluster and also be constrained by rank=same with a node not in the cluster.
// If newrank=true, the ranking algorithm does a single global ranking, ignoring clusters.
// This allows nodes to be subject to multiple constraints.
// Rank constraints will usually take precedence over edge constraints.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:newrank
func (g *Graph) SetNewRank(v bool) *Graph {
	g.SafeSet(string(newRankAttr), toBoolString(v), falseStr)
	return g
}

// SetNodeSeparator
// In dot, this specifies the minimum space between two adjacent nodes in the same rank, in inches.
// For other layouts, this affects the spacing between loops on a single node, or multiedges between a pair of nodes.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:nodesep
func (g *Graph) SetNodeSeparator(v float64) *Graph {
	g.SafeSet(string(nodeSepAttr), fmt.Sprint(v), "0.25")
	return g
}

// SetNoJustify
// By default, the justification of multi-line labels is done within the largest context that makes sense.
// Thus, in the label of a polygonal node, a left-justified line will align with the left side of the node (shifted by the prescribed margin).
// In record nodes, left-justified line will line up with the left side of the enclosing column of fields.
// If nojustify is "true", multi-line labels will be justified in the context of itself.
// For example, if the attribute is set, the first label line is long, and the second is shorter
// and left-justified, the second will align with the left-most character in the first line, regardless of how large the node might be.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:nojustify
func (g *Graph) SetNoJustify(v bool) *Graph {
	g.SafeSet(string(noJustifyAttr), toBoolString(v), falseStr)
	return g
}

// SetNoJustify
// By default, the justification of multi-line labels is done within the largest context that makes sense.
// Thus, in the label of a polygonal node, a left-justified line will align with the left side of the node (shifted by the prescribed margin).
// In record nodes, left-justified line will line up with the left side of the enclosing column of fields.
// If nojustify is "true", multi-line labels will be justified in the context of itself.
// For example, if the attribute is set, the first label line is long, and the second is shorter
// and left-justified, the second will align with the left-most character in the first line, regardless of how large the node might be.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:nojustify
func (n *Node) SetNoJustify(v bool) *Node {
	n.SafeSet(string(noJustifyAttr), toBoolString(v), falseStr)
	return n
}

// SetNoJustify
// By default, the justification of multi-line labels is done within the largest context that makes sense.
// Thus, in the label of a polygonal node, a left-justified line will align with the left side of the node (shifted by the prescribed margin).
// In record nodes, left-justified line will line up with the left side of the enclosing column of fields.
// If nojustify is "true", multi-line labels will be justified in the context of itself.
// For example, if the attribute is set, the first label line is long, and the second is shorter
// and left-justified, the second will align with the left-most character in the first line, regardless of how large the node might be.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:nojustify
func (e *Edge) SetNoJustify(v bool) *Edge {
	e.SafeSet(string(noJustifyAttr), toBoolString(v), falseStr)
	return e
}

// SetNormalize
// If set, normalize coordinates of final layout so that the first point is at the origin,
// and then rotate the layout so that the angle of the first edge is specified by the value of normalize in degrees.
// If normalize is not a number, it is evaluated as a bool, with true corresponding to 0 degrees.
// NOTE: Since the attribute is evaluated first as a number, 0 and 1 cannot be used for false and true.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:normalize
func (g *Graph) SetNormalize(v bool) *Graph {
	g.SafeSet(string(normalizeAttr), toBoolString(v), falseStr)
	return g
}

// SetNoTranslate
// By default, the final layout is translated so that the lower-left corner of the bounding box is at the origin.
// This can be annoying if some nodes are pinned or if the user runs neato -n.
// To avoid this translation, set notranslate to true.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:notranslate
func (g *Graph) SetNoTranslate(v bool) *Graph {
	g.SafeSet(string(noTranslateAttr), toBoolString(v), falseStr)
	return g
}

// SetNSLimit
// Used to set number of iterations in network simplex applications.
// nslimit is used in computing node x coordinates, nslimit1 for ranking nodes.
// If defined, # iterations = nslimit(1) * # nodes; otherwise, # iterations = MAXINT.
func (g *Graph) SetNsLimit(v float64) *Graph {
	g.SafeSet(string(nsLimitAttr), fmt.Sprint(v), "")
	return g
}

// SetNSLimit1
// Used to set number of iterations in network simplex applications.
// nslimit is used in computing node x coordinates, nslimit1 for ranking nodes.
// If defined, # iterations = nslimit(1) * # nodes; otherwise, # iterations = MAXINT.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:nslimit
func (g *Graph) SetNsLimit1(v float64) *Graph {
	g.SafeSet(string(nsLimit1Attr), fmt.Sprint(v), "")
	return g
}

type OrderingType string

const (
	OutOrdering OrderingType = "out"
	InOrdering  OrderingType = "in"
)

// SetOrdering
// If the value of the attribute is "out",
// then the outedges of a node, that is, edges with the node as its tail node,
// must appear left-to-right in the same order in which they are defined in the input.
// If the value of the attribute is "in",
// then the inedges of a node must appear left-to-right in the same order in which they are defined in the input.
// If defined as a graph or subgraph attribute, the value is applied to all nodes in the graph or subgraph.
// Note that the graph attribute takes precedence over the node attribute.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:ordering
func (g *Graph) SetOrdering(v OrderingType) *Graph {
	g.SafeSet(string(orderingAttr), string(v), "")
	return g
}

// SetOrdering
// If the value of the attribute is "out",
// then the outedges of a node, that is, edges with the node as its tail node,
// must appear left-to-right in the same order in which they are defined in the input.
// If the value of the attribute is "in",
// then the inedges of a node must appear left-to-right in the same order in which they are defined in the input.
// If defined as a graph or subgraph attribute, the value is applied to all nodes in the graph or subgraph.
// Note that the graph attribute takes precedence over the node attribute.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:ordering
func (n *Node) SetOrdering(v OrderingType) *Node {
	n.SafeSet(string(orderingAttr), string(v), "")
	return n
}

// SetOrientation
// If "[lL]*", set graph orientation to landscape Used only if rotate is not defined.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#aa:orientation
func (g *Graph) SetOrientation(v string) *Graph {
	g.SafeSet(string(orientationAttr), v, "")
	return g
}

// SetOrientation
// Angle, in degrees, used to rotate polygon node shapes.
// For any number of polygon sides, 0 degrees rotation results in a flat base.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:orientation
func (n *Node) SetOrientation(v float64) *Node {
	n.SafeSet(string(orientationAttr), fmt.Sprint(v), "0.0")
	return n
}

type OutputMode string

const (
	BreadthFirst OutputMode = "breadthfirst"
	NodesFirst   OutputMode = "nodesfirst"
	EdgesFirst   OutputMode = "edgesfirst"
)

// SetOutputOrder
// Specify order in which nodes and edges are drawn.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:outputorder
func (g *Graph) SetOutputOrder(v OutputMode) *Graph {
	g.SafeSet(string(outputOrderAttr), string(v), string(BreadthFirst))
	return g
}

// SetOverlap
// Determines if and how node overlaps should be removed.
// Nodes are first enlarged using the sep attribute.
// If "true" , overlaps are retained.
// If the value is "scale", overlaps are removed by uniformly scaling in x and y.
// If the value converts to "false", and it is available,
// Prism, a proximity graph-based algorithm, is used to remove node overlaps.
// This can also be invoked explicitly with "overlap=prism".
// This technique starts with a small scaling up, controlled by the overlap_scaling attribute,
// which can remove a significant portion of the overlap.
// The prism option also accepts an optional non-negative integer suffix.
// This can be used to control the number of attempts made at overlap removal.
// By default, overlap="prism" is equivalent to overlap="prism1000". Setting overlap="prism0" causes only the scaling phase to be run.
// If Prism is not available, or the version of Graphviz is earlier than 2.28, "overlap=false" uses a Voronoi-based technique.
// This can always be invoked explicitly with "overlap=voronoi".
//
// If the value is "scalexy", x and y are separately scaled to remove overlaps.
//
// If the value is "compress", the layout will be scaled down as much as possible without introducing any overlaps,
// obviously assuming there are none to begin with.
//
// N.B.The remaining allowed values of overlap correspond to algorithms which, at present,
// can produce bad aspect ratios. In addition, we deprecate the use of the "ortho*" and "portho*".
//
// If the value is "vpsc", overlap removal is done as a quadratic optimization to minimize node displacement while removing node overlaps.
//
// If the value is "orthoxy" or "orthoyx", overlaps are moved by optimizing two constraint problems, one for the x axis and one for the y.
// The suffix indicates which axis is processed first.
// If the value is "ortho", the technique is similar to "orthoxy" except a heuristic is used to reduce the bias between the two passes.
// If the value is "ortho_yx", the technique is the same as "ortho", except the roles of x and y are reversed.
// The values "portho", "porthoxy", "porthoxy", and "portho_yx" are similar to the previous four,
// except only pseudo-orthogonal ordering is enforced.
//
// If the layout is done by neato with mode="ipsep", then one can use overlap=ipsep.
// In this case, the overlap removal constraints are incorporated into the layout algorithm itself.
// N.B. At present, this only supports one level of clustering.
//
// Except for fdp and sfdp, the layouts assume overlap="true" as the default.
// Fdp first uses a number of passes using a built-in, force-directed technique to try to remove overlaps.
// Thus, fdp accepts overlap with an integer prefix followed by a colon, specifying the number of tries.
// If there is no prefix, no initial tries will be performed.
// If there is nothing following a colon, none of the above methods will be attempted.
// By default, fdp uses overlap="9:prism". Note that overlap="true", overlap="0:true" and overlap="0:" all turn off all overlap removal.
//
// By default, sfdp uses overlap="prism0".
//
// Except for the Voronoi and prism methods, all of these transforms preserve the orthogonal ordering of the original layout.
// That is, if the x coordinates of two nodes are originally the same, they will remain the same,
// and if the x coordinate of one node is originally less than the x coordinate of another,
// this relation will still hold in the transformed layout.
// The similar properties hold for the y coordinates.
// This is not quite true for the "porth*" cases.
// For these, orthogonal ordering is only preserved among nodes related by an edge.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:overlap
func (g *Graph) SetOverlap(v bool) *Graph {
	g.SafeSet(string(overlapAttr), toBoolString(v), trueStr)
	return g
}

// SetOverlapScaling
// When overlap=prism, the layout is scaled by this factor,
// thereby removing a fair amount of node overlap,
// and making node overlap removal faster and better able to retain the graph's shape.
// If overlap_scaling is negative,
// the layout is scaled by -1*overlap_scaling times the average label size.
// If overlap_scaling is positive, the layout is scaled by overlap_scaling.
// If overlap_scaling is zero, no scaling is done.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:overlap_scaling
func (g *Graph) SetOverlapScaling(v float64) *Graph {
	g.SafeSet(string(overlapScalingAttr), fmt.Sprint(v), "-4")
	return g
}

// SetOverlapShrink
// If true, the overlap removal algorithm will perform a compression pass to reduce the size of the layout.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:overlap_shrink
func (g *Graph) SetOverlapShrink(v bool) *Graph {
	g.SafeSet(string(overlapShrinkAttr), toBoolString(v), trueStr)
	return g
}

// SetPack
// This is true if the value of pack is "true" (case-insensitive) or a non-negative integer.
// If true, each connected component of the graph is laid out separately,
// and then the graphs are packed together.
// If pack has an integral value, this is used as the size, in points, of a margin around each part;
// otherwise, a default margin of 8 is used.
// If pack is interpreted as false, the entire graph is laid out together.
// The granularity and method of packing is influenced by the packmode attribute.
//
// For layouts which always do packing, such a twopi, the pack attribute is just used to set the margin.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:pack
func (g *Graph) SetPack(v bool) *Graph {
	g.SafeSet(string(packAttr), toBoolString(v), falseStr)
	return g
}

type PackMode string

const (
	NodePack    PackMode = "node"
	ClusterPack PackMode = "clust"
	GraphPack   PackMode = "graph"
)

// SetPackMode
// This indicates how connected components should be packed (cf. packMode).
// Note that defining packmode will automatically turn on packing as though one had set pack=true.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:packmode
func (g *Graph) SetPackMode(v PackMode) *Graph {
	g.SafeSet(string(packModeAttr), string(v), string(NodePack))
	return g
}

// SetPad
// The pad attribute specifies how much, in inches,
// to extend the drawing area around the minimal area needed to draw the graph.
// If the pad is a single double, both the x and y pad values are set equal to the given value.
// This area is part of the drawing and will be filled with the background color, if appropriate.
//
// Normally, a small pad is used for aesthetic reasons,
// especially when a background color is used,
// to avoid having nodes and edges abutting the boundary of the drawn region.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:pad
func (g *Graph) SetPad(v float64) *Graph {
	g.SafeSet(string(padAttr), fmt.Sprint(v), "0.0555")
	return g
}

// SetPage
// Width and height of output pages, in inches.
// If only a single value is given, this is used for both the width and height.
// If this is set and is smaller than the size of the layout,
// a rectangular array of pages of the specified page size is overlaid on the layout,
// with origins aligned in the lower-left corner, thereby partitioning the layout into pages.
// The pages are then produced one at a time, in pagedir order.
//
// At present, this only works for PostScript output.
// For other types of output, one should use another tool to split the output into multiple output files.
// Or use the viewport to generate multiple files.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:page
func (g *Graph) SetPage(v float64) *Graph {
	g.SafeSet(string(pageAttr), fmt.Sprint(v), "")
	return g
}

type PageDir string

const (
	BLDir PageDir = "BL"
	BRDir PageDir = "BR"
	TLDir PageDir = "TL"
	TRDir PageDir = "TR"
	RBDir PageDir = "RB"
	RTDir PageDir = "RT"
	LBDir PageDir = "LB"
	LTDir PageDir = "LT"
)

// SetPageDir
// If the page attribute is set and applicable,
// this attribute specifies the order in which the pages are emitted.
// This is limited to one of the 8 row or column major orders.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:pagedir
func (g *Graph) SetPageDir(v PageDir) *Graph {
	g.SafeSet(string(pageDirAttr), string(v), string(BLDir))
	return g
}

// SetPenWidth
// Specifies the width of the pen, in points, used to draw lines and curves,
// including the boundaries of edges and clusters.
// The value is inherited by subclusters.
// It has no effect on text.
//
// Previous to 31 January 2008,
// the effect of penwidth=W was achieved by including setlinewidth(W) as part of a style specification.
// If both are used, penwidth will be used.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:penwidth
func (n *Node) SetPenWidth(v float64) *Node {
	n.SafeSet(string(penWidthAttr), fmt.Sprint(v), "1.0")
	return n
}

// SetPenWidth
// Specifies the width of the pen, in points, used to draw lines and curves,
// including the boundaries of edges and clusters.
// The value is inherited by subclusters.
// It has no effect on text.
//
// Previous to 31 January 2008,
// the effect of penwidth=W was achieved by including setlinewidth(W) as part of a style specification.
// If both are used, penwidth will be used.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:penwidth
func (e *Edge) SetPenWidth(v float64) *Edge {
	e.SafeSet(string(penWidthAttr), fmt.Sprint(v), "1.0")
	return e
}

// SetPeripheries
// Set number of peripheries used in polygonal shapes and cluster boundaries.
// Note that user-defined shapes are treated as a form of box shape,
// so the default peripheries value is 1 and the user-defined shape will be drawn in a bounding rectangle.
// Setting peripheries=0 will turn this off.
// Also, 1 is the maximum peripheries value for clusters.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:peripheries
func (n *Node) SetPeripheries(v int) *Node {
	n.SafeSet(string(peripheriesAttr), fmt.Sprint(v), "1")
	return n
}

// SetPin
// If true and the node has a pos attribute on input,
// neato or fdp prevents the node from moving from the input position.
// This property can also be specified in the pos attribute itself (cf. the point type).
//
// Note: Due to an artifact of the implementation, previous to 27 Feb 2014, final coordinates are translated to the origin.
// Thus, if you look at the output coordinates given in the (x)dot or plain format,
// pinned nodes will not have the same output coordinates as were given on input.
// If this is important, a simple workaround is to maintain the coordinates of a pinned node.
// The vector difference between the old and new coordinates will give the translation,
// which can then be subtracted from all of the appropriate coordinates.
//
// After 27 Feb 2014, this translation can be avoided in neato by setting the notranslate to TRUE.
// However, if the graph specifies node overlap removal or a change in aspect ratio, node coordinates may still change.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:pin
func (n *Node) SetPin(v bool) *Node {
	n.SafeSet(string(pinAttr), toBoolString(v), falseStr)
	return n
}

// SetPos
// Position of node, or spline control points.
// For nodes, the position indicates the center of the node.
// On output, the coordinates are in points.
// In neato and fdp, pos can be used to set the initial position of a node.
// By default, the coordinates are assumed to be in inches.
// However, the -s command line flag can be used to specify different units.
// As the output coordinates are in points,
// feeding the output of a graph laid out by a Graphviz program into neato or fdp will almost always require the -s flag.
//
// When the -n command line flag is used with neato,
// it is assumed the positions have been set by one of the layout programs, and are therefore in points.
// Thus, neato -n can accept input correctly without requiring a -s flag and, in fact, ignores any such flag.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:pos
func (n *Node) SetPos(x, y float64) *Node {
	n.SafeSet(string(posAttr), fmt.Sprintf("%f,%f", x, y), "")
	return n
}

// SetPos
// Position of node, or spline control points.
// For nodes, the position indicates the center of the node.
// On output, the coordinates are in points.
// In neato and fdp, pos can be used to set the initial position of a node.
// By default, the coordinates are assumed to be in inches.
// However, the -s command line flag can be used to specify different units.
// As the output coordinates are in points,
// feeding the output of a graph laid out by a Graphviz program into neato or fdp will almost always require the -s flag.
//
// When the -n command line flag is used with neato,
// it is assumed the positions have been set by one of the layout programs, and are therefore in points.
// Thus, neato -n can accept input correctly without requiring a -s flag and, in fact, ignores any such flag.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:pos
func (e *Edge) SetPos(x, y float64) *Edge {
	e.SafeSet(string(posAttr), fmt.Sprintf("%f,%f", x, y), "")
	return e
}

type QuadType string

const (
	NormalQuad QuadType = "normal"
	FastQuad   QuadType = "fast"
	NoneQuad   QuadType = "none"
)

// SetQuadType
// Quadtree scheme to use.
//
// A TRUE bool value corresponds to "normal";
// a FALSE bool value corresponds to "none".
// As a slight exception to the normal interpretation of bool, a value of "2" corresponds to "fast".
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:quadtree
func (g *Graph) SetQuadTree(v QuadType) *Graph {
	g.SafeSet(string(quadTreeAttr), string(v), string(NormalQuad))
	return g
}

// SetQuantum
// If quantum > 0.0, node label dimensions will be rounded to integral multiples of the quantum.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#d:quantum
func (g *Graph) SetQuantum(v float64) *Graph {
	g.SafeSet(string(quantumAttr), fmt.Sprint(v), "0.0")
	return g
}

type RankDir string

const (
	TBRank RankDir = "TB"
	LRRank RankDir = "LR"
	BTRank RankDir = "BT"
	RLRank RankDir = "RL"
)

// SetRankDir
// Sets direction of graph layout.
// For example, if rankdir="LR", and barring cycles, an edge T -> H;
// will go from left to right. By default, graphs are laid out from top to bottom.
//
// This attribute also has a side-effect in determining how record nodes are interpreted.
// See record shapes.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:rankdir
func (g *Graph) SetRankDir(v RankDir) *Graph {
	g.SafeSet(string(rankDirAttr), string(v), string(TBRank))
	return g
}

// SetRankSeparator
// In dot, this gives the desired rank separation, in inches.
// This is the minimum vertical distance between the bottom of the nodes in one rank and the tops of nodes in the next.
// If the value contains "equally", the centers of all ranks are spaced equally apart.
// Note that both settings are possible, e.g., ranksep = "1.2 equally".
//
// In twopi, this attribute specifies the radial separation of concentric circles.
// For twopi, ranksep can also be a list of doubles.
// The first double specifies the radius of the inner circle;
// the second double specifies the increase in radius from the first circle to the second;
// etc. If there are more circles than numbers, the last number is used as the increment for the remainder.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:ranksep
func (g *Graph) SetRankSeparator(v float64) *Graph {
	g.SafeSet(string(rankSepAttr), fmt.Sprint(v), "0.5")
	return g
}

type RatioType string

const (
	FillRatio     RatioType = "fill"
	CompressRatio RatioType = "compress"
	ExpandRatio   RatioType = "expand"
	AutoRatio     RatioType = "auto"
)

// SetRatio
// Sets the aspect ratio (drawing height/drawing width) for the drawing.
// Note that this is adjusted before the size attribute constraints are enforced.
// In addition, the calculations usually ignore the node sizes,
// so the final drawing size may only approximate what is desired.
//
// If ratio is numeric, it is taken as the desired aspect ratio.
// Then, if the actual aspect ratio is less than the desired ratio,
// the drawing height is scaled up to achieve the desired ratio;
// if the actual ratio is greater than that desired ratio, the drawing width is scaled up.
//
// If ratio = "fill" and the size attribute is set,
// node positions are scaled, separately in both x and y,
// so that the final drawing exactly fills the specified size.
// If both size values exceed the width and height of the drawing,
// then both coordinate values of each node are scaled up accordingly.
// However, if either size dimension is smaller than the corresponding dimension in the drawing,
// one dimension is scaled up so that the final drawing has the same aspect ratio as specified by size.
// Then, when rendered, the layout will be scaled down uniformly in both dimensions to fit the given size,
// which may cause nodes and text to shrink as well.
// This may not be what the user wants,
// but it avoids the hard problem of how to reposition the nodes in an acceptable fashion to reduce the drawing size.
//
// If ratio = "compress" and the size attribute is set,
// dot attempts to compress the initial layout to fit in the given size.
// This achieves a tighter packing of nodes but reduces the balance and symmetry.
// This feature only works in dot.
//
// If ratio = "expand", the size attribute is set,
// and both the width and the height of the graph are less than the value in size,
// node positions are scaled uniformly until at least one dimension fits size exactly.
// Note that this is distinct from using size as the desired size,
// as here the drawing is expanded before edges are generated and all node and text sizes remain unchanged.
//
// If ratio = "auto", the page attribute is set and the graph cannot be drawn on a single page,
// then size is set to an ``ideal'' value.
// In particular, the size in a given dimension will be the smallest integral multiple of the page size in
// that dimension which is at least half the current size.
// The two dimensions are then scaled independently to the new size.
// This feature only works in dot.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:ratio
func (g *Graph) SetRatio(v RatioType) *Graph {
	g.SafeSet(string(ratioAttr), string(v), "")
	return g
}

// SetRects
// Rectangles for fields of records, in points.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:rects
func (n *Node) SetRects(llx, lly, urx, ury float64) *Node {
	n.SafeSet(string(rectsAttr), fmt.Sprintf("%f,%f,%f,%f", llx, lly, urx, ury), "")
	return n
}

// SetRegular
// If true, force polygon to be regular,
// i.e., the vertices of the polygon will lie on a circle whose center is the center of the node.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:regular
func (n *Node) SetRegular(v bool) *Node {
	n.SafeSet(string(regularAttr), toBoolString(v), falseStr)
	return n
}

// SetReminCross
// If true and there are multiple clusters, run crossing minimization a second time.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:remincross
func (g *Graph) SetReminCross(v bool) *Graph {
	g.SafeSet(string(remincrossAttr), toBoolString(v), trueStr)
	return g
}

// SetRepulsiveForce
// The power of the repulsive force used in an extended Fruchterman-Reingold force directed model.
// Values larger than 1 tend to reduce the warping effect at the expense of less clustering.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:repulsiveforce
func (g *Graph) SetRepulsiveForce(v float64) *Graph {
	g.SafeSet(string(repulsiveforceAttr), fmt.Sprint(v), "1.0")
	return g
}

// SetResolution
// This is a synonym for the dpi attribute.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:resolution
func (g *Graph) SetResolution(v float64) *Graph {
	g.SafeSet(string(resolutionAttr), fmt.Sprint(v), "96.0")
	return g
}

// SetRoot
// This specifies nodes to be used as the center of the layout and the root of the generated spanning tree.
// As a graph attribute, this gives the name of the node.
// As a node attribute, it specifies that the node should be used as a central node.
// In twopi, this will actually be the central node.
// In circo, the block containing the node will be central in the drawing of its connected component.
// If not defined, twopi will pick a most central node, and circo will pick a random node.
//
// If the root attribute is defined as the empty string, twopi will reset it to name of the node picked as the root node.
//
// For twopi, it is possible to have multiple roots, presumably one for each component.
// If more than one node in a component is marked as the root, twopi will pick one.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:root
func (g *Graph) SetRoot(v bool) *Graph {
	g.SafeSet(string(rootAttr), toBoolString(v), falseStr)
	return g
}

// SetRoot
// This specifies nodes to be used as the center of the layout and the root of the generated spanning tree.
// As a graph attribute, this gives the name of the node.
// As a node attribute, it specifies that the node should be used as a central node.
// In twopi, this will actually be the central node.
// In circo, the block containing the node will be central in the drawing of its connected component.
// If not defined, twopi will pick a most central node, and circo will pick a random node.
//
// If the root attribute is defined as the empty string, twopi will reset it to name of the node picked as the root node.
//
// For twopi, it is possible to have multiple roots, presumably one for each component.
// If more than one node in a component is marked as the root, twopi will pick one.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:root
func (n *Node) SetRoot(v bool) *Node {
	n.SafeSet(string(rootAttr), toBoolString(v), falseStr)
	return n
}

// SetRotate
// If 90, set drawing orientation to landscape.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:rotate
func (g *Graph) SetRotate(v int) *Graph {
	g.SafeSet(string(rotateAttr), fmt.Sprint(v), "0")
	return g
}

// SetRotation
// Causes the final layout to be rotated counter-clockwise by the specified number of degrees.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:rotation
func (g *Graph) SetRotation(v float64) *Graph {
	g.SafeSet(string(rotationAttr), fmt.Sprint(v), "0")
	return g
}

// SetSameHead
// Edges with the same head and the same samehead value are aimed at the same point on the head.
// This has no effect on loops.
// Each node can have at most 5 unique samehead values.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:samehead
func (e *Edge) SetSameHead(v string) *Edge {
	e.SafeSet(string(sameHeadAttr), v, "")
	return e
}

// SetSameTail
// Edges with the same tail and the same sametail value are aimed at the same point on the tail.
// This has no effect on loops.
// Each node can have at most 5 unique sametail values
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:sametail
func (e *Edge) SetSameTail(v string) *Edge {
	e.SafeSet(string(sameTailAttr), v, "")
	return e
}

// SetSamplePoints
// If the input graph defines the vertices attribute,
// and output is dot or xdot,
// this gives the number of points used for a node whose shape is a circle or ellipse.
// It plays the same role in neato,
// when adjusting the layout to avoid overlapping nodes, and in image maps.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:samplepoints
func (n *Node) SetSamplePoints(v int) *Node {
	n.SafeSet(string(samplePointsAttr), fmt.Sprint(v), "8")
	return n
}

// SetScale
// If set, after the initial layout, the layout is scaled by the given factors.
// If only a single number is given, this is used for both factors.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:scale
func (g *Graph) SetScale(x, y float64) *Graph {
	g.SafeSet(string(scaleAttr), fmt.Sprintf("%f,%f", x, y), "")
	return g
}

// SetSearchSize
// During network simplex,
// maximum number of edges with negative cut values to search when looking for one with minimum cut value.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:searchsize
func (g *Graph) SetSearchSize(v int) *Graph {
	g.SafeSet(string(searchSizeAttr), fmt.Sprint(v), "30")
	return g
}

// SetSeparator
// Specifies margin to leave around nodes when removing node overlap.
// This guarantees a minimal non-zero distance between nodes.
//
// If the attribute begins with a plus sign '+', an additive margin is specified.
// That is, "+w,h" causes the node's bounding box to be increased by w points on the left and right sides,
// and by h points on the top and bottom.
// Without a plus sign, the node is scaled by 1 + w in the x coordinate and 1 + h in the y coordinate.
//
// If only a single number is given, this is used for both dimensions.
//
// If unset but esep is defined, the sep values will be set to the esep values divided by 0.8.
// If esep is unset, the default value is used.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:sep
func (g *Graph) SetSeparator(v string) *Graph {
	g.SafeSet(string(sepAttr), v, "+4")
	return g
}

// Shape https://graphviz.gitlab.io/_pages/doc/info/shapes.html
type Shape string

const (
	BoxShape             Shape = "box"
	PolygonShape         Shape = "polygon"
	EllipseShape         Shape = "ellipse"
	OvalShape            Shape = "oval"
	CircleShape          Shape = "circle"
	PointShape           Shape = "point"
	EggShape             Shape = "egg"
	TriangleShape        Shape = "triangle"
	PlainTextShape       Shape = "plaintext"
	PlainShape           Shape = "plain"
	DiamondShape         Shape = "diamond"
	TrapeziumShape       Shape = "trapezium"
	ParallelogramShape   Shape = "parallelogram"
	HouseShape           Shape = "house"
	PentagonShape        Shape = "pentagon"
	HexagonShape         Shape = "hexagon"
	SeptagonShape        Shape = "septagon"
	OctagonShape         Shape = "octagon"
	DoubleCircleShape    Shape = "doublecircle"
	DoubleOctagonShape   Shape = "doubleoctagon"
	TripleOctagonShape   Shape = "tripleoctagon"
	InvTriangleShape     Shape = "invtriangle"
	InvTrapeziumShape    Shape = "invtrapezium"
	InvHouseShape        Shape = "invhouse"
	MdiamondShape        Shape = "Mdiamond"
	MsquareShape         Shape = "Msquare"
	McircleShape         Shape = "Mcircle"
	RectShape            Shape = "rect"
	RectangleShape       Shape = "rectangle"
	SquareShape          Shape = "square"
	StarShape            Shape = "star"
	NoneShape            Shape = "none"
	UnderlineShape       Shape = "underline"
	CylinderShape        Shape = "cylinder"
	NoteShape            Shape = "note"
	TabShape             Shape = "tab"
	FolderShape          Shape = "folder"
	Box3DShape           Shape = "box3d"
	ComponentShape       Shape = "component"
	PromoterShape        Shape = "promoter"
	CdsShape             Shape = "cds"
	TerminatorShape      Shape = "terminator"
	UtrShape             Shape = "utr"
	PrimersiteShape      Shape = "primersite"
	RestrictionSiteShape Shape = "restrictionsite"
	FivePoverHangShape   Shape = "fivepoverhang"
	ThreePoverHangShape  Shape = "threepoverhang"
	NoverHangShape       Shape = "noverhang"
	AssemblyShape        Shape = "assembly"
	SignatureShape       Shape = "signature"
	InsulatorShape       Shape = "insulator"
	RibositeShape        Shape = "ribosite"
	RnastabShape         Shape = "rnastab"
	ProteasesiteShape    Shape = "proteasesite"
	ProteinstabShape     Shape = "proteinstab"
	RPromoterShape       Shape = "rpromoter"
	RArrowShape          Shape = "rarrow"
	LArrowShape          Shape = "larrow"
	LPromoterShape       Shape = "lpromoter"
)

// SetShape
// Set the shape of a node.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:shape
func (n *Node) SetShape(v Shape) *Node {
	n.SafeSet(string(shapeAttr), string(v), string(EllipseShape))
	return n
}

// SetShapeFile
// (Deprecated) If defined, shapefile specifies a file containing user-supplied node content.
// The shape of the node is set to box.
// The image in the shapefile must be rectangular.
// The image formats supported as well as the precise semantics of how the file is used depends on the output format.
// For further details, see Image Formats and External PostScript files.
//
// There is one exception to this usage.
// If shape is set to "epsf", shapefile gives a filename containing a definition of the node in PostScript.
// The graphics defined must be contain all of the node content, including any desired boundaries.
// For further details, see External PostScript files.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:shapefile
func (n *Node) SetShapeFile(v string) *Node {
	n.SafeSet(string(shapeFileAttr), v, "")
	return n
}

// SetShowBoxes
// Print guide boxes in PostScript at the beginning of routesplines if 1, or at the end if 2. (Debugging)
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:showboxes
func (g *Graph) SetShowBoxes(v int) *Graph {
	g.SafeSet(string(showBoxesAttr), fmt.Sprint(v), "0")
	return g
}

// SetShowBoxes
// Print guide boxes in PostScript at the beginning of routesplines if 1, or at the end if 2. (Debugging)
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:showboxes
func (n *Node) SetShowBoxes(v int) *Node {
	n.SafeSet(string(showBoxesAttr), fmt.Sprint(v), "0")
	return n
}

// SetShowBoxes
// Print guide boxes in PostScript at the beginning of routesplines if 1, or at the end if 2. (Debugging)
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:showboxes
func (e *Edge) SetShowBoxes(v int) *Edge {
	e.SafeSet(string(showBoxesAttr), fmt.Sprint(v), "0")
	return e
}

// SetSides
// Number of sides if shape=polygon.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:sides
func (n *Node) SetSides(v int) *Node {
	n.SafeSet(string(sidesAttr), fmt.Sprint(v), "4")
	return n
}

// SetSize
// Maximum width and height of drawing, in inches.
// If only a single number is given, this is used for both the width and the height.
//
// If defined and the drawing is larger than the given size,
// the drawing is uniformly scaled down so that it fits within the given size.
//
// If size ends in an exclamation point (!), then it is taken to be the desired size.
// In this case, if both dimensions of the drawing are less than size,
// the drawing is scaled up uniformly until at least one dimension equals its dimension in size.
//
// Note that there is some interaction between the size and ratio attributes.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:size
func (g *Graph) SetSize(x, y float64) *Graph {
	g.SafeSet(string(sizeAttr), fmt.Sprintf("%f,%f", x, y), "")
	return g
}

// SetSkew
// Skew factor for shape=polygon.
// Positive values skew top of polygon to right; negative to left.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:skew
func (n *Node) SetSkew(v float64) *Node {
	n.SafeSet(string(skewAttr), fmt.Sprint(v), "0.0")
	return n
}

type SmoothType string

const (
	NoneSmooth      SmoothType = "none"
	AvgDistSmooth   SmoothType = "avg_dist"
	GraphDistSmooth SmoothType = "graph_dist"
	PowerDistSmooth SmoothType = "power_dist"
	RngSmooth       SmoothType = "rng"
	SprintSmooth    SmoothType = "spring"
	TriangleSmooth  SmoothType = "triangle"
)

// SetSmoothing
// Specifies a post-processing step used to smooth out an uneven distribution of nodes.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:smoothing
func (g *Graph) SetSmoothing(v SmoothType) *Graph {
	g.SafeSet(string(smoothingAttr), string(v), string(NoneSmooth))
	return g
}

// SetSortv
// If packmode indicates an array packing,
// this attribute specifies an insertion order among the components,
// with smaller values inserted first.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:sortv
func (g *Graph) SetSortv(v int) *Graph {
	g.SafeSet(string(sortvAttr), fmt.Sprint(v), "0")
	return g
}

// SetSortv
// If packmode indicates an array packing,
// this attribute specifies an insertion order among the components,
// with smaller values inserted first.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:sortv
func (n *Node) SetSortv(v int) *Node {
	n.SafeSet(string(sortvAttr), fmt.Sprint(v), "0")
	return n
}

// SetSplines
// Controls how, and if, edges are represented.
// If true, edges are drawn as splines routed around nodes;
// if false, edges are drawn as line segments.
// If set to none or "", no edges are drawn at all.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:splines
func (g *Graph) SetSplines(v string) *Graph {
	g.SafeSet(string(splinesAttr), v, "")
	return g
}

type StartType string

const (
	RegularStart StartType = "regular"
	SelfStart    StartType = "self"
	RandomStart  StartType = "random"
)

// SetStart
// Parameter used to determine the initial layout of nodes.
// If unset, the nodes are randomly placed in a unit square with the same seed is always used for the random number generator,
// so the initial placement is repeatable.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:start
func (g *Graph) SetStart(v StartType) *Graph {
	g.SafeSet(string(startAttr), string(v), "")
	return g
}

type GraphStyle string

const (
	SolidGraphStyle   GraphStyle = "solid"
	DashedGraphStyle  GraphStyle = "dashed"
	DottedGraphStyle  GraphStyle = "dotted"
	BoldGraphStyle    GraphStyle = "bold"
	RoundedGraphStyle GraphStyle = "rounded"
	FilledGraphStyle  GraphStyle = "filled"
	StripedGraphStyle GraphStyle = "striped"
)

type NodeStyle string

const (
	SolidNodeStyle     NodeStyle = "solid"
	DashedNodeStyle    NodeStyle = "dashed"
	DottedNodeStyle    NodeStyle = "dotted"
	BoldNodeStyle      NodeStyle = "bold"
	RoundedNodeStyle   NodeStyle = "rounded"
	DiagonalsNodeStyle NodeStyle = "diagonals"
	FilledNodeStyle    NodeStyle = "filled"
	StripedNodeStyle   NodeStyle = "striped"
	WedgesNodeStyle    NodeStyle = "wedged"
)

type EdgeStyle string

const (
	SolidEdgeStyle  EdgeStyle = "solid"
	DashedEdgeStyle EdgeStyle = "dashed"
	DottedEdgeStyle EdgeStyle = "dotted"
	BoldEdgeStyle   EdgeStyle = "bold"
)

// SetStyle
// Set style information for components of the graph.
// For cluster subgraphs, if style="filled", the cluster box's background is filled.
//
// If the default style attribute has been set for a component,
// an individual component can use style="" to revert to the normal default.
// For example, if the graph has
//
// edge [style="invis"]
//
// making all edges invisible, a specific edge can overrride this via:
//
// a -> b [style=""]
//
// Of course, the component can also explicitly set its style attribute to the desired value.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:style
func (g *Graph) SetStyle(v GraphStyle) *Graph {
	g.SafeSet(string(styleAttr), string(v), "")
	return g
}

// SetStyle
// Set style information for components of the graph.
// For cluster subgraphs, if style="filled", the cluster box's background is filled.
//
// If the default style attribute has been set for a component,
// an individual component can use style="" to revert to the normal default.
// For example, if the graph has
//
// edge [style="invis"]
//
// making all edges invisible, a specific edge can overrride this via:
//
// a -> b [style=""]
//
// Of course, the component can also explicitly set its style attribute to the desired value.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:style
func (n *Node) SetStyle(v NodeStyle) *Node {
	n.SafeSet(string(styleAttr), string(v), "")
	return n
}

// SetStyle
// Set style information for components of the graph.
// For cluster subgraphs, if style="filled", the cluster box's background is filled.
//
// If the default style attribute has been set for a component,
// an individual component can use style="" to revert to the normal default.
// For example, if the graph has
//
// edge [style="invis"]
//
// making all edges invisible, a specific edge can overrride this via:
//
// a -> b [style=""]
//
// Of course, the component can also explicitly set its style attribute to the desired value.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:style
func (e *Edge) SetStyle(v EdgeStyle) *Edge {
	e.SafeSet(string(styleAttr), string(v), "")
	return e
}

// SetStyleSheet
// A URL or pathname specifying an XML style sheet, used in SVG output.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:stylesheet
func (g *Graph) SetStyleSheet(v string) *Graph {
	g.SafeSet(string(stylesheetAttr), v, "")
	return g
}

// SetTailURL
// If tailURL is defined, it is output as part of the tail label of the edge.
// Also, this value is used near the tail node, overriding any URL value.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:tailURL
func (e *Edge) SetTailURL(v string) *Edge {
	e.SafeSet(string(tailURLAttr), v, "")
	return e
}

// SetTailLabelPoint
// Position of an edge's tail label, in points.
// The position indicates the center of the label.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:tail_lp
func (e *Edge) SetTailLabelPoint(x, y float64) *Edge {
	e.SafeSet(string(tailLpAttr), fmt.Sprintf("%f,%f", x, y), "")
	return e
}

// SetTailClip
// If true, the tail of an edge is clipped to the boundary of the tail node;
// otherwise, the end of the edge goes to the center of the node, or the center of a port, if applicable.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:tailclip
func (e *Edge) SetTailClip(v bool) *Edge {
	e.SafeSet(string(tailClipAttr), toBoolString(v), trueStr)
	return e
}

// SetTailHref
// Synonym for tailURL.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:tailhref
func (e *Edge) SetTailHref(v string) *Edge {
	e.SafeSet(string(tailHrefAttr), v, "")
	return e
}

// SetTailLabel
// Text label to be placed near tail of edge.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:taillabel
func (e *Edge) SetTailLabel(v string) *Edge {
	e.SafeSet(string(tailLabelAttr), v, "")
	return e
}

// SetTailPort
// Indicates where on the tail node to attach the tail of the edge.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:tailport
func (e *Edge) SetTailPort(v string) *Edge {
	e.SafeSet(string(tailPortAttr), v, "center")
	return e
}

// SetTailTarget
// If the edge has a tailURL, this attribute determines which window of the browser is used for the URL.
// Setting it to "_graphviz" will open a new window if it doesn't already exist, or reuse it if it does.
// If undefined, the value of the target is used.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:tailtarget
func (e *Edge) SetTailTarget(v string) *Edge {
	e.SafeSet(string(tailTargetAttr), v, "")
	return e
}

// SetTailTooltip
// Tooltip annotation attached to the tail of an edge.
// This is used only if the edge has a tailURL attribute.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:tailtooltip
func (e *Edge) SetTailTooltip(v string) *Edge {
	e.SafeSet(string(tailTooltipAttr), v, "")
	return e
}

// SetTarget
// If the object has a URL, this attribute determines which window of the browser is used for the URL.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:target
func (g *Graph) SetTarget(v string) *Graph {
	g.SafeSet(string(targetAttr), v, "")
	return g
}

// SetTarget
// If the object has a URL, this attribute determines which window of the browser is used for the URL.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:target
func (n *Node) SetTarget(v string) *Node {
	n.SafeSet(string(targetAttr), v, "")
	return n
}

// SetTarget
// If the object has a URL, this attribute determines which window of the browser is used for the URL.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:target
func (e *Edge) SetTarget(v string) *Edge {
	e.SafeSet(string(targetAttr), v, "")
	return e
}

// SetTooltip
// Tooltip annotation attached to the node or edge.
// If unset, Graphviz will use the object's label if defined.
// Note that if the label is a record specification or an HTML-like label, the resulting tooltip may be unhelpful.
// In this case, if tooltips will be generated, the user should set a tooltip attribute explicitly.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:tooltip
func (n *Node) SetTooltip(v string) *Node {
	n.SafeSet(string(tooltipAttr), v, "")
	return n
}

// SetTooltip
// Tooltip annotation attached to the node or edge.
// If unset, Graphviz will use the object's label if defined.
// Note that if the label is a record specification or an HTML-like label, the resulting tooltip may be unhelpful.
// In this case, if tooltips will be generated, the user should set a tooltip attribute explicitly.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:tooltip
func (e *Edge) SetTooltip(v string) *Edge {
	e.SafeSet(string(tooltipAttr), v, "")
	return e
}

// SetTrueColor
// If set explicitly to true or false,
// the value determines whether or not internal bitmap rendering relies on a truecolor color model or uses a color palette.
// If the attribute is unset, truecolor is not used unless there is a shapefile property for some node in the graph.
// The output model will use the input model when possible.
//
// Use of color palettes results in less memory usage during creation of the bitmaps and smaller output files.
//
// Usually, the only time it is necessary to specify the truecolor model is if the graph uses more than 256 colors.
// However, if one uses bgcolor=transparent with a color palette,
// font antialiasing can show up as a fuzzy white area around characters.
// Using truecolor=true avoids this problem.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:truecolor
func (g *Graph) SetTrueColor(v bool) *Graph {
	g.SafeSet(string(trueColorAttr), toBoolString(v), "")
	return g
}

// SetVertices
// If the input graph defines this attribute, the node is polygonal,
// and output is dot or xdot, this attribute provides the coordinates of the vertices of the node's polygon, in inches.
// If the node is an ellipse or circle, the samplepoints attribute affects the output.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:vertices
func (n *Node) SetVertices(v string) *Node {
	n.SafeSet(string(verticesAttr), v, "")
	return n
}

// SetViewport
// Clipping window on final drawing.
// Note that this attribute supersedes any size attribute.
// The width and height of the viewport specify precisely the final size of the output.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:viewport
func (g *Graph) SetViewport(v string) *Graph {
	g.SafeSet(string(viewportAttr), v, "")
	return g
}

// SetVoroMargin
// Factor to scale up drawing to allow margin for expansion in Voronoi technique.
// dim' = (1+2*margin)*dim.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:voro_margin
func (g *Graph) SetVoroMargin(v float64) *Graph {
	g.SafeSet(string(voroMarginAttr), fmt.Sprint(v), "0.05")
	return g
}

// SetWeight
// Weight of edge.
// In dot, the heavier the weight, the shorter, straighter and more vertical the edge is.
// N.B. Weights in dot must be integers.
// For twopi, a weight of 0 indicates the edge should not be used in constructing a spanning tree from the root.
// For other layouts, a larger weight encourages the layout to make the edge length closer to that specified by the len attribute.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:weight
func (e *Edge) SetWeight(v float64) *Edge {
	e.SafeSet(string(weightAttr), fmt.Sprint(v), "1")
	return e
}

// SetWidth
// Width of node, in inches.
// This is taken as the initial, minimum width of the node.
// If fixedsize is true, this will be the final width of the node.
// Otherwise, if the node label requires more width to fit, the node's width will be increased to contain the label.
// Note also that, if the output format is dot, the value given to width will be the final value.
//
// If the node shape is regular, the width and height are made identical.
// In this case, if either the width or the height is set explicitly, that value is used.
// In this case, if both the width or the height are set explicitly, the maximum of the two values is used.
// If neither is set explicitly, the minimum of the two default values is used.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:width
func (n *Node) SetWidth(v float64) *Node {
	n.SafeSet(string(widthAttr), fmt.Sprint(v), "0.75")
	return n
}

// SetXDotVersion
// For xdot output, if this attribute is set, this determines the version of xdot used in output.
// If not set, the attribute will be set to the xdot version used for output.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:xdotversion
func (g *Graph) SetXDotVersion(v string) *Graph {
	g.SafeSet(string(xdotVersionAttr), v, "")
	return g
}

// SetXLabel
// External label for a node or edge.
// For nodes, the label will be placed outside of the node but near it.
// For edges, the label will be placed near the center of the edge.
// This can be useful in dot to avoid the occasional problem when the use of edge labels distorts the layout.
// For other layouts, the xlabel attribute can be viewed as a synonym for the label attribute.
//
// These labels are added after all nodes and edges have been placed.
// The labels will be placed so that they do not overlap any node or label.
// This means it may not be possible to place all of them.
// To force placing all of them, use the forcelabels attribute.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:xlabel
func (n *Node) SetXLabel(v string) *Node {
	n.SafeSet(string(xlabelAttr), v, "")
	return n
}

// SetXLabel
// External label for a node or edge.
// For nodes, the label will be placed outside of the node but near it.
// For edges, the label will be placed near the center of the edge.
// This can be useful in dot to avoid the occasional problem when the use of edge labels distorts the layout.
// For other layouts, the xlabel attribute can be viewed as a synonym for the label attribute.
//
// These labels are added after all nodes and edges have been placed.
// The labels will be placed so that they do not overlap any node or label.
// This means it may not be possible to place all of them.
// To force placing all of them, use the forcelabels attribute.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:xlabel
func (e *Edge) SetXLabel(v string) *Edge {
	e.SafeSet(string(xlabelAttr), v, "")
	return e
}

// SetXLabelPosition
// Position of an exterior label, in points.
// The position indicates the center of the label.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:xlp
func (n *Node) SetXLabelPosition(x, y float64) *Node {
	n.SafeSet(string(xlpAttr), fmt.Sprintf("%f,%f", x, y), "")
	return n
}

// SetXLabelPosition
// Position of an exterior label, in points.
// The position indicates the center of the label.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:xlp
func (e *Edge) SetXLabelPosition(x, y float64) *Edge {
	e.SafeSet(string(xlpAttr), fmt.Sprintf("%f,%f", x, y), "")
	return e
}

// SetZ
// Deprecated:Use pos attribute, along with dimen and/or dim to specify dimensions.
//
// Provides z coordinate value for 3D layouts and displays.
// If the graph has dim set to 3 (or more),
// neato will use a node's z value for the z coordinate of its initial position if its pos attribute is also defined.
//
// Even if no z values are specified in the input,
// it is necessary to declare a z attribute for nodes, e.g, using node[z=""] in order to get z values on output.
// Thus, setting dim=3 but not declaring z will cause neato -Tvrml to layout the graph in 3D but project the layout onto the xy-plane for the rendering.
// If the z attribute is declared, the final rendering will be in 3D.
// https://graphviz.gitlab.io/_pages/doc/info/attrs.html#a:z
func (n *Node) SetZ(v float64) *Node {
	n.SafeSet(string(zAttr), fmt.Sprint(v), "0.0")
	return n
}
