package livestatus

import (
	"testing"
	"time"

	"github.com/ConSol-Monitoring/nagflux/pkg/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	intervalToCheckLivestatusCache = 2 * time.Second
}

func TestNewCacheBuilder(t *testing.T) {
	logging.InitTestLogger()
	connector := &Connector{logging.GetLogger(), "localhost:6558", "tcp"}
	builder := NewLivestatusCacheBuilder(connector)
	require.NotNilf(t, builder, "Constructor returned pointer")
	builder.Stop()
}

func TestDisabledServiceInDowntime(t *testing.T) {
	logging.InitTestLogger()
	queries := map[string]string{}
	queries[QueryForServicesInDowntime] = "1,2;host1;service1\n"
	queries[QueryForHostsInDowntime] = "3,4;host1\n5;host2\n"
	queries[QueryForDowntimeid] = "1;0;1\n2;2;3\n3;0;1\n4;1;2\n5;2;1\n"
	livestatus := &MockLivestatus{LivestatusAddress: "localhost:6558", ConnectionType: "tcp", Queries: queries, isRunning: true}
	go livestatus.StartMockLivestatus()
	connector := &Connector{logging.GetLogger(), livestatus.LivestatusAddress, livestatus.ConnectionType}

	cacheBuilder := NewLivestatusCacheBuilder(connector)

	intern := map[string]map[string]string{"host1": {"": "2", "service1": "3"}, "host2": {"": "2"}}

	// wait 10 seconds till cache matches
	assert.Eventually(t, func() bool {
		cacheBuilder.mutex.Lock()
		defer cacheBuilder.mutex.Unlock()
		return assert.ObjectsAreEqual(intern, cacheBuilder.downtimeCache.downtime)
	}, 10*time.Second, 100*time.Millisecond, "internal cache does not fit.")

	cacheBuilder.Stop()
	livestatus.StopMockLivestatus()

	cacheBuilder.mutex.Lock()
	assert.Equalf(t, intern, cacheBuilder.downtimeCache.downtime, "internal cache does not fit.")
	cacheBuilder.mutex.Unlock()

	assert.Falsef(t, cacheBuilder.IsServiceInDowntime("host1", "service1", "0"), `"host1","service1","0" should not be in downtime`)
	assert.Falsef(t, cacheBuilder.IsServiceInDowntime("host1", "service1", "1"), `"host1","service1","1" should not be in downtime, downtime starts at 3`)
	assert.Falsef(t, cacheBuilder.IsServiceInDowntime("host1", "service1", "2"), `"host1","service1","2" should not be in downtime, downtime starts at 3`)
	assert.Truef(t, cacheBuilder.IsServiceInDowntime("host1", "service1", "3"), `"host1","service1","3" should be in downtime`)
	assert.Falsef(t, cacheBuilder.IsServiceInDowntime("host1", "", "1"), `"host1","","1" should not be in downtime, downtime starts at 2`)
	assert.Falsef(t, cacheBuilder.IsServiceInDowntime("host1", "", "0"), `"host1","","0" should not be in downtime`)
	assert.Truef(t, cacheBuilder.IsServiceInDowntime("host1", "", "2"), `"host1","","2" should be in downtime`)
}
