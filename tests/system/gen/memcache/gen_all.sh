#!/usr/bin/env bash

IFC=$1
OUTDIR=${2:-"."}
HOST=${3:-"127.0.0.1"}
PORT=${4:-"11211"}

function experiment() {
    local exp=$1
    local proto=$2

    cmd=./${exp}.py
    file=${OUTDIR}/memcache_${proto}_${exp}.pcap

    filter="host $HOST and ((ip[6:2] & 0x3fff != 0x0000) or port $PORT)"
    tcpdump -i $IFC -w $file $filter &
    sleep 2
    cap=$!
    echo "cap: $cap" 1>&2
    echo "run $cmd" 1>&2
    $cmd -p $proto -r "$HOST:$PORT"
    sleep 2
    kill $cap
}

experiment tcp_single_load_store text
experiment tcp_single_load_store bin
experiment tcp_multi_store_load text
experiment tcp_multi_store_load bin
experiment tcp_counter_ops text
experiment tcp_counter_ops bin
experiment tcp_delete text
experiment tcp_delete bin
experiment tcp_stats text
experiment tcp_stats bin

experiment udp_counter_ops text
experiment udp_counter_ops bin
experiment udp_delete text
experiment udp_delete bin
experiment udp_multi_store text
experiment udp_multi_store bin
experiment udp_single_store text
experiment udp_single_store bin
