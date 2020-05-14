#!/usr/bin/env python3

import mc


def run(mc):
    print('run')
    mc.set('key', 'abc')
    print(mc.get('key'))
    print(mc.get_stats())


if __name__ == '__main__':
    mc.run_tcp(run)
