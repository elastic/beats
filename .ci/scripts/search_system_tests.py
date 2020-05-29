#!/usr/bin/env python3

import os


if __name__ == "__main__":

    for root, dirs, files in os.walk('.'):
        if root.endswith(('system-tests')):
            print("{}".format(root.replace(".{}".format(os.sep), '')))
