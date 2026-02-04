package helper

import (
	"reflect"
	"strings"

	"pkg/nagflux/logging"
)

// Config parser tries to populate the Config structy type with values from config file
// If the Config has a field, but config file does not specify it, it will be default initialized
// If the Config has a field that is a pointer to a type
// Option 1: Config file has it -> the pointer will point to an instance of that type
// Option 2: Config file does not have it -> the pointer will be set to nil
// This way we can determine if something is explicitly set in the config
// But the type of that value has to be a pointer, like *bool or *string

// GetPreferredConfigValue retrieves a value from a struct using a dot-separated path.
// It takes a primaryPath and other deprecatedPaths. If a value is found on the mainPath, it notifies the user about deprecatedPaths
func GetPreferredConfigValue(config any, primaryPath string, deprecatedPaths []string) (value any, found bool) {
	log := logging.GetLogger()

	if value, found := getConfigValueByPath(config, primaryPath); found {
		// if its not default, it should be explicitly set
		// report deprecated paths if you found the value in primary path
		for _, deprecatedPath := range deprecatedPaths {
			if _, found := getConfigValueByPath(config, deprecatedPath); found {
				log.Debugf("Config option '%s' is taken instead of '%s' as it has precedence", primaryPath, deprecatedPath)
			}
		}
		return value, true
	}

	// then try deprecated paths in order
	for _, deprecatedPath := range deprecatedPaths {
		if value, found := getConfigValueByPath(config, deprecatedPath); found {
			// Check if this deprecated value was explicitly set
			log.Debugf("Config option '%s' is deprecated, use '%s' instead", deprecatedPath, primaryPath)
			return value, true
		}
	}

	log.Warnf("No values found in config option with primaryPath '%s' and deprecated paths '%v' ", primaryPath, deprecatedPaths)

	return nil, false
}

// getConfigValueByPath recursively traverses a struct using dot-separated field names
// if a value is found, but if its a nil pointer or an interface that contains a nil pointer, found is set to false
func getConfigValueByPath(rootObject any, path string) (value any, found bool) {
	splits := strings.Split(path, ".")
	current := reflect.ValueOf(rootObject)

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
			// check if its nil or interface to a nil ptr e.g: interface {}(*bool) nil
			if !found || isNilOrInterfaceToNilPointer(value) {
				return nil, false
			}
			return value, found
		}

		// we arent in the last split, have to go deeper
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

// Helper function to detect if a value is a nil pointer
func isNilOrInterfaceToNilPointer(v any) bool {
	if v == nil {
		return false
	}
	rv := reflect.ValueOf(v)
	return rv.Kind() == reflect.Ptr && rv.IsNil()
}

// getFieldValue gets the value of a struct field by name
// If its does not exist in that object, returned value is: (nil,false)
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
