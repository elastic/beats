package v1

import "github.com/ericchiang/k8s"

func init() {
	k8s.Register("authentication.k8s.io", "v1", "tokenreviews", false, &TokenReview{})
}
