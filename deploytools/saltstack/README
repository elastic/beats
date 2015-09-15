This is salt state you can drop in your existing salt configuration to
automatically deploy Packetbeat to deb and rpm based hosts. It currently
only supports 64-bit hosts and has been tested on Ubuntu 14.04 and RedHat 6.

Config required

Change elasticsearch in init.sls to point to your elasticsearch cluster.
Change the list of procs to match the list of procs found in your environment.
Change the sources to match the latest packetbeat.

Deploy
Copy the packetbeat folder to your saltstack config.
This can be added to either your top file or deployed using state.sls.
It will resolve all of its dependencies automatically.
