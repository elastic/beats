
import json
import os
import argparse
import subprocess


def update(pkg_name):
    """Call govendor on the targeted golang/x packages"""

    vendor_file = os.path.join('vendor', 'vendor.json')
    target = 'golang.org/x/{}'.format(pkg_name)

    with open(vendor_file) as content:
        deps = json.load(content)
    packages = [dep['path'] for dep in deps['package'] if dep['path'].startswith(target)]
    revision = '@{revision}'.format(revision=args.revision) if args.revision else ''
    packages = ['{pkg}{revision}'.format(pkg=pkg, revision=revision) for pkg in packages]
    cmd = ['govendor', 'fetch'] + packages
    if args.verbose:
        print(' '.join(cmd))
    subprocess.check_call(cmd)


def get_parser():
    """Creates parser to parse script params
    """
    parser = argparse.ArgumentParser(description="Update golang.org/x/<name> in vendor folder")
    parser.add_argument('-q', '--quiet', dest='verbose', action='store_false', help='work quietly')
    parser.add_argument('--revision', help='update deps to this revision', default='')
    parser.add_argument('name', help='name of the golang.org/x/ package. Can be empty', default='', nargs='?')
    return parser


if __name__ == "__main__":

    parser = get_parser()
    args = parser.parse_args()

    update(args.name)
