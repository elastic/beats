from __future__ import print_function

import glob
import os
import datetime
import argparse
import six


def read_file(filename):

    if not os.path.isfile(filename):
        print("File not found {}".format(filename))
        return ""

    with open(filename, 'rb') as f:
        file_content = f.read()
        return file_content


def get_library_path(license):
    """
    Get the contents up to the vendor folder.
    """
    split = license.split(os.sep)
    for i, word in reversed(list(enumerate(split))):
        if word == "vendor":
            return "/".join(split[i + 1:])
    return "/".join(split)


def add_licenses(f, vendor_dirs):
    licenses = {}
    for vendor in vendor_dirs:
        # walk looking for LICENSE files
        for root, dirs, filenames in os.walk(vendor):
            for filename in sorted(filenames):
                if filename.startswith("LICENSE"):
                    lib_path = get_library_path(root)
                    if lib_path in licenses:
                        print("WARNING: Dependency appears multiple times: {}".format(lib_path))
                    licenses[lib_path] = os.path.abspath(os.path.join(root, filename))

    # Sort licenses by package path, ignore upper / lower case
    for key in sorted(licenses, key=str.lower):
        license_file = licenses[key]
        f.write(six.b("\n--------------------------------------------------------------------\n"))
        f.write(six.b("{}\n".format(key)))
        f.write(six.b("--------------------------------------------------------------------\n"))
        copyright = read_file(license_file)
        if six.b('Apache License') not in copyright:
            f.write(copyright)
        else:
            # it's an Apache License, so include only the NOTICE file
            f.write(six.b("Apache License\n\n"))
            for notice_file in glob.glob(os.path.join(os.path.dirname(license_file), "NOTICE*")):
                notice_file_hdr = "-------{}-----\n".format(os.path.basename(notice_file))
                f.write(six.b(notice_file_hdr))
                f.write(read_file(notice_file))


def create_notice(filename, beat, copyright, vendor_dirs):

    now = datetime.datetime.now()

    with open(filename, "wb+") as f:

        # Add header
        f.write(six.b("{}\n".format(beat)))
        f.write(six.b("Copyright 2014-{0} {1}\n".format(now.year, copyright)))
        f.write(six.b("\n"))
        f.write(six.b("This product includes software developed by The Apache Software \n") +
                six.b("Foundation (http://www.apache.org/).\n\n"))

        # Add licenses for 3rd party libraries
        f.write(six.b("==========================================================================\n"))
        f.write(six.b("Third party libraries used by the Beats project:\n"))
        f.write(six.b("==========================================================================\n\n"))
        add_licenses(f, vendor_dirs)


if __name__ == "__main__":

    parser = argparse.ArgumentParser(
        description="Generate the NOTICE file from all vendor directories available in a given directory")
    parser.add_argument("vendor",
                        help="directory where to search for vendor directories")
    parser.add_argument("-b", "--beat", default="Elastic Beats",
                        help="Beat name")
    parser.add_argument("-c", "--copyright", default="Elasticsearch BV",
                        help="copyright owner")

    args = parser.parse_args()

    cwd = os.getcwd()
    notice = os.path.join(cwd, "NOTICE")
    vendor_dirs = []

    print(args.vendor)

    for root, dirs, files in os.walk(args.vendor):

        # Skips all hidden paths like ".git"
        if '/.' in root:
            continue

        if 'vendor' in dirs:
            vendor_dirs.append(os.path.join(root, 'vendor'))

    print("Get the licenses available from {}".format(vendor_dirs))
    create_notice(notice, args.beat, args.copyright, vendor_dirs)

    print("Available at {}\n".format(notice))
