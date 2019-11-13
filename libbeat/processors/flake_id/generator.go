package flake_id

import (
	"crypto/rand"
	"net"
)

// Golang port of https://github.com/elastic/elasticsearch/commit/9c1ac95ba8e593c90b4681f2a554b12ff677cf89

type id []byte

const MAC_ADDR_LEN = 6

func (i *id) Base64() string {
	return "TODO"
}

func getSecureMungedAddress() ([]byte, error) {
	addr, err := getMacAddress()
	if err != nil {
		return nil, err
	}

	if !isValidAddress(addr) {
		addr = constructDummyMulticastAddress()
	}

	munged := make([]byte, MAC_ADDR_LEN)
	_, err = rand.Read(munged)
	if err != nil {
		return nil, err
	}

	for i := 0; i < MAC_ADDR_LEN; i++ {
		munged[i] ^= addr[i]
	}

	return munged, nil
}

func getMacAddress() ([]byte, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err

	}
	for _, i := range interfaces {
		if i.Flags != net.FlagLoopback {
			// Pick the first valid non loopback address we find
			addr := i.HardwareAddr
			if isValidAddress(addr) {
				return addr, nil
			}
		}
	}

	// Could not find a mac address
	return nil, nil
}

func isValidAddress(addr []byte) bool {
	if addr == nil || len(addr) != 6 {
		return false
	}

	for _, b := range addr {
		if b != 0x00 {
			return true // If any of the bytes are non zero assume a good address
		}
	}
}

func constructDummyMulticastAddress() []byte {

}
