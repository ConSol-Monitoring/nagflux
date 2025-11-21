package helper

import (
	"maps"
	"strings"
)

// CopyMap creates a real copy of a string to string map.
func CopyMap(old map[string]string) map[string]string {
	newMap := map[string]string{}
	maps.Copy(newMap, old)
	return newMap
}

// PrintMapAsString prints a map in the influxdb tags format.
func PrintMapAsString(toPrint map[string]string, fieldSeparator, assignmentSeparator string) string {
	str := strings.Builder{}
	for key, value := range toPrint {
		str.WriteString(key + assignmentSeparator + value + fieldSeparator)
	}
	result := strings.Trim(str.String(), fieldSeparator)
	return result
}
