package rules

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"
	"io"
	"reflect"
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/filter"
	"github.com/pkg/errors"
)

type Fingerprint struct {
	cond        *filter.Condition
	hashName    string           // Name of the hash function.
	hashFactory func() hash.Hash // Function to create a new Hash.
	fields      []string         // List of input fields.
	target      string           // Target field where hash is written.
}

type FingerprintConfig struct {
	filter.ConditionConfig `config:",inline"`

	// Hash function used for calculating the fingerprint. The accepted values
	// are sha1, sha256, sha512, and md5. This value is case-insensitive. The
	// default is sha1.
	Hash string `config:"hash"`

	// Fields is a list of fields whose values are to be used as the input to
	// the hash function. The field values are concatenated before the hashing
	// is performed. All fields must be present in the event otherwise an error
	// will be returned by the filter. The default is message.
	Fields []string `config:"fields"`

	// Target field for the hash value. The value is hex encoded. The default
	// is id.
	Target string `config:"target"`
}

var defaultFingerprintConfig = FingerprintConfig{
	Hash:   "sha1",
	Fields: []string{"message"},
	Target: "id",
}

func init() {
	if err := filter.RegisterPlugin("fingerprint", newFingerprint); err != nil {
		panic(err)
	}
}

func newFingerprint(c common.Config) (filter.FilterRule, error) {
	fc := defaultFingerprintConfig
	err := c.Unpack(&fc)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unpack fingerprint config")
	}

	conditions, err := filter.NewCondition(fc.ConditionConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create new condition")
	}

	hashName := strings.ToLower(fc.Hash)
	var hashFactory func() hash.Hash
	switch hashName {
	case "sha1":
		hashFactory = sha1.New
	case "sha256":
		hashFactory = sha256.New
	case "sha512":
		hashFactory = sha512.New
	case "md5":
		hashFactory = md5.New
	default:
		return nil, fmt.Errorf("unknown fingerprint hash function: %v", fc.Hash)
	}

	return &Fingerprint{
		cond:        conditions,
		hashName:    hashName,
		hashFactory: hashFactory,
		fields:      fc.Fields,
		target:      fc.Target,
	}, nil
}

func (f *Fingerprint) Filter(event common.MapStr) (common.MapStr, error) {
	if f.cond != nil && !f.cond.Check(event) {
		return event, nil
	}

	h := f.hashFactory()
	for _, field := range f.fields {
		v, err := event.GetValue(field)
		if err != nil {
			return event, err
		}
		writeValue(h, v)
	}

	// Compute the hash and encode the value in base64.
	hashBytes := h.Sum(nil)
	event[f.target] = fmt.Sprintf("%x", hashBytes)

	return event, nil
}

func (f Fingerprint) String() string {
	b := new(bytes.Buffer)
	b.WriteString("fingerprint=[")

	b.WriteString("fields=")
	b.WriteString(strings.Join(f.fields, ", "))

	b.WriteString(", hash=")
	b.WriteString(f.hashName)

	b.WriteString(", target=")
	b.WriteString(f.target)

	if f.cond != nil {
		b.WriteString(", condition=")
		b.WriteString(f.cond.String())
	}

	b.WriteRune(']')

	return b.String()
}

func writeValue(writer io.Writer, object interface{}) {
	val := reflect.ValueOf(object)

	// Follow the pointer.
	if val.Kind() == reflect.Ptr && !val.IsNil() {
		val = reflect.ValueOf(val.Elem().Interface())
	}

	if val.IsValid() {
		writer.Write([]byte(fmt.Sprintf("%v", val.Interface())))
	} else {
		writer.Write([]byte("nil"))
	}
}
