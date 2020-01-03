# Go daemon

Go daemon (or just **god**) is a utility to "daemonize" Go programs
that originally only run in foreground and write logs to the console.

Go daemon can turn these programs into daemons by managing essential aspects
of their execution. The process of making a program become a daemon has very
peculiar steps and can be done outside the code. This is what **god** is for.

It executes a program for you doing things that daemons do: switch to another
user and group, switch the directory of execution, detach from the terminal
and create a pid file. While the program runs, **god** consumes its output
(stdout and stderr) and write to a log file *using minimum system resources*.

It also handles all signals (SIGINT, SIGTERM, etc) and forward them to the
program being managed. On SIGHUP, **god** recycles its log file making it
easy to integrate with logrotate. If SIGHUP is not supported by your program,
**god** can handle the signal itself and not forward it, making your program
immune to hangups.

Go daemon is inspired by [twistd](http://twistedmatrix.com/documents/current/core/howto/basics.html#auto1),
but primarily for running servers written in the
[Go Programming Language](http://golang.org) that don't (or just can't)
care about daemonizing. However, it can also be used for running php, python
and any other type of long lived programs that need to be daemonized.

A typical command line looks like this:

	god --nohup --logfile foo.log --pidfile foo.pid --user nobody --group nobody --rundir /opt/foo -- ./foobar --foobar-opts


## Why?

Like if there's not enough options out there: upstart, systemd, launchd,
daemontools, supervisord, runit, you name it. There's also utilities like
apache's logger, etc.

Go daemon aims at being as simple as possible in regards to deployment and
usage, and to run with minimum resources. It doesn't supervise the program,
just run it as a daemon and takes care of its console output. It mixes well
with upstart and logrotate, for example.

Go daemon is not needed on systems with [systemd](http://www.freedesktop.org/wiki/Software/systemd/).


## Building

Go daemon is written in C and needs to be compiled. Debian and Ubuntu can
install the compiler and tools with the following command:

	apt-get install build-essential

Then build and install it:

	make
	make install

The `god` command line tool should be ready to use.

### Binary packages

Go daemon can be packaged for both [debian](debian/README.Debian.md) and
[rpm](rpm/README.md) based systems and has been tested on Ubuntu, CentOS and
RHEL.

Ubuntu 12.04 packages are available at
<https://launchpad.net/~fiorix/+archive/go-daemon/> and can be installed
with following commands:

	apt-get install python-software-properties
	add-apt-repository ppa:fiorix/go-daemon
	apt-get update
	apt-get install go-daemon
