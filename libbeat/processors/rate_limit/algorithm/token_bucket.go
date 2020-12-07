package algorithm

func init() {
	Register("token_bucket", newTokenBucket)
}

type tokenBucket struct {
	// TODO: flesh out
}

func newTokenBucket(config Config) Algorithm {
	return &tokenBucket{}
}

func (t *tokenBucket) IsAllowed(key string) bool {
	// TODO: flesh out
	return true
}
