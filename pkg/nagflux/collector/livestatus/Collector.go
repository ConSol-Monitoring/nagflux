package livestatus

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"pkg/nagflux/collector"
	"pkg/nagflux/config"
	"pkg/nagflux/data"
	"pkg/nagflux/filter"
	"pkg/nagflux/helper"
	"pkg/nagflux/logging"

	"github.com/kdar/factorlog"
)

// Collector fetches data from livestatus.
type Collector struct {
	quit                  chan bool
	jobs                  collector.ResultQueues
	livestatusConnector   *Connector
	log                   *factorlog.FactorLog
	logNotificationsQuery string
	commentsQuery         string
	downtimesQuery        string
	filterProcessor       filter.Processor
}

type queryType int

const (
	queryTypeNotification queryType = iota
	queryTypeComments
	queryTypeDowntimes
	queryTypeStatus
)

const (
	// Updateinterval on livestatus data for Icinga2.
	intervalToCheckLivestatus = time.Duration(2) * time.Minute
	QueryLivestatusVersion    = `GET status
Columns: livestatus_version
OutputFormat: csv

`
	// QueryIcinga2ForNotifications livestatus query for notifications with Icinga2 Livestatus.
	QueryIcinga2ForNotifications = `GET log
Columns: type time contact_name message
Filter: type ~ .*NOTIFICATION
Filter: time < %d
Negate:
OutputFormat: csv

`
	// QueryNagiosForNotifications livestatus query for notifications with nagioslike Livestatus.
	QueryNagiosForNotifications = `GET log
Columns: type time contact_name message
Filter: type ~ .*NOTIFICATION
Filter: time > %d
OutputFormat: csv

`
	// QueryForComments livestatus query for comments
	QueryForComments = `GET comments
Columns: host_name service_display_name comment entry_time author entry_type
Filter: entry_time > %d
OutputFormat: csv

`
	// QueryForDowntimes livestatus query for downtimes
	QueryForDowntimes = `GET downtimes
Columns: host_name service_display_name comment entry_time author end_time
Filter: entry_time > %d
OutputFormat: csv

`
)

const (
	// Nagios nagioslike Livestatus
	Nagios = iota
	// Icinga2 icinga2like Livestatus
	Icinga2
	// Naemon naemonlike Livestatus
	Naemon
)

// NewLivestatusCollector constructor, which also starts it immediately.
func NewLivestatusCollector(jobs collector.ResultQueues, livestatusConnector *Connector, detectVersion string) *Collector {
	cfg := config.GetConfig()
	live := &Collector{
		quit:                make(chan bool, 2),
		jobs:                jobs,
		livestatusConnector: livestatusConnector,
		log:                 logging.GetLogger(),
		filterProcessor:     filter.NewFilter(cfg.Filter.LivestatusLineTerms),
	}
	live.logNotificationsQuery = livestatusConnector.buildQuery(QueryNagiosForNotifications, cfg.Filter.LivestatusNotificationsFilter)
	live.commentsQuery = livestatusConnector.buildQuery(QueryForComments, cfg.Filter.LivestatusCommentsFilter)
	live.downtimesQuery = livestatusConnector.buildQuery(QueryForDowntimes, cfg.Filter.LivestatusDowntimesFilter)

	live.log.Debugf("query notifications: %s", live.logNotificationsQuery)
	live.log.Debugf("query comments: %s", live.commentsQuery)
	live.log.Debugf("query downtimes: %s", live.downtimesQuery)

	if detectVersion == "" {
		switch getLivestatusVersion(live) {
		case Nagios:
			live.log.Info("Livestatus type: Nagios")
		case Icinga2:
			live.log.Info("Livestatus type: Icinga2")
			live.logNotificationsQuery = QueryIcinga2ForNotifications
		case Naemon:
			live.log.Info("Livestatus type: Naemon")
		}
	} else {
		switch detectVersion {
		case "Nagios":
			live.log.Info("Setting Livestatus version to: Nagios")
		case "Icinga2":
			live.log.Info("Setting Livestatus version to: Icinga2")
			live.logNotificationsQuery = QueryIcinga2ForNotifications
		case "Naemon":
			live.log.Info("Setting Livestatus version to: Naemon")
		default:
			live.log.Info("Given Livestatusversion is unknown, using Nagios")
		}
	}
	go live.run()
	return live
}

// Stop signals the collector to stop.
func (live *Collector) Stop() {
	live.quit <- true
	<-live.quit
	live.log.Debug("LivestatusCollector stopped")
}

// Loop which checks livestats for data or waits to quit.
func (live *Collector) run() {
	live.queryData()
	for {
		select {
		case <-live.quit:
			live.quit <- true
			return
		case <-time.After(intervalToCheckLivestatus):
			live.queryData()
		}
	}
}

// Queries livestatus and returns the data to the global queue
func (live *Collector) queryData() {
	printables := make(chan collector.Printable)
	finished := make(chan bool)
	go live.requestPrintablesFromLivestatus(queryTypeNotification, live.logNotificationsQuery, true, printables, finished)
	go live.requestPrintablesFromLivestatus(queryTypeComments, live.commentsQuery, true, printables, finished)
	go live.requestPrintablesFromLivestatus(queryTypeDowntimes, live.downtimesQuery, true, printables, finished)
	jobsFinished := 0
	for jobsFinished < 3 {
		select {
		case job := <-printables:
			for _, j := range live.jobs {
				j <- job
			}
		case <-finished:
			jobsFinished++
		case <-time.After(intervalToCheckLivestatus):
			live.log.Warn("Livestatus timed out... (Collector.queryData())")
		}
	}
}

func (live *Collector) requestPrintablesFromLivestatus(queryType queryType, query string, addTimestampToQuery bool, printables chan collector.Printable, outerFinish chan bool) {
	queryWithTimestamp := query
	if addTimestampToQuery {
		queryWithTimestamp = addTimestampToLivestatusQuery(query)
	}

	csv := make(chan []string)
	finished := make(chan bool)
	go live.livestatusConnector.connectToLivestatus(queryWithTimestamp, csv, finished)

	for {
		select {
		case line := <-csv:
			logging.GetLogger().Debugf("[%d] livestatus line %#v", queryType, line)
			if skipLine := live.filterProcessor.TestLine([]byte(strings.Join(line, config.GetConfig().Main.FieldSeparator))); !skipLine {
				logging.GetLogger().Debugf("skipping line %#v", line)

				continue
			}
			switch queryType {
			case queryTypeNotification:
				if printable := live.handleQueryForNotifications(line); printable != nil {
					printables <- printable
				}
			case queryTypeComments:
				if len(line) == 6 {
					printables <- &CommentData{collector.AllFilterable, Data{line[0], line[1], line[2], line[3], line[4]}, line[5]}
				} else {
					live.log.Warn("QueryForComments out of range", line)
				}
			case queryTypeDowntimes:
				if len(line) == 6 {
					live.log.Debugf("adding downtime: %#v", line)
					printables <- &DowntimeData{collector.AllFilterable, Data{line[0], line[1], line[2], line[3], line[4]}, line[5]}
				} else {
					live.log.Warn("QueryForDowntimes out of range", line)
				}
			case queryTypeStatus:
				if len(line) == 1 {
					printables <- &collector.SimplePrintable{Filterable: collector.AllFilterable, Text: line[0], Datatype: data.InfluxDB}
				} else {
					live.log.Warn("QueryLivestatusVersion out of range", line)
				}
			default:
				live.log.Fatal("Found unknown query type" + query)
			}
		case result := <-finished:
			outerFinish <- result
			return
		case <-time.After(intervalToCheckLivestatus / 3):
			live.log.Warn("connectToLivestatus timed out")
		}
	}
}

func addTimestampToLivestatusQuery(query string) string {
	return fmt.Sprintf(query, time.Now().Add(intervalToCheckLivestatus/100*-150).Unix())
}

func (live *Collector) handleQueryForNotifications(line []string) *NotificationData {
	switch line[0] {
	case "HOST NOTIFICATION":
		if len(line) == 10 {
			// Custom: host_name, "", message, timestamp, author, notification_type, state
			return &NotificationData{collector.AllFilterable, Data{line[4], "", line[9], line[1], line[8]}, line[0], line[5]}
		} else if len(line) == 9 {
			return &NotificationData{collector.AllFilterable, Data{line[4], "", line[7], line[1], line[2]}, line[0], line[5]}
		} else if len(line) == 8 {
			return &NotificationData{collector.AllFilterable, Data{line[4], "", line[7], line[1], line[2]}, line[0], line[5]}
		}
		live.log.Warn("HOST NOTIFICATION, undefined line length: ", len(line), " Line:", helper.SPrintStringSlice(line))
	case "SERVICE NOTIFICATION":
		if len(line) == 11 {
			// Custom
			return &NotificationData{collector.AllFilterable, Data{line[4], line[5], line[10], line[1], line[9]}, line[0], line[6]}
		} else if len(line) == 10 || len(line) == 9 {
			return &NotificationData{collector.AllFilterable, Data{line[4], line[5], line[8], line[1], line[2]}, line[0], line[6]}
		}
		live.log.Warn("SERVICE NOTIFICATION, undefined line length: ", len(line), " Line:", helper.SPrintStringSlice(line))
	default:
		if strings.Contains(line[0], "NOTIFICATION SUPPRESSED") {
			live.log.Debugf("Ignoring suppressed Notification: '%s', Line: %s", line[0], helper.SPrintStringSlice(line))
		} else {
			live.log.Warnf("The notification type is unknown: '%s', whole line: '%s'", line[0], helper.SPrintStringSlice(line))
		}
	}
	return nil
}

func getLivestatusVersion(live *Collector) int {
	printables := make(chan collector.Printable, 1)
	finished := make(chan bool, 1)
	var version string
	live.requestPrintablesFromLivestatus(queryTypeStatus, QueryLivestatusVersion, false, printables, finished)
	i := 0
	oneMinute := time.Duration(1) * time.Minute
	roundsToWait := config.GetConfig().Livestatus.MinutesToWait
Loop:
	for roundsToWait != 0 {
		select {
		case versionPrintable := <-printables:
			version = versionPrintable.PrintForInfluxDB("0")
			break Loop
		case <-time.After(oneMinute):
			if i < roundsToWait {
				go live.requestPrintablesFromLivestatus(queryTypeStatus, QueryLivestatusVersion, false, printables, finished)
			} else {
				break Loop
			}
			i++
		case fin := <-finished:
			if !fin {
				live.log.Infof(
					"Could not detect livestatus version, waiting for %s %d times( %d/%d )...",
					oneMinute, roundsToWait, i, roundsToWait,
				)
			}
		}
	}

	live.log.Info("Livestatus version: ", version)
	if icinga2, _ := regexp.MatchString(`^r[\d\.-]+$`, version); icinga2 {
		return Icinga2
	} else if nagios, _ := regexp.MatchString(`^[\d\.]+p[\d\.]+$`, version); nagios {
		return Nagios
	} else if naemon, _ := regexp.MatchString(`^[\d\.]+(source-naemon|-naemon)?$`, version); naemon {
		return Naemon
	}
	live.log.Warn("Could not detect livestatus type, with version: ", version, ". Assuming Nagios")
	return -1
}
