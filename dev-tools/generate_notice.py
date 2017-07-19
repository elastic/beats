from __future__ import print_function

import glob
import os
import datetime
import argparse
import json
import csv
import re


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


def read_versions(vendor):
    libs = []
    with open(os.path.join(vendor, "vendor.json")) as f:
        govendor = json.load(f)
        for package in govendor["package"]:
            libs.append(package)
    return libs


def gather_dependencies(vendor_dirs):
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

                    lib["license_contents"] = read_file(lib["license_file"])
                    lib["license_summary"] = detect_license_summary(lib["license_contents"])
                    if lib["license_summary"] == "Unknown":
                        print("WARNING: Unknown license for: {}".format(lib_path))

                    if lib_path not in dependencies:
                        dependencies[lib_path] = [lib]
                    else:
                        dependencies[lib_path].append(lib)

            # don't walk down into another vendor dir
            if "vendor" in dirs:
                dirs.remove("vendor")
    return dependencies


def write_notice_file(f, beat, copyright, dependencies):

    now = datetime.datetime.now()

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

    # Sort licenses by package path, ignore upper / lower case
    for key in sorted(dependencies, key=str.lower):
        for lib in dependencies[key]:
            f.write("\n--------------------------------------------------------------------\n")
            f.write("Dependency: {}\n".format(key))
            if "version" in lib:
                f.write("Version: {}\n".format(lib["version"]))
            if "revision" in lib:
                f.write("Revision: {}\n".format(lib["revision"]))
            f.write("License type (autodetected): {}\n".format(lib["license_summary"]))
            f.write("{}:\n".format(lib["license_file"]))
            f.write("--------------------------------------------------------------------\n")
            if lib["license_summary"] != "Apache License 2.0":
                f.write(lib["license_contents"])
            else:
                # it's an Apache License, so include only the NOTICE file
                f.write("Apache License 2.0\n\n")
                for notice_file in glob.glob(os.path.join(os.path.dirname(lib["license_file"]), "NOTICE*")):
                    notice_file_hdr = "-------{}-----\n".format(os.path.basename(notice_file))
                    f.write(notice_file_hdr)
                    f.write(read_file(notice_file))


def write_csv_file(csvwriter, dependencies):
    csvwriter.writerow(["Dependency", "Version", "Revision", "License type (autodetected)", "License text"])
    for key in sorted(dependencies, key=str.lower):
        for lib in dependencies[key]:
            csvwriter.writerow([key, lib.get("version", ""), lib.get("revision", ""),
                                lib["license_summary"], lib["license_contents"]])


def create_notice(filename, beat, copyright, vendor_dirs, csvfile):
    dependencies = gather_dependencies(vendor_dirs)
    if not csvfile:
        with open(filename, "w+") as f:
            write_notice_file(f, beat, copyright, dependencies)
    else:
        with open(csvfile, "wb") as f:
            csvwriter = csv.writer(f)
            write_csv_file(csvwriter, dependencies)


APACHE2_LICENSE_TITLES = [
    "Apache License Version 2.0",
    "Apache License, Version 2.0"
]

MIT_LICENSES = [
    re.sub(r"\s+", " ", """Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.
    """),
    re.sub(r"\s+", " ", """Permission to use, copy, modify, and distribute this software for any
purpose with or without fee is hereby granted, provided that the above
copyright notice and this permission notice appear in all copies.""")
]

BSD_LICENSE_CONTENTS = [
    re.sub(r"\s+", " ", """Redistribution and use in source and binary forms, with or without modification,
are permitted provided that the following conditions are met:"""),
    re.sub(r"\s+", " ", """Redistributions of source code must retain the above copyright notice, this
   list of conditions and the following disclaimer."""),
    re.sub(r"\s+", " ", """Redistributions in binary form must reproduce the above copyright notice,
   this list of conditions and the following disclaimer in the documentation
   and/or other materials provided with the distribution.
""")]

BSD_LICENSE_3_CLAUSE = [
    re.sub(r"\s+", " ", """Neither the name of"""),
    re.sub(r"\s+", " ", """nor the
      names of its contributors may be used to endorse or promote products
      derived from this software without specific prior written permission.""")
]

BSD_LICENSE_4_CLAUSE = [
    re.sub(r"\s+", " ", """All advertising materials mentioning features or use of this software
   must display the following acknowledgement"""),
]

MPL_LICENSE_TITLES = [
    "Mozilla Public License Version 2.0",
    "Mozilla Public License, version 2.0"
]


def detect_license_summary(content):
    # replace all white spaces with a single space
    content = re.sub(r"\s+", ' ', content)
    if any(sentence in content[0:1000] for sentence in APACHE2_LICENSE_TITLES):
        return "Apache License 2.0"
    if any(sentence in content[0:1000] for sentence in MIT_LICENSES):
        return "MIT license"
    if all(sentence in content[0:1000] for sentence in BSD_LICENSE_CONTENTS):
        if all(sentence in content[0:1000] for sentence in BSD_LICENSE_3_CLAUSE):
            if all(sentence in content[0:1000] for sentence in BSD_LICENSE_4_CLAUSE):
                return "BSD 4-clause license"
            return "BSD 3-clause license"
        else:
            return "BSD 2-clause license"
    if any(sentence in content[0:300] for sentence in MPL_LICENSE_TITLES):
        return "Mozilla Public License 2.0"

    return "Unknown"


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
    parser.add_argument("--csv", dest="csvfile",
                        help="Output to a csv file")

    args = parser.parse_args()

    cwd = os.getcwd()
    notice = os.path.join(cwd, "NOTICE")
    vendor_dirs = []

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
    create_notice(notice, args.beat, args.copyright, vendor_dirs, args.csvfile)

    print("Available at {}".format(notice))
