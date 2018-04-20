package asset

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"io/ioutil"
)

// FieldsRegistry contains a list of fields.yml files
// As each entry is an array of bytes multiple fields.yml can be added under one path.
// This can become useful as we don't have to generate anymore the fields.yml but can
// package each local fields.yml from things like processors.
var FieldsRegistry = map[string][]string{}

// SetFields sets the fields for a given path
func SetFields(path string, asset func() string) error {
	data := asset()

	FieldsRegistry[path] = append(FieldsRegistry[path], data)

	return nil
}

// GetFields returns a byte array contains all fields for the given path
func GetFields(path string) ([]byte, error) {
	var fields []byte
	for _, data := range FieldsRegistry[path] {

		output, err := DecodeData(data)
		if err != nil {
			return nil, err
		}

		fields = append(fields, output...)
	}
	return fields, nil
}

// EncodeData compresses the data with zlib and base64 encodes it
func EncodeData(data string) (string, error) {
	var zlibBuf bytes.Buffer
	writer := zlib.NewWriter(&zlibBuf)
	_, err := writer.Write([]byte(data))
	if err != nil {
		return "", err
	}
	err = writer.Close()
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(zlibBuf.Bytes()), nil
}

// DecodeData base64 decodes the data and uncompresses it
func DecodeData(data string) ([]byte, error) {

	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}

	b := bytes.NewReader(decoded)
	r, err := zlib.NewReader(b)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	return ioutil.ReadAll(r)
}
