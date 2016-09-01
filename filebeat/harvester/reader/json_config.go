package reader

type JSONConfig struct {
	MessageKey    string `config:"message_key"`
	KeysUnderRoot bool   `config:"keys_under_root"`
	OverwriteKeys bool   `config:"overwrite_keys"`
	AddErrorKey   bool   `config:"add_error_key"`
	NestedParsing bool   `config:"nested_parsing"`
}

func (c *JSONConfig) Validate() error {
	return nil
}
