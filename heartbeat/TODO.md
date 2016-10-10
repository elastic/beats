## TODO
- add docs:
  - configuration
- create kibana dashboards
- fsevent file watcher for module config loading
- all monitors:
  - configure local ip/device use to ping remote
  - add go based tests
- icmp monitor:
  - add windows support
  - check for non-root alternative not requiring RAW socket
  - add support for optional icmp payload
- http monitor:
  - refine allowed HTTP methods
  - add cookie jar for request (store new cookies or preset?)
  - configure HTTP version (http module might choose 1.1 or 2.0)
  - collect and report all response validation failures (ATM first failed validation will
    be reported)
  - add more compression types (only gzip supported yet)
- tcp monitor:
  - if receive validator fails, report received and expected value
- DNS probe:
  - collect (validate) known DNS entries right from DNS servers

## Ideas
- active monitors for more protocols: UDP, DNS, FTP, SIP, FTP, POP3, IMAP, MySQL, ...
- traceroute-ping
- API for listing active monitors and (modifying) schedule?
- API/Support for adding/removing monitoring targets?
  - HTTP based API? Need persistent registry with active list of hosts between restarts.
  - configure and watch file with potential endpoints (update monitors on file change)
  - monitor config file and reload on change?
- passive monitors based on packet sniffing:
  - IP, TCP: based monitor by checking for packages with actual payload
  - application layer (e.g. HTTP): check responses only + stop
    parsing/processing new connections in time interval if server is marked as
    up.
