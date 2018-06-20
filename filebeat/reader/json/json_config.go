package json

// Config holds the options a JSON reader.
type Config struct {
	MessageKey          string `config:"message_key"`
	KeysUnderRoot       bool   `config:"keys_under_root"`
	OverwriteKeys       bool   `config:"overwrite_keys"`
	AddErrorKey         bool   `config:"add_error_key"`
	IgnoreDecodingError bool   `config:"ignore_decoding_error"`
}

// Validate validates the Config option for JSON reader.
func (c *Config) Validate() error {
	return nil
}
