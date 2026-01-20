#!/usr/bin/env bash
MIN_BASH_MAJOR_VERSION=4
if [[ ${BASH_VERSINFO[0]} -lt $MIN_BASH_MAJOR_VERSION ]]
then
    echo "Error: bash version greater than $MIN_BASH_MAJOR_VERSION is required."
    exit 1
fi

if ! command -v jq >/dev/null 2>&1
then
    echo "Error: jq must be installed and in PATH"
    exit 1
fi

if ! command -v gnuplot >/dev/null 2>&1
then
    echo "Error: gnuplot must be installed and in PATH"
    exit 1
fi

COMPONENTS=$(jq -r 'select (.message == "Non-zero metrics in the last 30s") | .component.id // ."service.name"' *.ndjson | sort -u)
SAFE_COMPONENTS=()
for COMPONENT in $COMPONENTS
do
    SAFE_COMPONENT=$(echo "$COMPONENT" | tr '/' '_')
    SAFE_COMPONENTS+=($SAFE_COMPONENT)
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


declare -A graphs
graphs["cgroup"]="cgroup memory;bytes;1:2;lines"
graphs["cpu"]="cpu time;milliseconds;1:3;lines"
graphs["open_handles"]="open file handles;count;1:4;lines"
graphs["rss"]="rss;bytes;1:5;lines"
graphs["goroutines"]="go routines;count;1:6;lines"
graphs["harvester_open"]="harvester open files;count;1:7;lines"
graphs["harvester_running"]="harvesters running;count;1:8;lines"
graphs["acked"]="output acked events;count;1:9;lines"
graphs["batches"]="output batches;count;1:10;lines"
graphs["latency"]="output latency;milliseconds;1:11:12;yerrorlines"
graphs["latency_delta"]="output delta latency;milliseconds;1:13:14;yerrorlines"
graphs["queue"]="queue filled pct;percent;1:15;lines"
graphs["load"]="load;load;1:16;lines"

for key in "${!graphs[@]}"
do
    oIFS="$IFS"
    IFS=';'
    declare -a graph=(${graphs[$key]})
    IFS="$oIFS"
    unset oIFS
    plot="plot "
    for SAFE_COMPONENT in "${SAFE_COMPONENTS[@]}"
    do
	plot+="\"$SAFE_COMPONENT.tsv\" using ${graph[2]} with ${graph[3]} title \"$SAFE_COMPONENT\", "
    done
    plot="${plot%?}"
	   
    cat <<EOF > "$key".gp
reset
set title "${graph[0]}"
set xdata time
set timefmt "%Y-%m-%dT%H:%M:%S"
set xlabel "Date/Time"
set ylabel "${graph[1]}"
$plot
set terminal push
set terminal svg background "white"
set termoption font "Arial"
set output "$key.svg"
replot
set terminal pop
set output
EOF

    gnuplot "$key".gp
done

cat <<EOF > index.html
<html>
<head>
<title>30 sec graphs</title>
</head>
<body>
EOF

for key in "${!graphs[@]}"
do
    echo "<img src=\"$key.svg\" alt=\"$key\">" >> index.html
done

cat <<EOF >> index.html
</body>
</html>
EOF
