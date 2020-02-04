#include "common/args.c"
#include "common/arrows.c"
#include "common/colxlate.c"
#include "common/ellipse.c"
#include "common/emit.c"
#include "common/geom.c"
#include "common/globals.c"
#include "common/htmllex.c"
#include "common/htmltable.c"
#include "common/input.c"
#include "common/intset.c"
#include "common/labels.c"
#include "common/memory.c"
#include "common/ns.c"
#include "common/output.c"
#include "common/pointset.c"
#include "common/postproc.c"
#include "common/psusershape.c"
#include "common/routespl.c"
#include "common/shapes.c"
#include "common/splines.c"
#include "common/taper.c"
#include "common/textspan.c"
#include "common/timing.c"
#include "common/utils.c"
#include "common/htmlparse.c"

int CL_type = 0;

char **Files;	/* from command line */
const char **Lib;		/* from command line */
char *CmdName;
char *specificFlags;
char *specificItems;
char *Gvfilepath;  /* Per-process path of files allowed in image attributes (also ps libs) */
char *Gvimagepath; /* Per-graph path of files allowed in image attributes  (also ps libs) */

unsigned char Verbose;
unsigned char Reduce;
int MemTest;
char *HTTPServerEnVar;
char *Output_file_name;
int graphviz_errors;
int Nop;
double PSinputscale;
int Syntax_errors;
int Show_cnt;
char** Show_boxes;	/* emit code for correct box coordinates */
int CL_type;		/* NONE, LOCAL, GLOBAL */
unsigned char Concentrate;	/* if parallel edges should be merged */
double Epsilon;	/* defined in input_graph */
int MaxIter;
int Ndim;
int State;		/* last finished phase */
int EdgeLabelsDone;	/* true if edge labels have been positioned */
double Initial_dist;
double Damping;
int Y_invert;	/* invert y in dot & plain output */
int GvExitOnUsage;   /* gvParseArgs() should exit on usage or error */

Agsym_t
	*G_activepencolor, *G_activefillcolor,
	*G_selectedpencolor, *G_selectedfillcolor,
	*G_visitedpencolor, *G_visitedfillcolor,
	*G_deletedpencolor, *G_deletedfillcolor,
	*G_ordering, *G_peripheries, *G_penwidth,
	*G_gradientangle, *G_margin;

Agsym_t
	*N_height, *N_width, *N_shape, *N_color, *N_fillcolor,
	*N_activepencolor, *N_activefillcolor,
	*N_selectedpencolor, *N_selectedfillcolor,
	*N_visitedpencolor, *N_visitedfillcolor,
	*N_deletedpencolor, *N_deletedfillcolor,
	*N_fontsize, *N_fontname, *N_fontcolor, *N_margin,
	*N_label, *N_xlabel, *N_nojustify, *N_style, *N_showboxes,
	*N_sides, *N_peripheries, *N_ordering, *N_orientation,
	*N_skew, *N_distortion, *N_fixed, *N_imagescale, *N_layer,
	*N_group, *N_comment, *N_vertices, *N_z,
	*N_penwidth, *N_gradientangle;
Agsym_t
	*E_weight, *E_minlen, *E_color, *E_fillcolor,
	*E_activepencolor, *E_activefillcolor,
	*E_selectedpencolor, *E_selectedfillcolor,
	*E_visitedpencolor, *E_visitedfillcolor,
	*E_deletedpencolor, *E_deletedfillcolor,
	*E_fontsize, *E_fontname, *E_fontcolor,
	*E_label, *E_xlabel, *E_dir, *E_style, *E_decorate,
	*E_showboxes, *E_arrowsz, *E_constr, *E_layer,
	*E_comment, *E_label_float,
	*E_samehead, *E_sametail,
	*E_arrowhead, *E_arrowtail,
	*E_headlabel, *E_taillabel,
	*E_labelfontsize, *E_labelfontname, *E_labelfontcolor,
	*E_labeldistance, *E_labelangle,
	*E_tailclip, *E_headclip,
	*E_penwidth;

YYSTYPE htmllval;

const char *lt_dlerror (void){ return NULL; }
int lt_dlinit(void) { return 0; }

