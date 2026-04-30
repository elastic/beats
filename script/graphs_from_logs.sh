#!/usr/bin/env sh
#
# Script should be run from the directory that contains the ndjson
# logs from beats or elastic-agent diagnostics.  The script will
# generate a tsv file for each component, the tsv will contain
# metrics, see jq query below.  The script will then generate a
# gnuplot script and generate a svg graph for each metric.  If you
# want to add new metrics, add to the end of the array otherwise you
# will have to adjust the gnuplot scripts to select the correct
# columns.  The script also generates an 'index.html' which will allow
# you to view all the graphs at once.
#
# Running the script with the 'clean' argument will remove all the
# intermediate files(tsv, gp, svg and index.html)
#
set -e
REQUIRED_COMMANDS="jq gnuplot sed tr sort"
for COMMAND in $REQUIRED_COMMANDS
do
    if ! command -v $COMMAND >/dev/null 2>&1
    then
	echo "Error: $COMMAND must be installed and in PATH"
	exit 1
    fi
done

if [ "$#" -gt 1 ]
then
    echo "Error: only one command line argument is supported 'clean'"
    exit 1
fi

if [ "$#" -eq 1 ]
then
    if [ "$1" == 'clean' ]
    then
	rm -f *.tsv
	rm -f *.gp
	rm -f *.svg
	rm -f index.html
	exit
    else
	echo "Error: $1 is not a recognized command line option"
	exit 1
    fi
fi


# find components from elastic-agent diagnostics or service.name from plain beats
COMPONENTS=`jq -r 'select (.message == "Non-zero metrics in the last 30s") | .component.id // ."service.name"' *.ndjson | sort -u`

SAFE_COMPONENTS=""
for COMPONENT in $COMPONENTS
do
    # some component names have slashes which can interfere with filenames
    SAFE_COMPONENT=`echo "$COMPONENT" | tr '/' '_'`
    SAFE_COMPONENTS="${SAFE_COMPONENT} ${SAFE_COMPONENTS}"
    jq --arg my_component "$COMPONENT" -r '
    select (.message == "Non-zero metrics in the last 30s") |
    select ((.component.id == $my_component) or (."service.name" == $my_component)) |
    [."@timestamp",
    .monitoring.metrics.beat.cgroup.memory.mem.usage.bytes //0,
    .monitoring.metrics.beat.cpu.total.time.ms //0,
    .monitoring.metrics.beat.handles.open //0,
    .monitoring.metrics.beat.memstats.rss // 0,
    .monitoring.metrics.beat.runtime.goroutines // 0,
    .monitoring.metrics.filebeat.harvester.open_files //0,
    .monitoring.metrics.filebeat.harvester.running //0,
    .monitoring.metrics.libbeat.output.events.acked // 0,
    .monitoring.metrics.libbeat.output.events.batches // 0,
    .monitoring.metrics.libbeat.output.write.latency.histogram.mean // 0,
    .monitoring.metrics.libbeat.output.write.latency.histogram.stddev // 0,
    .monitoring.metrics.libbeat.output.write.latency_delta.histogram.mean // 0,
    .monitoring.metrics.libbeat.output.write.latency_delta.histogram.stddev // 0,
    .monitoring.metrics.libbeat.pipeline.queue.filled.pct // 0,
    .monitoring.metrics.system.load."1" //0
    ] |
    @tsv' *.ndjson | sort > $SAFE_COMPONENT.tsv
done

#filename;title;ylabel;columns to graph;graph type
GRAPHS='cgroup;cgroup memory;bytes;1:2;lines
cpu;cpu time;milliseconds;1:3;lines
open_handles;open file handles;count;1:4;lines
rss;rss;bytes;1:5;lines
goroutines;go routines;count;1:6;lines
harvester_open;harvester open files;count;1:7;lines
harvester_running;harvesters running;count;1:8;lines
acked;output acked events;count;1:9;lines
batches;output batches;count;1:10;lines
latency;output latency;milliseconds;1:11:12;yerrorlines
latency_delta;output delta latency;milliseconds;1:13:14;yerrorlines
queue;queue filled pct;percent;1:15;lines
load;load;load;1:16;lines'

cat <<EOF > index.html
<html>
<head>
<title>30 sec graphs</title>
</head>
<body>
EOF

echo "${GRAPHS}" | while IFS='\n' read GRAPH; do
    echo "$GRAPH" | while IFS=';' read FILENAME TITLE YLABEL COLUMNS GRAPH_TYPE; do
	PLOT="plot "
	for SAFE_COMPONENT in ${SAFE_COMPONENTS}
	do
	    PLOT="${PLOT} \"${SAFE_COMPONENT}.tsv\" using ${COLUMNS} with ${GRAPH_TYPE} title \"$SAFE_COMPONENT\", "
	done
	# remove trailing comma and space
	PLOT=`echo $PLOT | sed 's/..$//'`
	   
	cat <<EOF > "${FILENAME}".gp
reset
set terminal push
set terminal svg background "white"
set termoption font "Arial"
set output "${FILENAME}.svg"
set title "${TITLE}"
set xdata time
set timefmt "%Y-%m-%dT%H:%M:%S"
set xlabel "Date/Time"
set ylabel "${YLABEL}"
${PLOT}
set terminal pop
set output
EOF

	gnuplot "${FILENAME}".gp
	echo "<img src=\"${FILENAME}.svg\" alt=\"${FILENAME}\">" >> index.html
    done
done
cat <<EOF >> index.html
</body>
</html>
EOF
exit

