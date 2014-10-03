package main

import (
	"bytes"
	"fmt"
	"labix.org/v2/mgo/bson"
)

// bare-bone Error type, to make it easy to create
// our own exceptions with a string.
type GenericError struct {
	msg string
}

func (err GenericError) Error() string {
	return err.msg
}

// Convenience function for quickly returning GenericErrors
func MsgError(format string, v ...interface{}) error {
	return GenericError{
		msg: fmt.Sprintf(format, v...),
	}
}

// Byte order utilities
func Bytes_Ntohs(b []byte) uint16 {
	return uint16(b[0])<<8 | uint16(b[1])
}

func Bytes_Ntohl(b []byte) uint32 {
	return uint32(b[0])<<24 | uint32(b[1])<<16 |
		uint32(b[2])<<8 | uint32(b[3])
}

func Bytes_Htohl(b []byte) uint32 {
	return uint32(b[3])<<24 | uint32(b[2])<<16 |
		uint32(b[1])<<8 | uint32(b[0])
}

func Bytes_Ntohll(b []byte) uint64 {
	return uint64(b[0])<<56 | uint64(b[1])<<48 |
		uint64(b[2])<<40 | uint64(b[3])<<32 |
		uint64(b[4])<<24 | uint64(b[5])<<16 |
		uint64(b[6])<<8 | uint64(b[7])
}

func bson_concat(dict1 bson.M, dict2 bson.M) bson.M {
	dict := bson.M{}

	for k, v := range dict1 {
		dict[k] = v
	}

	for k, v := range dict2 {
		dict[k] = v
	}
	return dict
}

func Ipv4_Ntoa(ip uint32) string {
	return fmt.Sprintf("%d.%d.%d.%d",
		byte(ip>>24), byte(ip>>16),
		byte(ip>>8), byte(ip))
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func readString(s []byte) (string, error) {
	i := bytes.IndexByte(s, 0)
	if i < 0 {
		return "", MsgError("No string found")
	}
	res := string(s[:i])
	return res, nil
}
