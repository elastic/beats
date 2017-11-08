package keystore

// SecureString Initial implementation for a SecureString representation in
// beats, currently we keep the password into a Bytes array, we need to implement a way
// to safely clean that array.
//
// Investigate memguard: https://github.com/awnumar/memguard
type SecureString struct {
	value []byte
}

// NewSecureString return a struct representing a secrets string.
func NewSecureString(value []byte) *SecureString {
	return &SecureString{
		value: value,
	}
}

// Get returns the byte value of the secret, or an error if we cannot return it.
func (s *SecureString) Get() ([]byte, error) {
	return s.value, nil
}

// String custom string implementation to make sure we don't bleed this struct into a string.
func (s SecureString) String() string {
	return "<SecureString>"
}

// GoString implements the GoStringer interface to hide the secret value.
func (s SecureString) GoString() string {
	return s.String()
}
