#!/usr/bin/env python


"""
Set the origin of all libbeat's dependencies to github.com/elastic/beats/

"""
from __future__ import print_function

import argparse
import json
import os
import sys


def main():
    comment = 'File patched by {}'.format(os.path.basename(sys.argv[0]))
    default_vendor_file = os.path.join(os.path.abspath(os.path.dirname(sys.argv[0])), '..', 'vendor', 'vendor.json')

    parser = argparse.ArgumentParser()
    parser.add_argument('-f', '--vendor-file',
                        default=default_vendor_file, type=argparse.FileType(mode='r'), help='path to vendor.json')
    parser.add_argument('-n', '--dryrun', action='store_true',
                        help='do not perform updates, just list what would be updated')
    parser.add_argument('--origin', default='github.com/elastic/beats/', help='update deps of this origin prefix')
    parser.add_argument('-q', '--quiet', dest='verbose', action='store_false', help='work quietly')
    parser.add_argument('-d', '--done', action='store_true', help='mark vendor.json as patched')
    args = parser.parse_args()

    vendor_file = args.vendor_file
    done = args.done
    origin = args.origin

    modified = False
    vendor_filename = os.path.abspath(vendor_file.name)
    content = json.load(vendor_file)
    vendor_file.close()

    if comment in content['comment']:
        print('{} already patched.'.format(vendor_filename))
        return

    for pkg in content['package']:
        path = pkg.get('path', '')
        if 'origin' not in pkg and not path.startswith(origin):
            pkg['origin'] = '{}vendor/{}'.format(origin, path)
            modified = True

    if done:
        content['comment'] = '. '.join([comment, content['comment']])

    if not modified:
        print('No need to update {}.'.format(vendor_filename))

    if args.dryrun:
        print(json.dumps(content, indent=4, sort_keys=True))
        return

    with open(vendor_filename, "w") as vendor_file:
        json.dump(content, vendor_file, indent=4)

    print('{} has been patched.'.format(vendor_filename))


if __name__ == '__main__':
    main()
