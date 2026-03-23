//go:build windows

package main

import "golang.org/x/sys/windows/registry"

func detectEnvSources(processValues map[string]string) map[string]string {
	userKeys := readRegistryValueNames(registry.CURRENT_USER, `Environment`)
	systemKeys := readRegistryValueNames(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\Session Manager\Environment`)

	sources := make(map[string]string, len(processValues))
	for key := range processValues {
		_, inUser := userKeys[key]
		_, inSystem := systemKeys[key]

		switch {
		case inUser && inSystem:
			sources[key] = sourceUserOverride
		case inUser:
			sources[key] = sourceUser
		case inSystem:
			sources[key] = sourceSystem
		default:
			sources[key] = sourceProcess
		}
	}

	return sources
}

func readRegistryValueNames(root registry.Key, path string) map[string]struct{} {
	names := make(map[string]struct{})

	key, err := registry.OpenKey(root, path, registry.READ)
	if err != nil {
		return names
	}
	defer key.Close()

	valueNames, err := key.ReadValueNames(-1)
	if err != nil {
		return names
	}

	for _, name := range valueNames {
		names[name] = struct{}{}
	}

	return names
}
