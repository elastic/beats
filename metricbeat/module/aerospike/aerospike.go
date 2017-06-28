package aerospike

import (
	"strconv"
	"strings"

	"github.com/pkg/errors"

	as "github.com/aerospike/aerospike-client-go"
)

func ParseHost(host string) (*as.Host, error) {
	pieces := strings.Split(host, ":")
	if len(pieces) != 2 {
		return nil, errors.Errorf("Can't parse host %s", host)
	}
	port, err := strconv.Atoi(pieces[1])
	if err != nil {
		return nil, errors.Wrapf(err, "Can't parse port")
	}
	return as.NewHost(pieces[0], port), nil
}

func ParseInfo(info string) map[string]interface{} {
	result := make(map[string]interface{})

	for _, keyValueStr := range strings.Split(info, ";") {
		KeyValArr := strings.Split(keyValueStr, "=")
		if len(KeyValArr) == 2 {
			result[KeyValArr[0]] = KeyValArr[1]
		}
	}

	return result
}
