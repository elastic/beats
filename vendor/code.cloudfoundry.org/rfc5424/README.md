
[![Build Status](https://travis-ci.org/crewjam/rfc5424.png)](https://travis-ci.org/crewjam/rfc5424)

[![](https://godoc.org/github.com/crewjam/rfc5424?status.png)](http://godoc.org/github.com/crewjam/rfc5424)

This is a Go library that can read and write RFC-5424 syslog messages:

Example usage:

    m := rfc5424.Message{
        Priority:  rfc5424.Daemon | rfc5424.Info,
        Timestamp: time.Now(),
        Hostname:  "myhostname",
        AppName:   "someapp",
        Message:   []byte("Hello, World!"),
    }
    m.AddDatum("foo@1234", "Revision", "1.2.3.4")
    m.WriteTo(os.Stdout)

Produces output like:

    107 <7>1 2016-02-28T09:57:10.804642398-05:00 myhostname someapp - - [foo@1234 Revision="1.2.3.4"] Hello, World!

You can also use the library to parse syslog messages:

    m := rfc5424.Message{}
    _, err := m.ReadFrom(os.Stdin)
    fmt.Printf("%s\n", m.Message)
