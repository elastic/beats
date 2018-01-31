#!/usr/bin/env python
"""
Update packages of all libbeat's dependencies.

"""
from __future__ import print_function

import argparse
import json
import os
import subprocess
import sys


def find_packages(fp, origin, revision):
    """generate list of packages for the given origin prefix not already on revision"""
    j = json.load(fp)
    for pkg in j['package']:
        if pkg['revision'] != revision and pkg.get('origin', '').startswith(origin):
            yield pkg['path']


def main():
    default_vendor_file = os.path.join(os.path.abspath(os.path.dirname(sys.argv[0])), '..', 'vendor', 'vendor.json')

    parser = argparse.ArgumentParser()
    parser.add_argument('-f', '--vendor-file',
                        default=default_vendor_file, type=argparse.FileType(mode='r'), help='path to vendor.json')
    parser.add_argument('-n', '--dryrun', action='store_true',
                        help='do not perform updates, just list what would be updated')
    parser.add_argument('--origin', default='github.com/elastic/beats/', help='update deps of this origin prefix')
    parser.add_argument('-q', '--quiet', dest='verbose', action='store_false', help='work quietly')
    parser.add_argument('revision', help='update deps to this revision')
    args = parser.parse_args()

    packages = find_packages(args.vendor_file, args.origin, args.revision)

    if args.dryrun:
        for pkg in sorted(packages):
            print(pkg)
        return

    # much faster to call govendor once and minimize the git reset & status calls it makes
    update = ['{pkg}@{revision}'.format(pkg=pkg, revision=args.revision) for pkg in packages]
    if not update:
        print("No packages found, everything up-to-date.")
        return
    cmd = ['govendor', 'fetch'] + update
    if args.verbose:
        print(' '.join(cmd))
    subprocess.check_call(cmd)


if __name__ == '__main__':
    main()
