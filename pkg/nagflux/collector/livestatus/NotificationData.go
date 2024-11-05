package livestatus

import (
	"fmt"
	"strings"

	"pkg/nagflux/collector"
	"pkg/nagflux/helper"
	"pkg/nagflux/logging"
)

// NotificationData adds notification types to the livestatus data
type NotificationData struct {
	collector.Filterable
	Data
	notificationType  string
	notificationLevel string
}

func (notification *NotificationData) sanitizeValues() {
	notification.Data.sanitizeValues()
	notification.notificationType = helper.SanitizeInfluxInput(notification.notificationType)
	notification.notificationLevel = helper.SanitizeInfluxInput(notification.notificationLevel)
}

// PrintForInfluxDB prints the data in influxdb lineformat
func (notification NotificationData) PrintForInfluxDB(version string) string {
	notification.sanitizeValues()
	if helper.VersionOrdinal(version) >= helper.VersionOrdinal("0.9") {
		var tags string
		if text := notificationToText(notification.notificationType); text != "" {
			tags = ",type=" + text
		}
		value := fmt.Sprintf("%s:<br> %s", strings.TrimSpace(notification.notificationLevel), notification.comment)
		return notification.genInfluxLineWithValue(tags, value)
	}
	logging.GetLogger().Criticalf("This influxversion [%s] given in the config is not supported", version)
	panic("")
}

// PrintForElasticsearch prints in the elasticsearch json format
func (notification NotificationData) PrintForElasticsearch(version, index string) string {
	if helper.VersionOrdinal(version) >= helper.VersionOrdinal("2.0") {
		text := notificationToText(notification.notificationType)
		value := fmt.Sprintf("%s:<br> %s", strings.TrimSpace(notification.notificationLevel), notification.comment)
		return notification.genElasticLineWithValue(index, text, value, notification.entryTime)
	}
	logging.GetLogger().Criticalf("This elasticsearchversion [%s] given in the config is not supported", version)
	panic("")
}

func notificationToText(input string) string {
	switch input {
	case `HOST NOTIFICATION`:
		return "host_notification"
	case `HOST\ NOTIFICATION`:
		return "host_notification"
	case `SERVICE NOTIFICATION`:
		return "service_notification"
	case `SERVICE\ NOTIFICATION`:
		return "service_notification"
	}
	logging.GetLogger().Warn("This notification type is not supported:" + input)
	return ""
}
