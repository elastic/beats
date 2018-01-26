package v1beta1

import "github.com/ericchiang/k8s"

func init() {
	k8s.Register("authorization.k8s.io", "v1beta1", "localsubjectaccessreviews", true, &LocalSubjectAccessReview{})
	k8s.Register("authorization.k8s.io", "v1beta1", "selfsubjectaccessreviews", false, &SelfSubjectAccessReview{})
	k8s.Register("authorization.k8s.io", "v1beta1", "selfsubjectrulesreviews", false, &SelfSubjectRulesReview{})
	k8s.Register("authorization.k8s.io", "v1beta1", "subjectaccessreviews", false, &SubjectAccessReview{})
}
