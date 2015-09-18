#!/usr/bin/env python

import mc


def run(mc):
    print('run')

    # write 2kb entry
    v = 2046*'a'
    if not mc.set('test_key', v):
        raise RuntimeError("failed to set value")

    if v != mc.get('test_key'):
        raise RuntimeError("returned value differs")

if __name__ == '__main__':
    mc.run_tcp(run)
