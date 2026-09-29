package livestatus

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddDowntime(t *testing.T) {
	cache := Cache{make(map[string]map[string]string)}
	assert.Equal(t, make(map[string]map[string]string), cache.downtime, "cache should be empty at the beginning")

	cache.addDowntime("hostname", "servicename", "123")
	assert.Equal(t, map[string]map[string]string{"hostname": {"servicename": "123"}}, cache.downtime, "added element is missing")

	cache.addDowntime("hostname2", "", "123")
	assert.Equal(t, map[string]map[string]string{"hostname": {"servicename": "123"}, "hostname2": {"": "123"}}, cache.downtime, "added element is missing")

	cache.addDowntime("hostname2", "", "1")
	assert.Equal(t, map[string]map[string]string{"hostname": {"servicename": "123"}, "hostname2": {"": "123"}}, cache.downtime, "an older timestamp must not replace the newer one")
}
