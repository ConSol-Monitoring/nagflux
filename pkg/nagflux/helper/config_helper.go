package helper

import (
	"reflect"
	"strings"

	"pkg/nagflux/logging"
)

// GetConfigValue retrieves a value from a struct using a dot-separated path , while notifying about deprecated paths
func GetConfigValue(config any, primaryPath string, deprecatedPaths []string) (value any, found bool) {
	log := logging.GetLogger()

	if value, found := getConfigValueByPath(config, primaryPath); found {
		// report deprecated paths if you foudn the value in primary path
		for _, deprecatedPath := range deprecatedPaths {
			if _, found := getConfigValueByPath(config, deprecatedPath); found {
				log.Infof("Warning: Config option '%s' is taken as it has predence over '%s'\n", primaryPath, deprecatedPath)
			}
		}

		return value, true
	}

	// Then try deprecated paths in order
	for _, deprecatedPath := range deprecatedPaths {
		if value, found := getConfigValueByPath(config, deprecatedPath); found {
			log.Infof("Warning: Config option '%s' is deprecated, use '%s' instead\n", deprecatedPath, primaryPath)
			return value, true
		}
	}

	return nil, false
}

// getConfigValueByPath recursively traverses a struct using dot-separated field names
func getConfigValueByPath(configObject any, path string) (value any, found bool) {
	// dot separate the name string for traversal
	splits := strings.Split(path, ".")
	current := reflect.ValueOf(configObject)

	if current.Kind() == reflect.Ptr {
		if current.IsNil() {
			return nil, false
		}
		current = current.Elem()
	}

	for i, field := range splits {
		// if we reached last split, return the value
		if i == len(splits)-1 {
			value, found := getFieldValue(current, field)
			return value, found
		}

		// navigate deeper into the struct if not found

		fieldValue, found := getFieldValue(current, field)
		if !found {
			return nil, false
		}
		current = reflect.ValueOf(fieldValue)
		if current.Kind() == reflect.Ptr {
			if current.IsNil() {
				return nil, false
			}
			current = current.Elem()
		}
	}

	return nil, false
}

// getFieldValue gets the value of a struct field by name
func getFieldValue(object reflect.Value, fieldName string) (value any, found bool) {
	// if its a pointer we have to get the name of it
	if object.Kind() == reflect.Ptr {
		if object.IsNil() {
			return nil, false
		}
		object = object.Elem()
	}

	// find it by name, and check if its valid
	field := object.FieldByName(fieldName)
	if !field.IsValid() {
		// iterate through the fields again, and match the name
		for i := range object.NumField() {
			// try case-insensitive match
			if strings.EqualFold(object.Type().Field(i).Name, fieldName) {
				field = object.Field(i)
				break
			}
		}
	}

	if !field.IsValid() {
		return nil, false
	}

	// the return type is the interface
	return field.Interface(), true
}
