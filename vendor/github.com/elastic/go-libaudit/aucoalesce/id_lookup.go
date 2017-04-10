package aucoalesce

import (
	"math"
	"os/user"
	"strings"
	"time"
)

const cacheTimeout = 0

var (
	userLookup  = NewUserCache()
	groupLookup = NewGroupCache()
)

type stringItem struct {
	timeout time.Time
	value   string
}

func (i *stringItem) isExpired() bool {
	return time.Now().After(i.timeout)
}

// UserCache is a cache of UID to username.
type UserCache map[string]stringItem

// NewUserCache returns a new UserCache.
func NewUserCache() UserCache {
	return map[string]stringItem{
		"0": {timeout: time.Unix(math.MaxInt64, 0), value: "root"},
	}
}

// LookupUID looks up a UID and returns the username associated with it. If
// no username could be found an empty string is returned. The value will be
// cached for a minute. This requires cgo on Linux.
func (c UserCache) LookupUID(uid string) string {
	if uid == "" || uid == "unset" {
		return ""
	}

	if item, found := c[uid]; found && !item.isExpired() {
		return item.value
	}

	// Cache the value (even on error).
	user, err := user.LookupId(uid)
	if err != nil {
		c[uid] = stringItem{timeout: time.Now().Add(cacheTimeout), value: ""}
		return ""
	}

	c[uid] = stringItem{timeout: time.Now().Add(cacheTimeout), value: user.Username}
	return user.Username
}

// GroupCache is a cache of GID to group name.
type GroupCache map[string]stringItem

// NewGroupCache returns a new GroupCache.
func NewGroupCache() GroupCache {
	return map[string]stringItem{
		"0": {timeout: time.Unix(math.MaxInt64, 0), value: "root"},
	}
}

// LookupGID looks up a GID and returns the group associated with it. If
// no group could be found an empty string is returned. The value will be
// cached for a minute. This requires cgo on Linux.
func (c GroupCache) LookupGID(gid string) string {
	if gid == "" || gid == "unset" {
		return ""
	}

	if item, found := c[gid]; found && !item.isExpired() {
		return item.value
	}

	// Cache the value (even on error).
	group, err := user.LookupGroupId(gid)
	if err != nil {
		c[gid] = stringItem{timeout: time.Now().Add(cacheTimeout), value: ""}
		return ""
	}

	c[gid] = stringItem{timeout: time.Now().Add(cacheTimeout), value: group.Name}
	return group.Name
}

// ResolveIDs translates all uid and gid values to their associated names.
// This requires cgo on Linux.
func ResolveIDs(event *Event) {
	if v := userLookup.LookupUID(event.Subject.Primary); v != "" {
		event.Subject.Primary = v
	}
	if v := userLookup.LookupUID(event.Subject.Secondary); v != "" {
		event.Subject.Secondary = v
	}

	processMap := func(m map[string]string) {
		for key, id := range m {
			if strings.HasSuffix(key, "uid") {
				if v := userLookup.LookupUID(id); v != "" {
					m[key] = v
				}
			} else if strings.HasSuffix(key, "gid") {
				if v := groupLookup.LookupGID(id); v != "" {
					m[key] = v
				}
			}
		}
	}
	processMap(event.Subject.Attributes)
	for _, path := range event.Paths {
		processMap(path)
	}
}
