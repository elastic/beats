Please see the main [CONTRIBUTING](../CONTRIBUTING.md) file, and consider the
following notes:

* If you are planning to add a new protocol, please read our [developer guide
for adding new
protocols](https://www.elastic.co/guide/en/beats/devguide/current/new-protocol.html)
* Packetbeat uses Cgo, so in addition to having Go installed you need a C
  compiler
* Packetbeat depends on libpcap. You need to install libpcap-dev on Debian
  based systems or libpcap-devel on RedHat based systems. On Windows, you
  need winpcap.
