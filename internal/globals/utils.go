package globals

import "strings"

// CopyMap return a map that is a real copy of the original
// Ref: https://go.dev/blog/maps
func CopyMap(src map[string]interface{}) map[string]interface{} {
	m := make(map[string]interface{}, len(src))
	for k, v := range src {
		m[k] = v
	}
	return m
}

// SplitCommaSeparatedValues get a list of strings and return a new list
// where each element containing commas is divided in separated elements
func SplitCommaSeparatedValues(input []string) []string {
	var result []string
	for _, item := range input {
		parts := strings.Split(item, ",")
		result = append(result, parts...)
	}
	return result
}
