package v2

type responseConfig struct {
	Transforms transformsConfig `config:"transforms"`
	Pagination transformsConfig `config:"pagination"`
}

func (c *responseConfig) Validate() error {
	if _, err := newResponseTransformsFromConfig(c.Transforms); err != nil {
		return err
	}
	if _, err := newPaginationTransformsFromConfig(c.Transforms); err != nil {
		return err
	}

	return nil
}
