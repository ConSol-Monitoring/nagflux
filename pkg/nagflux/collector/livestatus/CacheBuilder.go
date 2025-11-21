package livestatus

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"pkg/nagflux/logging"

	"github.com/kdar/factorlog"
)

// CacheBuilder fetches data from livestatus.
type CacheBuilder struct {
	livestatusConnector *Connector
	quit                chan bool
	log                 *factorlog.FactorLog
	downtimeCache       Cache
	mutex               *sync.Mutex
}

const (
	// default update interval on livestatus data.
	defaultIntervalToCheckLivestatusCache = time.Duration(30) * time.Second

	// QueryForServicesInDowntime livestatus query for services in downtime.
	QueryForServicesInDowntime = `GET services
Columns: downtimes host_name display_name
Filter: scheduled_downtime_depth > 0
OutputFormat: csv

`
	// QueryForHostsInDowntime livestatus query for hosts in downtime
	QueryForHostsInDowntime = `GET hosts
Columns: downtimes name
Filter: scheduled_downtime_depth > 0
OutputFormat: csv

`
	// QueryForDowntimeid livestatus query for downtime start/end
	QueryForDowntimeid = `GET downtimes
Columns: id start_time entry_time
OutputFormat: csv

`
)

var intervalToCheckLivestatusCache = defaultIntervalToCheckLivestatusCache

// NewLivestatusCacheBuilder constructor, which also starts it immediately.
func NewLivestatusCacheBuilder(livestatusConnector *Connector) *CacheBuilder {
	cache := &CacheBuilder{livestatusConnector, make(chan bool, 2), logging.GetLogger(), Cache{make(map[string]map[string]string)}, &sync.Mutex{}}
	go cache.run(intervalToCheckLivestatusCache)
	return cache
}

// Stop signals the cache to stop.
func (builder *CacheBuilder) Stop() {
	builder.quit <- true
	<-builder.quit
	builder.log.Debug("LivestatusCacheBuilder stopped")
}

// Loop which caches livestatus downtimes and waits to quit.
func (builder *CacheBuilder) run(checkInterval time.Duration) {
	newCache := builder.createLivestatusCache()
	builder.mutex.Lock()
	builder.downtimeCache = newCache
	builder.mutex.Unlock()
	for {
		select {
		case <-builder.quit:
			builder.quit <- true
			return
		case <-time.After(checkInterval):
			newCache = builder.createLivestatusCache()
			builder.mutex.Lock()
			builder.downtimeCache = newCache
			builder.mutex.Unlock()
		}
	}
}

// Builds host/service map which are in downtime
func (builder *CacheBuilder) createLivestatusCache() Cache {
	result := Cache{downtime: make(map[string]map[string]string)}
	downtimeCsv := make(chan []string)
	finishedDowntime := make(chan bool)
	hostServiceCsv := make(chan []string)
	finished := make(chan bool)
	go builder.livestatusConnector.connectToLivestatus(QueryForDowntimeid, downtimeCsv, finishedDowntime)
	go builder.livestatusConnector.connectToLivestatus(QueryForHostsInDowntime, hostServiceCsv, finished)
	go builder.livestatusConnector.connectToLivestatus(QueryForServicesInDowntime, hostServiceCsv, finished)

	jobsFinished := 0
	// contains id to starttime
	downtimes := map[string]string{}
	for jobsFinished < 2 {
		select {
		case downtimesLine := <-downtimeCsv:
			if len(downtimesLine) < 3 {
				builder.log.Debug("downtimesLine", downtimesLine)
				break
			}
			startTime, _ := strconv.Atoi(downtimesLine[1])
			entryTime, _ := strconv.Atoi(downtimesLine[2])
			latestTime := startTime
			if startTime < entryTime {
				latestTime = entryTime
			}
			for id := range strings.SplitSeq(downtimesLine[0], ",") {
				downtimes[id] = strconv.Itoa(latestTime)
			}
		case <-finishedDowntime:
			for jobsFinished < 2 {
				select {
				case hostService := <-hostServiceCsv:
					for id := range strings.SplitSeq(hostService[0], ",") {
						if len(hostService) == 2 {
							result.addDowntime(hostService[1], "", downtimes[id])
						} else if len(hostService) == 3 {
							result.addDowntime(hostService[1], hostService[2], downtimes[id])
						}
					}
				case <-finished:
					jobsFinished++
				case <-time.After(intervalToCheckLivestatusCache / 3):
					builder.log.Info("Livestatus timed out...(host/service)")
					return result
				}
			}
		case <-time.After(intervalToCheckLivestatusCache / 3):
			builder.log.Info("Livestatus timed out...(downtimes)")
			return result
		}
	}
	return result
}

// IsServiceInDowntime returns true if the host/service is in downtime
func (builder *CacheBuilder) IsServiceInDowntime(host, service, time string) bool {
	result := false
	builder.mutex.Lock()
	if _, hostExists := builder.downtimeCache.downtime[host]; hostExists {
		if _, serviceExists := builder.downtimeCache.downtime[host][service]; serviceExists {
			if builder.downtimeCache.downtime[host][service] <= time {
				result = true
			}
		}
	}

	builder.mutex.Unlock()
	return result
}
