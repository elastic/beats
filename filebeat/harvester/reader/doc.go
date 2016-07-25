/*
Package reader provides interface and struct to read messages and report them to a harvester

The interface used is:

	type Reader interface {
		Next() (Message, error)
	}

Each time Next is called on a reader, a Message object is returned.

*/
package reader
