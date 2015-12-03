#!/usr/bin/env python

"""Simple script to concatenate coverage reports.
"""

import os
import sys
import argparse
import fnmatch

def main(arguments):

    parser = argparse.ArgumentParser(description=__doc__,
                                     formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument('dir', help="Input dir to search recursively for .cov files")
    parser.add_argument('-o', '--outfile', help="Output file",
                        default=sys.stdout, type=argparse.FileType('w'))

    args = parser.parse_args(arguments)

    # Recursively find all matching .cov files.
    matches = []
    for root, dirnames, filenames in os.walk(args.dir):
        for filename in fnmatch.filter(filenames, '*.cov'):
            matches.append(os.path.join(root, filename))

    # Write to output.
    args.outfile.write('mode: atomic\n')
    for m in matches:
        with open(m) as f:
            for line in f:
                if not line.startswith('mode:'):
                    args.outfile.write(line)

if __name__ == '__main__':
    sys.exit(main(sys.argv[1:]))
