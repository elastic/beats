
from contextlib import (contextmanager)
import argparse
import sys

import pylibmc


def parse_args(args=None):
    p = argparse.ArgumentParser()
    p.add_argument('--protocol', '-p', default='text',
                   help="choose protocol type. One of text or bin")
    p.add_argument('--remote', '-r', default='127.0.0.1:11211',
                   help="remote server address")
    return p.parse_args(sys.argv[1:] if args is None else args)


def connect_tcp(opts=None):
    opts = opts or parse_args()
    opts.transport = 'tcp'
    return connect(opts)


def connect_udp(opts=None):
    opts = opts or parse_args()
    opts.transport = 'udp'
    return connect(opts)


def connect(opts):
    if opts.transport == 'udp':
        addr = 'udp:' + opts.remote
    else:
        addr = opts.remote
    return pylibmc.Client([addr],
                          binary=opts.protocol == 'bin')


def make_connect_cmd(con):
    def go(opts=None):
        mc = con(opts)
        try:
            yield mc
        finally:
            mc.disconnect_all()
    return contextmanager(go)


def make_run(con):
    def go(fn, opts=None):
        with con() as mc:
            fn(mc)
    return go


connection = make_connect_cmd(connect)
tcp_connection = make_connect_cmd(connect_tcp)
udp_connection = make_connect_cmd(connect_udp)

run_tcp = make_run(tcp_connection)
run_udp = make_run(udp_connection)
