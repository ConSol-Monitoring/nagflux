package nagflux

import (
	"fmt"

	"github.com/ConSol-Monitoring/nagflux/collector"
	"github.com/ConSol-Monitoring/nagflux/helper"
)

// Printable converts from nagfluxfile format to X
type Printable struct {
	collector.Filterable
	Table     string
	Timestamp string
	tags      map[string]string
	fields    map[string]string
}

// PrintForInfluxDB prints the data in influxdb lineformat
func (p Printable) PrintForInfluxDB(version string) string {
	if helper.VersionOrdinal(version) >= helper.VersionOrdinal("0.9") {
		line := p.Table
		if len(p.tags) > 0 {
			line += `,` + helper.PrintMapAsString(p.tags, ",", "=")
		}
		line += " "
		if len(p.fields) > 0 {
			line += helper.PrintMapAsString(p.fields, ",", "=")
		}
		return line + ` ` + p.Timestamp
	}
	return ""
}

// PrintForElasticsearch prints in the elasticsearch json format
func (p Printable) PrintForElasticsearch(version, index string) string {
	if helper.VersionOrdinal(version) >= helper.VersionOrdinal("2.0") {
		head := fmt.Sprintf(`{"index":{"_index":"%s","_type":"%s"}}`, helper.GenIndex(index, p.Timestamp), p.Table) + "\n"
		data := `{"timestamp":` + p.Timestamp
		data += helper.CreateJSONFromStringMap(p.tags)
		data += helper.CreateJSONFromStringMap(p.fields)
		data += "}\n"
		return head + data
	}
	return ""
}
