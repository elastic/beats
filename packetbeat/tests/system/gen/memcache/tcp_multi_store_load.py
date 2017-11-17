#!/usr/bin/env python

import mc


def run(mc):
    print('run')

    res = mc.set_multi({
        "k1": 100 * 'a',
        "k2": 20 * 'b',
        "k3": 10 * 'c',
    })
    print(res)
    if len(res) > 0:
        raise RuntimeError("failed to set value")

    res = mc.get_multi(["x", "k1", "k2", "k3", "y"])
    print(res)


if __name__ == '__main__':
    mc.run_tcp(run)
