## Readme

Code generator for packetbeat tcp based protocol analyzers.

In order to create a new protocol analyzer, run inside your GOPATH where you
want to create the protocol analyzer (stand-alone, within packetbeat based
project or packetbeat itself):

```
python ${GOPATH}/src/github.com/elastic/beats/packetbeat/scripts/create_tcp_protocol.py
```

Note: If you have multiple go paths use `${GOPATH%%:*}`instead of `${GOPATH}`.

This requires [python](https://www.python.org/downloads/) to be installed.

## Tutorial (TODO):

### 1. Implement protocol analyzer for simple echo server based protocol:

  - client: Send request to echo server. All requests must start with `>`
    character and end with newline character `\n`
  - server: Send echo response upon receiving a message. Echo response begins
    with `<` character. Errors responses begin with `!` character. An error message
    will be returned for any received request not starting with `>`.

- Echo Server sample code:

```
package main

/*
Echo protocol test server listening on port 3030
First character in message indicates message type:
'>': request
'<': response
'!': error response
Sample test run:
    $ nc localhost 3030
    >abc
    <abc
    asdfkjhasf
    !asdfkjhasf
    > 123456
    < 123456
    asdfasdf
    !asdfasdf
    ^C
*/

import (
	"bufio"
	"net"
	"os"
)

func main() {
	bind := ":3030"
	server, err := net.Listen("tcp", bind)
	if server == nil {
		panic("couldn't start listening: " + err.Error())
	}

	for {
		client, err := server.Accept()
		if err != nil {
			panic("failed accepting new client: " + err.Error())
		}

		go echo(client)
	}
}

func echo(sock net.Conn) {
	defer sock.Close()

	in := bufio.NewReader(sock)
	for {
		line, err := in.ReadBytes('\n')
		if err != nil {
			os.Stderr.Write([]byte(err.Error()))
			return
		}

		if len(line) == 0 {
			continue
		}

		if line[0] == '>' {
			line[0] = '<'
			sock.Write(line)
		} else {
			sock.Write([]byte{'!'})
			sock.Write(line)
		}
	}
}
```

### 2.1 Add protocol analyzer (echo) to packetbeat:

Create analyzer skeleton from code generator template.

```
  $ cd ${GOPATH}/src/github.com/elastic/beats/packetbeat
  $ python ${GOPATH}/src/github.com/elastic/beats/packetbeat/scripts/create_tcp_protocol.py
```

Load plugin into packetbeat by running `make update`. Or add `_
"github.com/elastic/beats/v8/packetbeat/protos/echo"` to the import list in
`$GOPATH/src/github.com/elastic/beats/packetbeat/include/list.go`.

### 2.2 Standalone beat with protocol analyzer (echo):

Use packetbeat as framework to build custom beat (e.g. for testing) with
selected protocol plugins only. A protocol plugin can still be added to
packetbeat later by copying the final plugin to
`$GOPATH/src/github.com/elastic/beats/packetbeat/protos` and importing module in
`$GOPATH/src/github.com/elastic/beats/packetbeat/include/list.go`.

Create custom beat (e.g. github.com/<username>/pb_echo):

```
$ mkdir -p ${GOPATH}/src/github.com/<username>/pb_echo
$ cd ${GOPATH}/src/github.com/<username>/pb_echo
```

Add main.go importing packetbeat + new protocol (to be added to pb_echo/proto)
package main

```
import (
	"os"

	"github.com/elastic/beats/v8/libbeat/beat"
	"github.com/elastic/beats/v8/packetbeat/beater"

	// import supported protocol modules
	_ "github.com/urso/pb_echo/protos/echo"
)

var Name = "pb_echo"

// Setups and Runs Packetbeat
func main() {
	if err := beat.Run(Name, "", beater.New); err != nil {
		os.Exit(1)
	}
}
```

Create protocol analyzer module (use name ‘echo’ for new protocol):

```
$ mkdir proto
$ cd proto
$ python ${GOPATH}/src/github.com/elastic/beats/packetbeat/scripts/create_tcp_protocol.py
```

### 3 Implement application layer analyzer

Protocol analyzer structure for `echo` module.

- `config.go`: protocol analyzer configuration options + validation
- `echo.go`: protocol analyzer module keeping track of state in TCP context
- `parser.go`: message parser implementation.
- `trans.go`: correlate messages into transactions. Simple implementation with
  support for pipelining already provided.
- `pub.go`: create+publish events. Generic event fields are already set

Protocol analyzers to receive raw TCP payloads from packetbeat and must combine
and parse them into messages `parser.go`. The parser state is stored in
`connection`-struct in `echo.go`. Parsed messages are forwarded for merging with
previously parsed messages in same direction (for example if responses consist
of multiple messages being streamed) and correlation of requests and responses
(See `trans.go`). Some simple message correlation with pipelining support is
already provided. Once messages have been correlated a transaction event created
in `createEvent` (file `pub.go`) is published. Common event fields are already
populated by generated code. Do not remove these common event fields.

### 3.1 Add parser

Add code to parse message from network stream to `func (*parser) parse(...)`:

```
type message struct {
    ...

	failed  bool
	content common.NetString
}

...

func (p *parser) parse() (*message, error) {
	// wait for message being complete
	buf, err := p.buf.CollectUntil([]byte{'\n'})
	if err == streambuf.ErrNoMoreBytes {
		return nil, nil
	}

	msg := p.message
	msg.Size = uint64(p.buf.BufferConsumed())

	isRequest := true
	dir := applayer.NetOriginalDirection
	if len(buf) > 0 {
		c := buf[0]
		isRequest = !(c == '<' || c == '!')
		if !isRequest {
			msg.failed = c == '!'
			dir = applayer.NetReverseDirection
		}
		buf = buf[1:]
	}

	msg.content = common.NetString(buf)
	msg.IsRequest = isRequest
	msg.Direction = dir

	return msg, nil
}

```

If possible you can use third-party libraries for parsing messages. This might require some more changes to the parser struct.

### 3.2 Add additional fields to transaction event

```
func (pub *transPub) createEvent(requ, resp *message) beat.Event {
	status := common.OK_STATUS
	if resp.failed {
		status = common.ERROR_STATUS
	}

	// resp_time in milliseconds
	responseTime := int32(resp.Ts.Sub(requ.Ts).Nanoseconds() / 1e6)

	src := &common.Endpoint{
		IP:   requ.Tuple.SrcIP.String(),
		Port: requ.Tuple.SrcPort,
		Proc: string(requ.CmdlineTuple.Src),
	}
	dst := &common.Endpoint{
		IP:   requ.Tuple.DstIP.String(),
		Port: requ.Tuple.DstPort,
		Proc: string(requ.CmdlineTuple.Dst),
	}

	fields := common.MapStr{
		"type":         "echo",
		"status":       status,
		"responsetime": responseTime,
		"bytes_in":     requ.Size,
		"bytes_out":    resp.Size,
		"src":          src,
		"dst":          dst,
	}

	// add processing notes/errors to event
	if len(requ.Notes)+len(resp.Notes) > 0 {
		fields["notes"] = append(requ.Notes, resp.Notes...)
	}

	if pub.sendRequest {
		fields["request"] = requ.content
	}
	if pub.sendResponse {
		fields["response"] = requ.content
	}

	return beat.Event{
		Timestamp: requ.Ts,
		Fields: fields,
	}
}
```

### 4 (TODO) Add protocol analyzer module to config files

### 5 (TODO) Build kibana dashboard

### 6 Tips

- Prepare pcap (with tcpdump) for testing protocol analyzer during development

- At least add tests using pcaps to system/test.
