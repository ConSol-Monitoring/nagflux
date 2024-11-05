package livestatus

import (
	"testing"
	"time"

	"pkg/nagflux/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCacheBuilder(t *testing.T) {
	logging.InitTestLogger()
	connector := &Connector{logging.GetLogger(), "localhost:6558", "tcp"}
	builder := NewLivestatusCacheBuilder(connector)
	require.NotNilf(t, builder, "Constructor returned pointer")
}

func TestDisabledServiceInDowntime(t *testing.T) {
	logging.InitTestLogger()
	queries := map[string]string{}
	queries[QueryForServicesInDowntime] = "1,2;host1;service1\n"
	queries[QueryForHostsInDowntime] = "3,4;host1\n5;host2\n"
	queries[QueryForDowntimeid] = "1;0;1\n2;2;3\n3;0;1\n4;1;2\n5;2;1\n"
	livestatus := &MockLivestatus{"localhost:6558", "tcp", queries, true}
	go livestatus.StartMockLivestatus()
	connector := &Connector{logging.GetLogger(), livestatus.LivestatusAddress, livestatus.ConnectionType}

	intervalToCheckLivestatusCache = 2 * time.Second
	cacheBuilder := NewLivestatusCacheBuilder(connector)

	// wait 10 seconds till cache matches
	waitUntil := time.Now().Add(10 * time.Second)
	for time.Now().Before(waitUntil) {
		if cacheBuilder.IsServiceInDowntime("host1", "service1", "1") {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	cacheBuilder.Stop()
	livestatus.StopMockLivestatus()

	intern := map[string]map[string]string{"host1": {"": "1", "service1": "1"}, "host2": {"": "2"}}
	cacheBuilder.mutex.Lock()
	assert.Equalf(t, intern, cacheBuilder.downtimeCache.downtime, "internal cache does not fit.")
	cacheBuilder.mutex.Unlock()

	assert.Truef(t, cacheBuilder.IsServiceInDowntime("host1", "service1", "1"), `"host1","service1","1" should be in downtime`)
	assert.Truef(t, cacheBuilder.IsServiceInDowntime("host1", "service1", "2"), `"host1","service1","2" should be in downtime`)
	assert.Falsef(t, cacheBuilder.IsServiceInDowntime("host1", "service1", "0"), `"host1","service1","0" should not be in downtime`)
	assert.Falsef(t, cacheBuilder.IsServiceInDowntime("host1", "", "0"), `"host1","","0" should not be in downtime`)
	assert.Truef(t, cacheBuilder.IsServiceInDowntime("host1", "", "2"), `"host1","","2" should not be in downtime`)
}
