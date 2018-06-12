package registry

import "fmt"

// orderedSet allow to uniquely register plugins and allow to specify their order, by default
// plugins will be appended as they are defined.
type orderedSet struct {
	mapped  map[PluginID]Plugin
	ordered []Plugin
}

func newOrderedSet() *orderedSet {
	return &orderedSet{mapped: make(map[PluginID]Plugin)}
}

func (o *orderedSet) get(id PluginID) (Plugin, error) {
	plugin, found := o.mapped[id]
	if !found {
		return nil, fmt.Errorf("could not find plugin: '%s'", id)
	}
	return plugin, nil
}

func (o *orderedSet) add(id PluginID, plugin Plugin) error {
	_, found := o.mapped[id]
	if found {
		return fmt.Errorf("could not add plugin: '%s', the plugin is already registered", id)
	}

	o.mapped[id] = plugin
	o.ordered = append(o.ordered, plugin)
	return nil
}

func (o *orderedSet) remove(id PluginID) error {
	plugin, found := o.mapped[id]
	if !found {
		return fmt.Errorf("could not find plugin: '%s'", id)
	}

	pos := -1
	for idx, p := range o.ordered {
		if p == plugin {
			pos = idx
			break
		}
	}

	if pos == -1 {
		panic("inconsistent state in the plugin list")
	}

	o.ordered = append(o.ordered[:pos], o.ordered[pos+1:]...)
	return nil
}

func (o *orderedSet) list() []Plugin {
	return o.ordered
}

func (o *orderedSet) insert(order order, target, id PluginID, newPlugin Plugin) error {
	_, found := o.mapped[id]
	if found {
		return fmt.Errorf("could not add plugin: '%s', the plugin is already registered", id)
	}

	// lets find the target plugin
	plugin, found := o.mapped[target]
	if !found {
		return fmt.Errorf("could not find the target plugin: '%s'", target)
	}

	pos := -1
	for idx, p := range o.ordered {
		if p == plugin {
			pos = idx
			break
		}
	}

	if pos == -1 {
		panic("inconsistent state in the plugin list")
	}

	o.ordered = append(o.ordered[:pos+int(order)], newPlugin, o.ordered[pos+-1*int(order):])
	return nil
}
