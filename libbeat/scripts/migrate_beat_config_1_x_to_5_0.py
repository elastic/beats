#!/usr/bin/env python
import argparse
import os
import re


def collect_lines(fn):
    return lambda content: "\n".join(fn(content)) + "\n"


def process_lines(fn):
    def go(content):
        return (l for line in content.splitlines() for l in fn(line))
    return collect_lines(go)


@process_lines
def migrate_packetbeat(line):
    """
    Changes things like `interfaces:` to `packetbeat.interfaces:`
    at the top level.
    """
    sections = ["interfaces", "protocols", "procs",
                "runoptions", "ignore_outgoing"]
    for sec in sections:
        if line.startswith(sec + ":"):
            return ["packetbeat." + line]
    return [line]


@collect_lines
def migrate_shipper(content):
    """
    Moves everything under the `shipper:` section to be top level.
    """
    state = "out"
    for line in content.splitlines():
        if state == "out":
            if line.startswith("shipper:"):
                state = "in"
                # eat line
            else:
                yield line
        elif state == "in":
            if line.startswith("  "):
                yield line[2:]
            elif line.startswith("\t"):
                yield line[1:]
            elif line == "":
                yield line
            else:
                yield line
                state = "out"


@collect_lines
def migrate_tls_settings(content):
    """
    Updates tls section, changing tls to ssl and adapting field changes.
    """
    state = "out"
    indent = 0
    comments = []

    keep_settings = [
        'certificate_authorities',
        'certificate',
        'cipher_suites',
        'curve_types'
    ]

    rename_settings = {
        "certificate_key": "key",
    }

    regex_replace_settings = {
        "insecure": [
            [re.compile(".*insecure.*false"), "verification_mode: full"],
            [re.compile(".*insecure.*true"), "verification_mode: none"],
        ]
    }

    version_indent = [None]
    min_version = [None]
    max_version = [None]
    min_version_used = max_version_used = False

    ssl_versions_old = {"SSL-3.0": 0, "1.0": 1, "1.1": 2, "1.2": 3}
    ssl_versions_new = ["SSLv3", "TLSv1.0", "TLSv1.1", "TLSv1.2"]

    def get_old_tls_version(v, default):
        if v not in ssl_versions_old:
            return ssl_versions_old[default]
        return ssl_versions_old[v]

    def make_version_info():
        if version_indent[0] is None:
            return

        indent = version_indent[0]
        commented_out = not (min_version_used or max_version_used)
        v_start = min_version[0]
        v_end = max_version[0]

        if min_version_used != max_version_used:
            if not min_version_used:
                v_start = "1.0"
            if not max_version_used:
                v_end = "1.2"

        v_start = get_old_tls_version(v_start, "1.0")
        v_end = get_old_tls_version(v_end, "1.2")
        versions = (ssl_versions_new[i] for i in xrange(v_start, v_end+1))

        line = indent * ' ' + ('#' if commented_out else '')
        line += "supported_protocols:"
        line += "[" + ", ".join(versions) + "]"

        yield ""
        yield line

        version_indent[0] = None
        min_version[0] = None
        max_version[0] = None

    for line in content.splitlines():
        tmp = line.expandtabs()
        line_indent = len(tmp) - len(tmp.lstrip())
        line_start = tmp.lstrip()
        tmp = line_start.split(':', 1)
        setting = None
        value = None
        commented_out = len(line_start) > 0 and line_start[0] == '#'
        if len(tmp) > 1:
            setting = tmp[0]
            value = tmp[1].strip()
            if setting[0] == '#':
                setting = setting[1:]

        def create_setting(l):
            return (line_indent * " ") + ('#' if commented_out else '') + l

        if state == "out":
            if setting == "tls":
                state = "in"
                indent = line_indent
                yield create_setting("ssl:")
            else:
                yield line
        elif state == "in":
            if setting is not None and line_indent <= indent:
                for l in make_version_info():
                    yield l
                # last few comments have been part of next line -> print
                for l in comments:
                    yield l
                yield line
                comments = []
                state = "out"
            elif setting is None:
                comments.append(line)
            elif setting in keep_settings:
                for c in comments:
                    yield c
                comments = []
                yield line
            elif setting in rename_settings:
                new_name = rename_settings[setting]
                for c in comments:
                    yield c
                comments = []
                yield line.replace(setting, new_name, 1)
            elif setting in regex_replace_settings:
                # drop comments and add empty line before new setting
                comments = []
                yield ""

                for pattern in regex_replace_settings[setting]:
                    regex, val = pattern
                    if regex.match(line):
                        yield create_setting(regex.sub(line, val, 1))
                        break
            elif setting == 'min_version':
                comments = []
                min_version[0] = value
                min_version_used = not commented_out
                version_indent[0] = line_indent
            elif setting == 'max_version':
                comments = []
                max_version[0] = value
                max_version_used = not commented_out
                version_indent[0] = line_indent
            else:
                yield line
        else:
            yield line

    # add version info in case end of output is SSL/TLS section
    if state == 'in':
        for l in make_version_info():
            yield l


def main():
    # List of migrations to apply. Shipper must be migrated first for
    # ignore_outgoing to be applied properly
    migrations = [
        migrate_shipper,
        migrate_packetbeat,
        migrate_tls_settings]

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
        for m in migrations:
            content = m(content)

    if args.dry:
        print(content)
    else:
        os.rename(args.file, args.file + ".bak")
        print("Backup file created: {}".format(args.file + ".bak"))
        with open(args.file, "w") as f:
            f.write(content)


if __name__ == "__main__":
    main()


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
ignore_outgoing: true
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
packetbeat.ignore_outgoing: true
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

def test_migrate_tls_settings():
    test = """
output:
  # Elasticsearch output
  elasticsearch:
    # tls configuration. By default is off.
    tls:
      # List of root certificates for HTTPS server verifications
      certificate_authorities: ["/etc/pki/root/ca.pem"]

      # Certificate for TLS client authentication
      #certificate: "/etc/pki/client/cert.pem"

      # Client Certificate Key
      #certificate_key: "/etc/pki/client/cert.key"

      # Controls whether the client verifies server certificates and host name.
      # If insecure is set to true, all server host names and certificates will be
      # accepted. In this mode TLS based connections are susceptible to
      # man-in-the-middle attacks. Use only for testing.
      #insecure: true

      # Configure cipher suites to be used for TLS connections
      #cipher_suites: []

      # Configure curve types for ECDHE based cipher suites
      #curve_types: []

      # Configure minimum TLS version allowed for connection to logstash
      min_version: 1.1

      # Configure maximum TLS version allowed for connection to logstash
      max_version: 1.2

  # Logstash output.
  #logstash:
    # tls configuration. By default is off.
    #tls:
      # List of root certificates for HTTPS server verifications
      #certificate_authorities: ["/etc/pki/root/ca.pem"]

      # Certificate for TLS client authentication
      #certificate: "/etc/pki/client/cert.pem"

      # Client Certificate Key
      #certificate_key: "/etc/pki/client/cert.key"

      # Controls whether the client verifies server certificates and host name.
      # If insecure is set to true, all server host names and certificates will be
      # accepted. In this mode TLS based connections are susceptible to
      # man-in-the-middle attacks. Use only for testing.
      #insecure: true

      # Configure cipher suites to be used for TLS connections
      #cipher_suites: []

      # Configure curve types for ECDHE based cipher suites
      #curve_types: []

      # Configure minimum TLS version allowed for connection to logstash
      #min_version: 1.0

      # Configure maximum TLS version allowed for connection to logstash
      #max_version: 1.2

  # Redis output
  redis:
    tls:
      # List of root certificates for HTTPS server verifications
      certificate_authorities: ["/etc/pki/root/ca.pem"]

      # Certificate for TLS client authentication
      certificate: "/etc/pki/client/cert.pem"

      # Client Certificate Key
      certificate_key: "/etc/pki/client/cert.key"

      # Controls whether the client verifies server certificates and host name.
      # If insecure is set to true, all server host names and certificates will be
      # accepted. In this mode TLS based connections are susceptible to
      # man-in-the-middle attacks. Use only for testing.
      insecure: false

      # Configure cipher suites to be used for TLS connections
      #cipher_suites: []

      # Configure curve types for ECDHE based cipher suites
      #curve_types: []

      # Configure minimum TLS version allowed for connection to logstash
      #min_version: 1.1

      # Configure maximum TLS version allowed for connection to logstash
      max_version: 1.1
    """

    output = migrate_tls_settings(test)
    assert output == """
output:
  # Elasticsearch output
  elasticsearch:
    # tls configuration. By default is off.
    ssl:
      # List of root certificates for HTTPS server verifications
      certificate_authorities: ["/etc/pki/root/ca.pem"]

      # Certificate for TLS client authentication
      #certificate: "/etc/pki/client/cert.pem"

      # Client Certificate Key
      #key: "/etc/pki/client/cert.key"

      #verification_mode: none

      # Configure cipher suites to be used for TLS connections
      #cipher_suites: []

      # Configure curve types for ECDHE based cipher suites
      #curve_types: []

      supported_protocols:[TLSv1.1, TLSv1.2]

  # Logstash output.
  #logstash:
    # tls configuration. By default is off.
    #ssl:
      # List of root certificates for HTTPS server verifications
      #certificate_authorities: ["/etc/pki/root/ca.pem"]

      # Certificate for TLS client authentication
      #certificate: "/etc/pki/client/cert.pem"

      # Client Certificate Key
      #key: "/etc/pki/client/cert.key"

      #verification_mode: none

      # Configure cipher suites to be used for TLS connections
      #cipher_suites: []

      # Configure curve types for ECDHE based cipher suites
      #curve_types: []

      #supported_protocols:[TLSv1.0, TLSv1.1, TLSv1.2]

  # Redis output
  redis:
    ssl:
      # List of root certificates for HTTPS server verifications
      certificate_authorities: ["/etc/pki/root/ca.pem"]

      # Certificate for TLS client authentication
      certificate: "/etc/pki/client/cert.pem"

      # Client Certificate Key
      key: "/etc/pki/client/cert.key"

      verification_mode: full

      # Configure cipher suites to be used for TLS connections
      #cipher_suites: []

      # Configure curve types for ECDHE based cipher suites
      #curve_types: []

      supported_protocols:[TLSv1.0, TLSv1.1]
"""
