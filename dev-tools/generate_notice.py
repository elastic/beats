import glob
import sys
import os
import datetime
import argparse


def read_file(filename):

    if not os.path.isfile(filename):
        print("File not found {}".format(filename))
        return ""

    with open(filename, 'r') as f:
        file_content = f.read()
        return file_content


def get_library_name(license):

    lib = ""
    path = os.path.dirname(license)
    # get the last three directories
    for i in range(0, 3):
        path, x = os.path.split(path)
        if len(lib) == 0:
            lib = x
        elif len(x) > 0:
            lib = x + "/" + lib

    return lib


def add_licenses(f, licenses):

    for license in licenses:
        for license_file in glob.glob(license):
            f.write("\n--------------------------------------------------------------------\n")
            f.write("{}\n".format(get_library_name(license_file)))
            f.write("--------------------------------------------------------------------\n")
            copyright = read_file(license_file)
            if "Apache License" not in copyright:
                f.write(copyright)
            else:
                # it's an Apache License, so include only the NOTICE file
                f.write("Apache License\n\n")
                for notice_file in glob.glob(os.path.join(os.path.dirname(license_file), "NOTICE*")):
                    f.write("-------{}-----\n".format(os.path.basename(notice_file)))
                    f.write(read_file(notice_file))


def create_notice(filename, beat, copyright, licenses):

    now = datetime.datetime.now()

    with open(filename, "w+") as f:

        # Add header
        f.write("{}\n".format(beat))
        f.write("Copyright 2014-{0} {1}\n".format(now.year, copyright))
        f.write("\n")
        f.write("This product includes software developed by The Apache Software \nFoundation (http://www.apache.org/).\n\n")

        # Add licenses for 3rd party libraries
        f.write("==========================================================================\n")
        f.write("Third party libraries used by the Beats project:\n")
        f.write("==========================================================================\n\n")
        add_licenses(f, licenses)


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
    licenses = []

    for root, dirs, files in os.walk(args.vendor):
        if 'vendor' in dirs:
            license = os.path.join(os.path.join(root, 'vendor'),
                                   '**/**/**/LICENSE*')
            licenses.append(license)

    print("Get the licenses available from {}".format(licenses))
    create_notice(notice, args.beat, args.copyright, licenses)

    print("Available at {}\n".format(notice))
