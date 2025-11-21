package helper

import (
	"fmt"
	"pkg/nagflux/config"
	"strings"
)

// CreateJSONFromStringMap creates a part of a JSON object
func CreateJSONFromStringMap(input map[string]string) string {
	str := strings.Builder{}
	for k, v := range input {
		str.WriteString(fmt.Sprintf(`,%s:%s`, GenJSONValueString(k), GenJSONValueString(v)))
	}
	return str.String()
}

// GenJSONValueString quotes the string if it's not a number.
func GenJSONValueString(input string) string {
	if IsStringANumber(input) {
		return input
	}
	return fmt.Sprintf(`"%s"`, input)
}

// SanitizeElasicInput escapes backslashes and trims single ticks.
func SanitizeElasicInput(input string) string {
	input = strings.Trim(input, `'`)
	input = strings.ReplaceAll(input, `\`, `\\`)
	input = strings.ReplaceAll(input, `"`, `\"`)
	return input
}

// GenIndex generates an index depending on the config, ending with year and month
func GenIndex(index, timeString string) string {
	rotation := config.GetConfig().ElasticsearchGlobal.IndexRotation
	year, month := GetYearMonthFromStringTimeMs(timeString)
	switch rotation {
	case "monthly":
		return fmt.Sprintf("%s-%d.%02d", index, year, month)
	case "yearly":
		return fmt.Sprintf("%s-%d", index, year)
	default:
		panic(fmt.Sprintf("The given IndexRotation[%s] is not supported", rotation))
	}
}
