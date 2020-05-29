#!/usr/bin/env python3

import os


if __name__ == "__main__":

    for root, dirs, files in os.walk('build'):
        if root.endswith(('system-tests')):
            print(root.replace(".{}".format(os.sep), ''))
