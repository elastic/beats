package util

import (
	"regexp"
	"strings"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// ExtractWorkloadName 根据 Pod 名称提取出非随机的前缀部分，即工作负载名称
func ExtractWorkloadName(podName string) string {
	// 匹配 Deployment/ReplicaSet 生成的 Pod 名称
	// 格式：<工作负载名称>-<副本集哈希>-<Pod 随机后缀>
	deployPattern := regexp.MustCompile(`^(.*)-[a-z0-9]{8,10}-[a-z0-9]{5}$`)
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

func EnrichWorkloadInfo(fields mapstr.M, podNameKey string, event mb.Event) {
	workloadName := ""

	podNameValue, _ := fields.GetValue(podNameKey)
	if podName, ok := podNameValue.(string); ok {
		workloadName = ExtractWorkloadNameWithEvent(podName, event)
	}

	event.ModuleFields.DeepUpdate(mapstr.M{
		"workload": mapstr.M{
			"name": workloadName,
		},
	})
}

func DuplicateWorkloadInfo(fields mapstr.M, workloadNameKey string, event mb.Event) {
	workloadNameValue, _ := fields.GetValue(workloadNameKey)

	workloadName, _ := workloadNameValue.(string)

	event.ModuleFields.DeepUpdate(mapstr.M{
		"workload": mapstr.M{
			"name": workloadName,
		},
	})
}

// ExtractWorkloadNameWithEvent 基于 event.ModuleFields 中的工作负载类型优先选择对应正则；
// 如未识别到明确类型则回退到原有的通用提取逻辑。
func ExtractWorkloadNameWithEvent(podName string, event mb.Event) string {
	if kind, ok := detectWorkloadKind(event.ModuleFields); ok {
		switch kind {
		case "deployment", "replicaset":
			if m := regexp.MustCompile(`^(.*)-[a-z0-9]{8,10}-[a-z0-9]{5}$`).FindStringSubmatch(podName); m != nil {
				return m[1]
			}
		case "statefulset":
			if m := regexp.MustCompile(`^(.*)-\d+$`).FindStringSubmatch(podName); m != nil {
				return m[1]
			}
		case "daemonset", "job", "cronjob":
			if m := regexp.MustCompile(`^(.*)-[a-z0-9]{5,}$`).FindStringSubmatch(podName); m != nil {
				return m[1]
			}
		}
		// 若按类型的正则未匹配，则回退到通用逻辑
		return ExtractWorkloadName(podName)
	}
	// 未能识别出类型，使用通用逻辑
	return ExtractWorkloadName(podName)
}

// detectWorkloadKind 扫描 map，判断是否包含目标工作负载类型字段名
// 返回 (类型小写, 是否找到)
func detectWorkloadKind(m mapstr.M) (string, bool) {
	if m == nil {
		return "", false
	}
	// 扁平检查当前层的键名
	for k := range m {
		lk := strings.ToLower(k)
		switch lk {
		case "deployment", "replicaset", "daemonset", "job", "cronjob", "statefulset":
			return lk, true
		}
	}
	return "", false
}
