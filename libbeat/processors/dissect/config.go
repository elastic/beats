package dissect

type config struct {
	Tokenizer    *tokenizer `config:"tokenizer"`
	Field        string     `config:"field"`
	TargetPrefix string     `config:"target_prefix"`
}

var defaultConfig = config{
	Field:        "message",
	TargetPrefix: "dissect",
}

// tokenizer add validation at the unpack level for this specific field.
type tokenizer = Dissector

// Unpack a tokenizer into a dissector this will trigger the normal validation of the dissector.
func (t *tokenizer) Unpack(v string) error {
	d, err := New(v)
	if err != nil {
		return err
	}
	*t = *d
	return nil
}
