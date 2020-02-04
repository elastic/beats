#include "config.h"
#include "gvplugin.h"
#include "gvplugin_render.h"
#include <stdio.h>

extern gvplugin_library_t gvplugin_dot_layout_LTX_library;
extern gvplugin_library_t gvplugin_neato_layout_LTX_library;
extern gvplugin_library_t gvplugin_core_LTX_library;
extern gvplugin_library_t gvplugin_go_library;

lt_symlist_t lt_preloaded_symbols[] = {
	{ "gvplugin_dot_layout_LTX_library", (void*)(&gvplugin_dot_layout_LTX_library) },
    { "gvplugin_neato_layout_LTX_library", (void*)(&gvplugin_neato_layout_LTX_library) },
	{ "gvplugin_core_LTX_library", (void*)(&gvplugin_core_LTX_library) },
    { "gvplugin_go_LTX_library", (void*)(&gvplugin_go_library) },
	{ 0, 0 }
};

extern gvplugin_installed_t gvdevice_go_types[];
extern gvplugin_installed_t gvrender_go_types[];

static gvplugin_api_t go_apis[] = {
    {API_device, gvdevice_go_types},
    {API_render, gvrender_go_types},
    {(api_t)0, 0},
};

gvplugin_library_t gvplugin_go_library = { "go", go_apis };

typedef enum { FORMAT_PNG, FORMAT_JPG } go_format_type;

#include "_cgo_export.h"

void *call_searchf(Dtsearch_f searchf, Dt_t *a0, void *a1, int a2) {
    return searchf(a0, a1, a2);
}

void *call_memoryf(Dtmemory_f memoryf, Dt_t *a0, void *a1, size_t a2, Dtdisc_t *a3) {
    return memoryf(a0, a1, a2, a3);
}

void *call_makef(Dtmake_f makef, Dt_t *a0, void *a1, Dtdisc_t *a2) {
    return makef(a0, a1, a2);
}

int call_comparf(Dtcompar_f comparf, Dt_t *a0, void *a1, void *a2, Dtdisc_t *a3) {
    return comparf(a0, a1, a2, a3);
}

void call_freef(Dtfree_f freef, Dt_t *a0, void *a1, Dtdisc_t *a2) {
    return freef(a0, a1, a2);
}

unsigned int call_hashf(Dthash_f hashf, Dt_t *a0, void *a1, Dtdisc_t *a2) {
    return hashf(a0, a1, a2);
}

int call_eventf(Dtevent_f eventf, Dt_t *a0, int a1, void *a2, Dtdisc_t *a3) {
    return eventf(a0, a1, a2, a3);
}

static int dtwalk_gocallback(Dt_t *a0, void *a1, void *a2) {
    return GoDtwalkCallback(a0, a1, a2);
}

int call_dtwalk(Dt_t *a0, void *a1) {
    return dtwalk(a0, dtwalk_gocallback, a1);
}

void go_begin_job(GVJ_t * job)
{
  GoBeginJob(job);
}

void go_end_job(GVJ_t * job)
{
  GoEndJob(job);
}

void go_begin_graph(GVJ_t * job)
{
  GoBeginGraph(job);
}

void go_end_graph(GVJ_t * job)
{
  GoEndGraph(job);
}

void go_begin_layer(GVJ_t * job, char *layername, int layerNum, int numLayers)
{
  GoBeginLayer(job, layername, layerNum, numLayers);
}

void go_end_layer(GVJ_t * job)
{
  GoEndLayer(job);
}

void go_begin_page(GVJ_t * job)
{
  GoBeginPage(job);
}

void go_end_page(GVJ_t * job)
{
  GoEndPage(job);
}

void go_begin_cluster(GVJ_t * job)
{
  GoBeginCluster(job);
}

void go_end_cluster(GVJ_t * job)
{
  GoEndCluster(job);
}

void go_begin_nodes(GVJ_t * job)
{
  GoBeginNodes(job);
}

void go_end_nodes(GVJ_t * job)
{
  GoEndNodes(job);
}

void go_begin_edges(GVJ_t * job)
{
  GoBeginEdges(job);
}

void go_end_edges(GVJ_t * job)
{
  GoEndEdges(job);
}

void go_begin_node(GVJ_t * job)
{
  GoBeginNode(job);
}

void go_end_node(GVJ_t * job)
{
  GoEndNode(job);
}

void go_begin_edge(GVJ_t * job)
{
  GoBeginEdge(job);
}

void go_end_edge(GVJ_t * job)
{
  GoEndEdge(job);
}

void go_begin_anchor(GVJ_t * job, char *href, char *tooltip, char *target, char *id)
{
  GoBeginAnchor(job, href, tooltip, target, id);
}

void go_end_anchor(GVJ_t * job)
{
  GoEndAnchor(job);
}

void go_begin_label(GVJ_t * job, label_type type)
{
  GoBeginLabel(job, type);
}

void go_end_label(GVJ_t * job)
{
  GoEndLabel(job);
}

void go_textspan(GVJ_t * job, pointf p, textspan_t * span)
{
  GoTextspan(job, p, span);
}

void go_resolve_color(GVJ_t * job, gvcolor_t * color)
{
  GoResolveColor(job, color->u.rgba[0], color->u.rgba[1], color->u.rgba[2], color->u.rgba[3]);
}

void go_ellipse(GVJ_t * job, pointf *A, int filled)
{
  GoEllipse(job, A[0], A[1], filled);
}

void go_polygon(GVJ_t * job, pointf * A, int n, int filled)
{
  GoPolygon(job, A, n, filled);
}

void go_beziercurve(GVJ_t *job, pointf *A, int n, int arrow_at_start, int arrow_at_end, int ext)
{
  GoBeziercurve(job, A, n, arrow_at_start, arrow_at_end, ext);
}

void go_polyline(GVJ_t *job, pointf *A, int n)
{
  GoPolyline(job, A, n);
}

void go_comment(GVJ_t * job, char *comment)
{
  GoComment(job, comment);
}

void go_library_shape(GVJ_t * job, char *name, pointf * A, int n, int filled)
{
  GoLibraryShape(job, name, A, n, filled);
}

static gvrender_engine_t go_engine = {
    go_begin_job,
    go_end_job,
    go_begin_graph,
    go_end_graph,
    go_begin_layer,
    go_end_layer,
    go_begin_page,
    go_end_page,
    go_begin_cluster,
    go_end_cluster,
    go_begin_nodes,
    go_end_nodes,
    go_begin_edges,
    go_end_edges,
    go_begin_node,
    go_end_node,
    go_begin_edge,
    go_end_edge,
    go_begin_anchor,
    go_end_anchor,
    go_begin_label,
    go_end_label,
    go_textspan,
    go_resolve_color,
    go_ellipse,
    go_polygon,
    go_beziercurve,
    go_polyline,
    go_comment,
    go_library_shape,
};

static gvrender_features_t render_features_go = {
    GVRENDER_Y_GOES_DOWN | GVRENDER_DOES_TRANSFORM, /* flags */
    4.,        /* default pad - graph units */
    0,         /* knowncolors */
    0,         /* sizeof knowncolors */
    RGBA_BYTE, /* color_type */
};

static gvdevice_features_t go_device_features = {
    GVDEVICE_BINARY_FORMAT | GVDEVICE_DOES_TRUECOLOR, /* flags */
    {0.,0.},   /* default margin - points */
    {0.,0.},   /* default page width, height - points */
    {96.,96.}, /* typical monitor dpi */
};

gvplugin_installed_t gvrender_go_types[] = {
    {FORMAT_PNG, "png", 1, &go_engine, &render_features_go},
    {FORMAT_JPG, "jpg", 1, &go_engine, &render_features_go},
    {0, NULL, 0, NULL, NULL}
};

gvplugin_installed_t gvdevice_go_types[] = {
    {FORMAT_PNG, "png:png", 1, NULL, &go_device_features},
    {FORMAT_JPG, "jpg:jpg", 1, NULL, &go_device_features},
    {0, NULL, 0, NULL, NULL}
};
