package algorithm

func init() {
	Register(&tokenBucket{})
}

type tokenBucket struct {
	// TODO: flesh out
}

func (t *tokenBucket) ID() string {
	return "token_bucket"
}

func (t *tokenBucket) Configure(config Config) error {
	// TODO: flesh out
	return nil
}

func (t *tokenBucket) IsAllowed(key string) bool {
	// TODO: flesh out
	return true
}
