#!/usr/bin/env python

import mc


def run(mc):
    print('run')

    # write 1kb entry
    v = 1024 * 'a'
    if not mc.set('test_key', v):
        raise RuntimeError("failed to set value")


if __name__ == '__main__':
    mc.run_udp(run)
