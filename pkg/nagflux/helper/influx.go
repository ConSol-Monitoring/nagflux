package helper

import (
	"strings"

	"pkg/nagflux/config"
)

// SanitizeInfluxInput adds backslashes to special chars.
func SanitizeInfluxInput(input string) string {
	if len(input) == 0 {
		return input
	}
	if string(input[0]) == `"` && string(input[len(input)-1]) == `"` {
		return input
	}
	if config.GetConfig().InfluxDBGlobal.NastyString != "" {
		input = strings.ReplaceAll(
			input,
			config.GetConfig().InfluxDBGlobal.NastyString,
			config.GetConfig().InfluxDBGlobal.NastyStringToReplace,
		)
	}
	input = strings.Trim(input, `'`)
	input = strings.ReplaceAll(input, " ", `\ `)
	input = strings.ReplaceAll(input, ",", `\,`)

	return input
}

// SanitizeMap calls SanitizeInfluxInput in key and value
func SanitizeMap(input map[string]string) map[string]string {
	result := map[string]string{}
	for k, v := range input {
		result[SanitizeInfluxInput(k)] = SanitizeInfluxInput(v)
	}
	return result
}
