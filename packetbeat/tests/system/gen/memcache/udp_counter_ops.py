#!/usr/bin/env python3

import mc


def run(mc):
    mc.set('cnt', 0)
    mc.incr('cnt', 2)
    mc.decr('cnt', 5)


if __name__ == '__main__':
    mc.run_udp(run)
