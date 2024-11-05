package livestatus

import (
	"fmt"
	"strings"

	"github.com/ConSol-Monitoring/nagflux/collector"
	"github.com/ConSol-Monitoring/nagflux/helper"
	"github.com/ConSol-Monitoring/nagflux/logging"
)

// DowntimeData adds Comments types to the livestatus data
type DowntimeData struct {
	collector.Filterable
	Data
	endTime string
}

func (downtime *DowntimeData) sanitizeValues() {
	downtime.Data.sanitizeValues()
	downtime.endTime = helper.SanitizeInfluxInput(downtime.endTime)
}

// PrintForInfluxDB prints the data in influxdb lineformat
func (downtime DowntimeData) PrintForInfluxDB(version string) string {
	downtime.sanitizeValues()
	if helper.VersionOrdinal(version) >= helper.VersionOrdinal("0.9") {
		tags := ",type=downtime,author=" + downtime.author
		start := fmt.Sprintf("%s%s message=\"%s\" %s", downtime.getTablename(), tags, strings.TrimSpace("Downtime start: <br>"+downtime.comment), helper.CastStringTimeFromSToMs(downtime.entryTime))
		end := fmt.Sprintf("%s%s message=\"%s\" %s", downtime.getTablename(), tags, strings.TrimSpace("Downtime end: <br>"+downtime.comment), helper.CastStringTimeFromSToMs(downtime.endTime))
		return start + "\n" + end
	}
	logging.GetLogger().Criticalf("This influxversion [%s] given in the config is not supported", version)
	panic("")
}

// PrintForElasticsearch prints in the elasticsearch json format
func (downtime DowntimeData) PrintForElasticsearch(version, index string) string {
	if helper.VersionOrdinal(version) >= helper.VersionOrdinal("2.0") {
		typ := `downtime`
		start := downtime.genElasticLineWithValue(index, typ, strings.TrimSpace("Downtime start: <br>"+downtime.comment), downtime.entryTime)
		end := downtime.genElasticLineWithValue(index, typ, strings.TrimSpace("Downtime end: <br>"+downtime.comment), downtime.endTime)
		return start + "\n" + end
	}
	logging.GetLogger().Criticalf("This elasticsearchversion [%s] given in the config is not supported", version)
	panic("")
}
