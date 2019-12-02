#!/usr/bin/env python

from __future__ import print_function
import mc


def run(mc):
    print('run')

    mc.set('cnt', 0)
    mc.incr('cnt', 2)
    mc.decr('cnt', 5)
    print(mc.get('cnt')))


if __name__ == '__main__':
    mc.run_tcp(run)
