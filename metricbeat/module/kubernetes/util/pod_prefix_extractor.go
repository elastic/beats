package util

import (
	"regexp"
)

// ExtractWorkloadName 根据 Pod 名称提取出非随机的前缀部分，即工作负载名称
func ExtractWorkloadName(podName string) string {
	// 匹配 Deployment/ReplicaSet 生成的 Pod 名称
	// 格式：<工作负载名称>-<副本集哈希>-<Pod 随机后缀>
	deployPattern := regexp.MustCompile(`^(.*)-[a-z0-9]{9,10}-[a-z0-9]{5}$`)
	if m := deployPattern.FindStringSubmatch(podName); m != nil {
		return m[1]
	}

	// 匹配 StatefulSet 生成的 Pod 名称
	// 格式：<工作负载名称>-<序号>
	statefulPattern := regexp.MustCompile(`^(.*)-\d+$`)
	if m := statefulPattern.FindStringSubmatch(podName); m != nil {
		return m[1]
	}

	// 匹配 Job 或 DaemonSet 生成的 Pod 名称（只追加了一个随机后缀）
	// 格式：<工作负载名称>-<随机后缀>
	simplePattern := regexp.MustCompile(`^(.*)-[a-z0-9]{5,}$`)
	if m := simplePattern.FindStringSubmatch(podName); m != nil {
		return m[1]
	}

	// 如果都没有匹配到，则返回原始名称
	return podName
}
