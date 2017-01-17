package kafka

import "strings"

type APIVersion uint16

type APIKey uint16

const (
	requestMinHeaderSize  = 8 + 2
	responseMinHeaderSize = 4
)

const (
	V0 APIVersion = 0
	V1 APIVersion = 1
	V2 APIVersion = 2
)

const (
	APIProduce          APIKey = 0
	APIFetch            APIKey = 1
	APIOffset           APIKey = 2
	APIMetadata         APIKey = 3
	APIInternal4        APIKey = 4
	APIInternal5        APIKey = 5
	APIInternal6        APIKey = 6
	APIInternal7        APIKey = 7
	APIOffsetCommit     APIKey = 8
	APIOffsetFetch      APIKey = 9
	APIGroupCoordinator APIKey = 10
	APIJoinGroup        APIKey = 11
	APIHeartbeat        APIKey = 12
	APILeaveGroup       APIKey = 13
	APISyncGroup        APIKey = 14
	APIDescribeGroups   APIKey = 15
	APIListGroups       APIKey = 16

	APIKeyInvalid APIKey = 0xffff
)

const APITypes int = 17

var apiNames = map[APIKey]string{
	APIProduce:          "Produce",
	APIFetch:            "Fetch",
	APIOffset:           "Offset",
	APIMetadata:         "Metadata",
	APIInternal4:        "Internal4",
	APIInternal5:        "Internal5",
	APIInternal6:        "Internal6",
	APIInternal7:        "Internal7",
	APIOffsetCommit:     "OffsetCommit",
	APIOffsetFetch:      "OffsetFetch",
	APIGroupCoordinator: "GroupCoordinator",
	APIJoinGroup:        "JoinGroup",
	APIHeartbeat:        "Heartbeat",
	APILeaveGroup:       "LeaveGroup",
	APISyncGroup:        "SyncGroup",
	APIDescribeGroups:   "DescribeGroups",
	APIListGroups:       "ListGroups",
}

var versionNames = map[APIVersion]string{
	V0: "Version 0",
	V1: "Version 1",
	V2: "Version 2",
}

func APIKeyByName(name string) APIKey {
	n := strings.ToLower(name)
	for key, other := range apiNames {
		if strings.ToLower(other) == n {
			return key
		}
	}
	return APIKeyInvalid
}

func (k APIKey) Valid() bool {
	_, exists := apiNames[k]
	return exists
}

func (k APIKey) String() string {
	if name, exists := apiNames[k]; exists {
		return name
	}
	return "Unknown"
}

func (v APIVersion) Supported() bool {
	_, exists := versionNames[v]
	return exists
}

func (v APIVersion) String() string {
	if name, exists := versionNames[v]; exists {
		return name
	}
	return "Unknown"
}
