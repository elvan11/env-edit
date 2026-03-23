//go:build !windows

package main

func detectEnvSources(processValues map[string]string) map[string]string {
	sources := make(map[string]string, len(processValues))
	for key := range processValues {
		sources[key] = sourceProcess
	}
	return sources
}
