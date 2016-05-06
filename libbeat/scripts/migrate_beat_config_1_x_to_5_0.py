#!/usr/bin/env python
import argparse
import os


def migrate_topbeat(content):
    """
    Changes top level `input:` into `topbeat:`
    """
    lines = content.splitlines()
    outlines = []
    for line in lines:
        if line.startswith("input:"):
            outlines.append("topbeat:")
        else:
            outlines.append(line)
    return "\n".join(outlines) + "\n"


def migrate_packetbeat(content):
    """
    Changes things like `interfaces:` to `packetbeat.interfaces:`
    at the top level.
    """
    sections = ["interfaces", "protocols", "procs", "runoptions"]
    lines = content.splitlines()
    outlines = []
    for line in lines:
        found = False
        for sec in sections:
            if line.startswith(sec + ":"):
                outlines.append("packetbeat." + line)
                found = True
                break
        if not found:
            outlines.append(line)
    return "\n".join(outlines) + "\n"


def migrate_shipper(content):
    """
    Moves everything under the `shipper:` section to be top level.
    """
    lines = content.splitlines()
    outlines = []
    state = "out"
    for line in lines:
        if state == "out":
            if line.startswith("shipper:"):
                state = "in"
                # eat line
            else:
                outlines.append(line)
        elif state == "in":
            if line.startswith("  "):
                outlines.append(line[2:])
            elif line.startswith("\t"):
                outlines.append(line[1:])
            elif line == "":
                outlines.append(line)
            else:
                outlines.append(line)
                state = "out"
    return "\n".join(outlines) + "\n"


def main():
    parser = argparse.ArgumentParser(
        description="Migrates beats configuration from 1.x to 5.0")
    parser.add_argument("file",
                        help="Configuration file to migrate")
    parser.add_argument("--dry", action="store_true",
                        help="Don't do any changes, just print the" +
                             " modified config to the screen")
    args = parser.parse_args()

    with open(args.file, "r") as f:
        content = f.read()
        out = migrate_packetbeat(content)
        out = migrate_topbeat(out)
        out = migrate_shipper(out)

    if args.dry:
        print(out)
    else:
        os.rename(args.file, args.file + ".bak")
        print("Backup file created: {}".format(args.file + ".bak"))
        with open(args.file, "w") as f:
            f.write(out)


if __name__ == "__main__":
    main()


def test_migrate_topbeat():
    test = """
input:
  # In seconds, defines how often to read server statistics
  period: 10

  # Regular expression to match the processes that are monitored
  # By default, all the processes are monitored
  procs: [".*"]
"""
    output = migrate_topbeat(test)
    assert output == """
topbeat:
  # In seconds, defines how often to read server statistics
  period: 10

  # Regular expression to match the processes that are monitored
  # By default, all the processes are monitored
  procs: [".*"]
"""


def test_migrate_packetbeat():
    test = """
# Select the network interfaces to sniff the data. You can use the "any"
# keyword to sniff on all connected interfaces.
interfaces:
  device: en0

############################# Protocols #######################################
protocols:
  dns:
    # Configure the ports where to listen for DNS traffic. You can disable
    # the DNS protocol by commenting out the list of ports.
    ports: [53]
runoptions:
procs:
"""

    output = migrate_packetbeat(test)
    assert output == """
# Select the network interfaces to sniff the data. You can use the "any"
# keyword to sniff on all connected interfaces.
packetbeat.interfaces:
  device: en0

############################# Protocols #######################################
packetbeat.protocols:
  dns:
    # Configure the ports where to listen for DNS traffic. You can disable
    # the DNS protocol by commenting out the list of ports.
    ports: [53]
packetbeat.runoptions:
packetbeat.procs:
"""


def test_migrate_shipper():
    test = """
############################# Shipper #########################################

shipper:
  # The name of the shipper that publishes the network data. It can be used to group
  # all the transactions sent by a single shipper in the web interface.
  # If this options is not defined, the hostname is used.
  name:

  # The tags of the shipper are included in their own field with each
  # transaction published. Tags make it easy to group servers by different
  # logical properties.
  #tags: ["service-X", "web-tier"]
test:
"""
    output = migrate_shipper(test)
    assert output == """
############################# Shipper #########################################

# The name of the shipper that publishes the network data. It can be used to group
# all the transactions sent by a single shipper in the web interface.
# If this options is not defined, the hostname is used.
name:

# The tags of the shipper are included in their own field with each
# transaction published. Tags make it easy to group servers by different
# logical properties.
#tags: ["service-X", "web-tier"]
test:
"""
