package harvester

import "github.com/elastic/beats/libbeat/common/match"

// Contains available input types
const (
	LogType    = "log"
	StdinType  = "stdin"
	RedisType  = "redis"
	UdpType    = "udp"
	DockerType = "docker"
)

// MatchAny checks if the text matches any of the regular expressions
func MatchAny(matchers []match.Matcher, text string) bool {
	for _, m := range matchers {
		if m.MatchString(text) {
			return true
		}
	}
	return false
}
