These Scripts are used to create the integration tests pcap files.

Requirements:

- A memcached service must be running and listening for
  UDP/TCP traffic both on the same port number.
- tcpdump plus rights to create pcap (run gen_all.sh with sudo if required)


Use the gen_all.sh script to run all experiments and create new traces:

./gen_all.sh <ifc> <outdir> <memcached-host> <memcached-port>

  - <ifc>  : The network interface tcpdump will capture packets from
  - <outdir> : output directory for generated pcaps.
               Use ../../pcaps to store generated traces with the integration tests
               Defaults to the current working directory.
  - <memcached-host> : ip or hostname of memcached server (defaults to 127.0.0.1)
  - <memcached-port> : port memcached service listens on (default: 11211)

After generating traces check UDP based traces for packets being lost. If all traces are ok, commit them to the repository.
