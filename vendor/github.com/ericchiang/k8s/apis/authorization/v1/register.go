package v1

import "github.com/ericchiang/k8s"

func init() {
	k8s.Register("authorization.k8s.io", "v1", "localsubjectaccessreviews", true, &LocalSubjectAccessReview{})
	k8s.Register("authorization.k8s.io", "v1", "selfsubjectaccessreviews", false, &SelfSubjectAccessReview{})
	k8s.Register("authorization.k8s.io", "v1", "selfsubjectrulesreviews", false, &SelfSubjectRulesReview{})
	k8s.Register("authorization.k8s.io", "v1", "subjectaccessreviews", false, &SubjectAccessReview{})
}
