#!/usr/bin/env python

from __future__ import print_function
import mc


def run(mc):
    print('run')

    res = mc.set_multi({
        "k1": 100 * 'a',
        "k2": 20 * 'b',
        "k3": 10 * 'c',
    })
    print(res)


if __name__ == '__main__':
    mc.run_udp(run)
