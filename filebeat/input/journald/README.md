# Journald input

The Journald input reads journal entries by calling `journalctl`.

## Adding entries to the journal
### Using `systemd-cat`
The easiest way to add entries to the journal is to use `systemd-cat`:
```
root@vagrant-debian-12:~/filebeat# echo "Hello Journal!" | systemd-cat
root@vagrant-debian-12:~/filebeat# journalctl -n 1
Oct 02 04:17:01 vagrant-debian-12 CRON[1912]: pam_unix(cron:session): session closed for user root
```

The syslog identifier can be specified with the `-t` parameter:
```
root@vagrant-debian-12:~/filebeat# echo "Hello Journal!" | systemd-cat -t my-test
root@vagrant-debian-12:~/filebeat# journalctl -n 1
Oct 02 04:17:50 vagrant-debian-12 my-test[1924]: Hello Journal!
```

### Writing to Journald's socket
The following Go program will write directly to Journald's socket
using the method that supports `\n` and binary data.
```go
package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net"
)

func main() {
	jd, err := newJdWriter("experiment")
	if err != nil {
		log.Fatal(err)
	}
	defer jd.Close()

	messges := [][]byte{
		{0, 2, 4, 8, 10, 12, 14, 16, 18},
		{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100},
		[]byte(`FOO\nBAR\nFOO`),
	}

	for _, msg := range messges {
		written, err := jd.Write(msg)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%d bytes written to Journald socket\n", written)
	}
}

type jdWriter struct {
	id   string
	conn net.Conn
}

func newJdWriter(id string) (jdWriter, error) {
	conn, err := net.Dial("unixgram", "/run/systemd/journal/socket")
	if err != nil {
		return jdWriter{}, fmt.Errorf("cannot open unix socket: %w", err)
	}

	jd := jdWriter{
		id:   id,
		conn: conn,
	}

	return jd, nil
}

func (j jdWriter) Write(msg []byte) (int, error) {
	w := &bytes.Buffer{}

	fmt.Fprintf(w, "SYSLOG_IDENTIFIER=%s\n", j.id)
	w.WriteString("MESSAGE")
	w.WriteString("\n")
	l := len(msg)
	if err := binary.Write(w, binary.LittleEndian, uint64(l)); err != nil {
		log.Fatal(err)
	}

	w.Write(msg)
	w.WriteString("\n")

	return j.conn.Write(w.Bytes())
}

func (j jdWriter) Close() error {
	return j.conn.Close()
}
```

## Crafting a journal file
The easiest way to craft a journal file with the entries you need is
to use
[`systemd-journald-remote`](https://www.freedesktop.org/software/systemd/man/latest/systemd-journal-remote.service.html).
First we need to export some entries to a file:
```
root@vagrant-debian-12:~/filebeat# journalctl -g "Hello" -o export >export
```
One good thing of the `-o export` is that you can just concatenate the
output of any number of runs and the result will be a valid file.

Then you can use `systemd-journald-remote` to generate the journal
file:
```
root@vagrant-debian-12:~/filebeat# /usr/lib/systemd/systemd-journal-remote -o example.journal export
Finishing after writing 2 entries
``
Or you can run as a one liner:
```
root@vagrant-debian-12:~/filebeat# journalctl -g "Hello" -o export | /usr/lib/systemd/systemd-journal-remote -o example.journal -
```

Then you can read the newly created file:
```
root@vagrant-debian-12:~/filebeat# journalctl --file ./example.journal
Oct 02 04:16:54 vagrant-debian-12 unknown[1908]: Hello Journal!
Oct 02 04:17:50 vagrant-debian-12 my-test[1924]: Hello Journal!
root@vagrant-debian-12:~/filebeat# 
```

Bear in mind that `systemd-journal-remote` will **append** to the
output file.

## References
- https://systemd.io/JOURNAL_NATIVE_PROTOCOL/
- https://www.freedesktop.org/software/systemd/man/latest/journalctl.html
- https://www.freedesktop.org/software/systemd/man/latest/systemd-cat.html
- https://www.freedesktop.org/software/systemd/man/latest/systemd-journal-remote.service.html
