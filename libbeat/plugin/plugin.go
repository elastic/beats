package plugin

import "fmt"

type PluginLoader func(p interface{}) error

var registry = map[string]PluginLoader{}

func Bundle(
	bundles ...map[string][]interface{},
) map[string][]interface{} {
	ret := map[string][]interface{}{}

	for _, bundle := range bundles {
		for name, plugins := range bundle {
			ret[name] = append(ret[name], plugins...)
		}
	}

	return ret
}

func MakePlugin(key string, ifc interface{}) map[string][]interface{} {
	return map[string][]interface{}{
		key: {ifc},
	}
}

func MustRegisterLoader(name string, l PluginLoader) {
	err := RegisterLoader(name, l)
	if err != nil {
		panic(err)
	}
}

func RegisterLoader(name string, l PluginLoader) error {
	if l := registry[name]; l != nil {
		return fmt.Errorf("plugin loader '%v' already registered", name)
	}

	registry[name] = l
	return nil
}

func LoadPlugins(path string) error {
	// TODO: add flag to enable/disable plugins?
	return loadPlugins(path)
}
