package file_integrity

import (
	"math/bits"
	"strings"
)

// Action is a description of the changes described by an event.
type Action uint8

// ActionArray is just syntactic sugar to invoke methods on []Action receiver
type ActionArray []Action

// List of possible Actions.
const (
	None               Action = 0
	AttributesModified        = 1 << (iota - 1)
	Created
	Deleted
	Updated
	Moved
	ConfigChange
)

var actionNames = map[Action]string{
	None:               "none",
	AttributesModified: "attributes_modified",
	Created:            "created",
	Deleted:            "deleted",
	Updated:            "updated",
	Moved:              "moved",
	ConfigChange:       "config_change",
}

type actionOrderKey struct {
	ExistsBefore, ExistsNow bool
	Action                  Action
}

// Given the previous and current state of the file, and an action mask
// returns a meaningful ordering for the actions in the mask
var actionOrderMap = map[actionOrderKey]ActionArray{
	{false, false, Created | Deleted}:                  {Created, Deleted},
	{true, true, Created | Deleted}:                    {Deleted, Created},
	{false, false, Moved | Created}:                    {Created, Moved},
	{true, true, Moved | Created}:                      {Moved, Created},
	{true, true, Moved | Deleted}:                      {Deleted, Moved},
	{false, false, Moved | Deleted}:                    {Moved, Deleted},
	{false, true, Updated | Created}:                   {Created, Updated},
	{true, false, Updated | Deleted}:                   {Updated, Deleted},
	{false, true, Updated | Moved}:                     {Moved, Updated},
	{true, false, Updated | Moved}:                     {Updated, Moved},
	{false, true, Moved | Created | Deleted}:           {Created, Deleted, Moved},
	{true, false, Moved | Created | Deleted}:           {Deleted, Created, Moved},
	{false, false, Updated | Moved | Created}:          {Created, Updated, Moved},
	{true, true, Updated | Moved | Created}:            {Moved, Created, Updated},
	{false, false, Updated | Moved | Deleted}:          {Moved, Updated, Deleted},
	{true, true, Updated | Moved | Deleted}:            {Deleted, Moved, Updated},
	{false, false, Updated | Created | Deleted}:        {Created, Updated, Deleted},
	{true, true, Updated | Created | Deleted}:          {Deleted, Created, Updated},
	{false, true, Updated | Moved | Created | Deleted}: {Created, Deleted, Moved, Updated},
	{true, false, Updated | Moved | Created | Deleted}: {Deleted, Created, Updated, Moved},
}

func (action Action) isMultiple() bool {
	return bits.OnesCount8(uint8(action)) > 1
}

func (action Action) String() string {
	if name, found := actionNames[action]; found {
		return name
	}
	var list []string
	for flag, name := range actionNames {
		if action&flag != 0 {
			action ^= flag
			list = append(list, name)
		}
	}
	if action != 0 {
		list = append(list, "unknown")
	}
	return strings.Join(list, "|")
}

// MarshalText marshals the Action to a textual representation of itself.
func (action Action) MarshalText() ([]byte, error) { return []byte(action.String()), nil }

func resolveActionOrder(action Action, existedBefore, existsNow bool) ActionArray {
	if action == None {
		return nil
	}
	if !action.isMultiple() {
		return []Action{action}
	}
	key := actionOrderKey{existedBefore, existsNow, action}
	if result, ok := actionOrderMap[key]; ok {
		return result
	}

	// Can't resolve a meaningful order for the actions, usually the file
	// has received further actions after the event being processed
	return action.InAnyOrder()
}

func (action Action) InOrder(existedBefore, existsNow bool) ActionArray {
	hasConfigChange := 0 != action&ConfigChange
	hasUpdate := 0 != action&Updated
	hasAttrMod := 0 != action&AttributesModified
	action = Action(int(action) & int(^(ConfigChange | AttributesModified)))
	if hasAttrMod {
		action |= Updated
	}

	result := resolveActionOrder(action, existedBefore, existsNow)

	if hasConfigChange {
		result = append(result, ConfigChange)
	}

	if hasAttrMod {
		for idx, value := range result {
			if value == Updated {
				if !hasUpdate {
					result[idx] = AttributesModified
				} else {
					result = append(result, None)
					copy(result[idx+2:], result[idx+1:])
					result[idx+1] = AttributesModified
				}
				break
			}
		}
	}
	return result
}

func (action Action) InAnyOrder() ActionArray {
	if !action.isMultiple() {
		return []Action{action}
	}
	var result []Action
	for k := range actionNames {
		if 0 != action&k {
			result = append(result, k)
		}
	}
	return result
}

func (actions ActionArray) StringArray() []string {
	result := make([]string, len(actions))
	for index, value := range actions {
		result[index] = value.String()
	}
	return result
}
