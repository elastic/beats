from __future__ import print_function

import glob
import os
import datetime
import argparse
import json


def read_file(filename):

    if not os.path.isfile(filename):
        print("File not found {}".format(filename))
        return ""

    try:
        with open(filename, 'r') as f:
            return f.read()
    except UnicodeDecodeError:
        # try latin-1
        with open(filename, 'r', encoding="ISO-8859-1") as f:
            return f.read()


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
    dependencies = {}   # lib_path -> [array of lib]
    for vendor in vendor_dirs:
        libs = read_versions(vendor)

        # walk looking for LICENSE files
        for root, dirs, filenames in os.walk(vendor):
            for filename in sorted(filenames):
                if filename.startswith("LICENSE"):
                    lib_path = get_library_path(root)
                    lib_search = [l for l in libs if l["path"].startswith(lib_path)]
                    if len(lib_search) == 0:
                        print("WARNING: No version information found for: {}".format(lib_path))
                        lib = {"path": lib_path}
                    else:
                        lib = lib_search[0]
                    lib["license_file"] = os.path.join(root, filename)

                    if lib_path not in dependencies:
                        dependencies[lib_path] = [lib]
                    else:
                        dependencies[lib_path].append(lib)

            # don't walk down into another vendor dir
            if "vendor" in dirs:
                dirs.remove("vendor")

    # Sort licenses by package path, ignore upper / lower case
    for key in sorted(dependencies, key=str.lower):
        for lib in dependencies[key]:
            f.write("\n--------------------------------------------------------------------\n")
            f.write("Dependency: {}\n".format(key))
            if "version" in lib:
                f.write("Version: {}\n".format(lib["version"]))
            if "revision" in lib:
                f.write("Revision: {}\n".format(lib["revision"]))
            f.write("{}:\n".format(lib["license_file"]))
            f.write("--------------------------------------------------------------------\n")
            copyright = read_file(lib["license_file"])
            if "Apache License" not in copyright:
                f.write(copyright)
            else:
                # it's an Apache License, so include only the NOTICE file
                f.write("Apache License\n\n")
                for notice_file in glob.glob(os.path.join(os.path.dirname(lib["license_file"]), "NOTICE*")):
                    notice_file_hdr = "-------{}-----\n".format(os.path.basename(notice_file))
                    f.write(notice_file_hdr)
                    f.write(read_file(notice_file))


def read_versions(vendor):
    libs = []
    with open(os.path.join(vendor, "vendor.json")) as f:
        govendor = json.load(f)
        for package in govendor["package"]:
            libs.append(package)
    return libs


def create_notice(filename, beat, copyright, vendor_dirs):

    now = datetime.datetime.now()

    with open(filename, "w+") as f:

        # Add header
        f.write("{}\n".format(beat))
        f.write("Copyright 2014-{0} {1}\n".format(now.year, copyright))
        f.write("\n")
        f.write("This product includes software developed by The Apache Software \n" +
                "Foundation (http://www.apache.org/).\n\n")

        # Add licenses for 3rd party libraries
        f.write("==========================================================================\n")
        f.write("Third party libraries used by the Beats project:\n")
        f.write("==========================================================================\n\n")
        add_licenses(f, vendor_dirs)


EXCLUDES = ["dev-tools"]

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
            dirs.remove('vendor')   # don't walk down into sub-vendors

        for exclude in EXCLUDES:
            if exclude in dirs:
                dirs.remove(exclude)

    print("Get the licenses available from {}".format(vendor_dirs))
    create_notice(notice, args.beat, args.copyright, vendor_dirs)

    print("Available at {}\n".format(notice))
