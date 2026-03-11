package modgearman

import (
	"os"
	"strings"
)

// DefaultModGearmanKeyLength length of an gearman key
const DefaultModGearmanKeyLength = 32

// GetSecret parses the mod_gearman secret/file and returns one key.
func GetSecret(secret, secretFile string) string {
	if secret != "" {
		return secret
	}
	if secretFile != "" {
		data, err := os.ReadFile(secretFile)
		if err != nil {
			panic(err)
		}
		return strings.TrimSpace(string(data))
	}
	return ""
}

// ShapeKey expands the key to length, or cuts it.
func ShapeKey(key string, length int) []byte {
	for i := 0; i <= length-len(key); i++ {
		key = key + string([]rune{'\x00'})
	}
	return []byte(key)[:length]
}
