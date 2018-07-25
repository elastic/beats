FreePort
========

Get a free open TCP port that is ready to use.

## Command Line Example:
```bash
# Ask the kernel to give us an open port.
export port=$(freeport)

# Start standalone httpd server for testing
httpd -X -c "Listen $port" &

# Curl local server on the selected port
curl localhost:$port
```

## Golang example:
```go
package main

import "github.com/phayes/freeport"

func main() {
	port, err := freeport.GetFreePort()
	if err != nil {
		log.Fatal(err)
	}
	// port is ready to listen on
}

```

## Installation

#### Mac OSX
```bash
brew install phayes/repo/freeport
```


#### CentOS and other RPM based systems
```bash
wget https://github.com/phayes/freeport/releases/download/1.0.2/freeport_1.0.2_linux_386.rpm
rpm -Uvh freeport_1.0.2_linux_386.rpm
```

#### Ubuntu and other DEB based systems
```bash
wget wget https://github.com/phayes/freeport/releases/download/1.0.2/freeport_1.0.2_linux_amd64.deb
dpkg -i freeport_1.0.2_linux_amd64.deb
```

#### Building From Source
```bash
sudo apt-get install golang                    # Download go. Alternativly build from source: https://golang.org/doc/install/source
go get github.com/phayes/freeport
```
