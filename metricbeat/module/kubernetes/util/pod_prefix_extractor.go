package util

import (
	"reflect"
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

// ExtractWorkloadNameWithEvent 基于 event.ModuleFields 和 event.MetricSetFields 中的工作负载类型优先选择对应正则；
// 如果从 event 中直接找到工作负载名称，则直接使用；如未识别到明确类型则回退到原有的通用提取逻辑。
func ExtractWorkloadNameWithEvent(podName string, event mb.Event) string {
	// 尝试从 event 中直接获取工作负载名称
	if workloadName, kind, found := detectWorkloadKind(event); found {
		// 如果找到了工作负载名称，直接使用
		if workloadName != "" {
			return workloadName
		}
		// 如果找到了工作负载类型但没有名称，使用正则表达式提取
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

// detectWorkloadKind 分别从 ModuleFields 和 MetricSetFields 中嵌套检测是否存在工作负载类型键名
// 如果找到这些键，检查对应 value 是否存在 name 字段，如果有，直接使用 name 的值
// 返回 (工作负载名称, 工作负载类型小写, 是否找到)
func detectWorkloadKind(event mb.Event) (string, string, bool) {
	// 检查 ModuleFields
	if workloadName, kind, found := detectWorkloadKindInMap(event.ModuleFields); found {
		return workloadName, kind, true
	}
	// 检查 MetricSetFields
	if workloadName, kind, found := detectWorkloadKindInMap(event.MetricSetFields); found {
		return workloadName, kind, true
	}
	return "", "", false
}

const (
	// maxRecursionDepth 最大递归深度，防止栈溢出
	maxRecursionDepth = 100
)

// detectWorkloadKindInMap 在给定的 mapstr.M 中递归嵌套检测是否存在工作负载类型键名
// 返回 (工作负载名称, 工作负载类型小写, 是否找到)
func detectWorkloadKindInMap(m mapstr.M) (string, string, bool) {
	if m == nil {
		return "", "", false
	}
	// 用于跟踪已访问的 map，防止循环引用
	// 使用 map 的指针地址作为唯一标识
	visited := make(map[uintptr]bool)
	// 递归查找函数
	var findWorkload func(mapstr.M, int) (string, string, bool)
	findWorkload = func(current mapstr.M, depth int) (string, string, bool) {
		if current == nil || depth > maxRecursionDepth {
			return "", "", false
		}
		// 检查循环引用：使用反射获取 map 的指针地址
		currentPtr := reflect.ValueOf(current).Pointer()
		if visited[currentPtr] {
			return "", "", false
		}
		visited[currentPtr] = true
		defer delete(visited, currentPtr)

		// 检查当前层的键名
		for k := range current {
			lk := strings.ToLower(k)
			// 获取对应键的值（只调用一次）
			value, _ := current.GetValue(k)
			if value == nil {
				continue
			}

			switch lk {
			case "deployment", "replicaset", "daemonset", "job", "cronjob", "statefulset":
				// 检查值是否为 map，如果是，检查是否有 name 字段
				if workloadMap := toMapStr(value); workloadMap != nil {
					if nameValue, _ := workloadMap.GetValue("name"); nameValue != nil {
						if name, ok := nameValue.(string); ok && name != "" {
							return name, lk, true
						}
					}
				}
				// 即使没有找到 name 字段，也返回找到的工作负载类型
				return "", lk, true
			default:
				// 如果当前键的值是 map，递归查找
				if subMap := toMapStr(value); subMap != nil {
					if workloadName, kind, found := findWorkload(subMap, depth+1); found {
						return workloadName, kind, true
					}
				}
			}
		}
		return "", "", false
	}
	return findWorkload(m, 0)
}

// toMapStr 将 interface{} 转换为 mapstr.M，支持 mapstr.M 和 map[string]interface{} 类型
func toMapStr(v interface{}) mapstr.M {
	if v == nil {
		return nil
	}
	// 尝试直接转换为 mapstr.M
	if m, ok := v.(mapstr.M); ok {
		return m
	}
	// 尝试转换为 map[string]interface{}
	if m, ok := v.(map[string]interface{}); ok {
		return mapstr.M(m)
	}
	return nil
}
