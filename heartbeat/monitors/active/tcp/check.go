package tcp

import (
	"bytes"
	"errors"
	"net"
)

type ConnCheck func(net.Conn) error

var (
	errNoDataReceived = errors.New("no data")
	errRecvMismatch   = errors.New("received string mismatch")
)

func (c ConnCheck) Validate(conn net.Conn) error {
	return c(conn)
}

func makeValidateConn(config *Config) ConnCheck {
	send := config.SendString
	recv := config.ReceiveString

	switch {
	case send == "" && recv == "":
		return nil
	case send != "" && recv == "":
		return checkAll(checkSend([]byte(send)), checkRecvAny)
	case send == "" && recv != "":
		return checkRecv([]byte(recv))
	default: // send != "" && recv != "":
		return checkAll(checkSend([]byte(send)), checkRecv([]byte(recv)))
	}
}

func checkOk(_ net.Conn) error { return nil }

func checkAll(checks ...ConnCheck) ConnCheck {
	return func(conn net.Conn) error {
		for _, check := range checks {
			if err := check(conn); err != nil {
				return err
			}
		}
		return nil
	}
}

func checkSend(buf []byte) ConnCheck {
	return func(conn net.Conn) error {
		return sendBuffer(conn, buf)
	}
}

func checkRecv(expected []byte) ConnCheck {
	return func(conn net.Conn) error {
		buf := make([]byte, len(expected))
		if err := recvBuffer(conn, buf); err != nil {
			return err
		}
		if !bytes.Equal(expected, buf) {
			// TODO: report received value and expected value in event
			return errRecvMismatch
		}
		return nil
	}
}

func checkRecvAny(conn net.Conn) error {
	// receive 'anything'
	var buf [1024]byte
	n, err := conn.Read(buf[:])
	if err != nil {
		return err
	}
	if n == 0 {
		return errNoDataReceived
	}
	return nil
}

func sendBuffer(conn net.Conn, buf []byte) error {
	for len(buf) > 0 {
		n, err := conn.Write(buf)
		if err != nil {
			return err
		}
		buf = buf[n:]
	}
	return nil
}

func recvBuffer(conn net.Conn, buf []byte) error {
	for len(buf) > 0 {
		n, err := conn.Read(buf)
		if err != nil {
			return err
		}
		buf = buf[n:]
	}
	return nil
}
