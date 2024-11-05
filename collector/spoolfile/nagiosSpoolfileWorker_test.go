package spoolfile

import (
	"testing"

	"github.com/ConSol-Monitoring/nagflux/collector"
	"github.com/ConSol-Monitoring/nagflux/helper"
	"github.com/stretchr/testify/assert"
)

var TestPerformanceData = []struct {
	input    string
	expected []PerformanceData
}{
	{
		"DATATYPE::SERVICEPERFDATA	TIMET::1441791000	HOSTNAME::xxx	SERVICEDESC::range	SERVICEPERFDATA::a used=4	SERVICECHECKCOMMAND::check_ranges!-w 3: -c 4: -g :46 -l :48 SERVICESTATE::0	SERVICESTATETYPE::1",
		[]PerformanceData{{
			Hostname:         "xxx",
			Service:          "range",
			Command:          "check_ranges",
			Time:             "1441791000000",
			PerformanceLabel: "a used",
			Unit:             "",
			Tags:             map[string]string{},
			Fields:           map[string]string{"value": "4.0"},
			Filterable:       collector.AllFilterable,
		}},
	},
	{
		`DATATYPE::SERVICEPERFDATA	TIMET::1441791000	HOSTNAME::xxx	SERVICEDESC::range	SERVICEPERFDATA::a used=4 'C:\ used %'=44%;89;94;0;100	SERVICECHECKCOMMAND::check_ranges!-w 3: -c 4: -g :46 -l :48 SERVICESTATE::0	SERVICESTATETYPE::1`,
		[]PerformanceData{{
			Hostname:         "xxx",
			Service:          "range",
			Command:          "check_ranges",
			Time:             "1441791000000",
			PerformanceLabel: "a used",
			Unit:             "",
			Tags:             map[string]string{},
			Fields:           map[string]string{"value": "4.0"},
			Filterable:       collector.AllFilterable,
		}, {
			Hostname:         "xxx",
			Service:          "range",
			Command:          "check_ranges",
			Time:             "1441791000000",
			PerformanceLabel: `'C:\ used %'`,
			Unit:             "%",
			Tags:             map[string]string{"warn-fill": "none", "crit-fill": "none"},
			Fields:           map[string]string{"value": "44.0", "warn": "89.0", "crit": "94.0", "min": "0.0", "max": "100.0"},
			Filterable:       collector.AllFilterable,
		}},
	},
	{
		"DATATYPE::SERVICEPERFDATA	TIMET::1441791001	HOSTNAME::xxx	SERVICEDESC::range	SERVICEPERFDATA::a used=4;2;10	SERVICECHECKCOMMAND::check_ranges!-w 3: -c 4: -g :46 -l :48 SERVICESTATE::0	SERVICESTATETYPE::1",
		[]PerformanceData{{
			Hostname:         "xxx",
			Service:          "range",
			Command:          "check_ranges",
			Time:             "1441791001000",
			PerformanceLabel: "a used",
			Unit:             "",
			Tags:             map[string]string{"warn-fill": "none", "crit-fill": "none"},
			Fields:           map[string]string{"value": "4.0", "warn": "2.0", "crit": "10.0"},
			Filterable:       collector.AllFilterable,
		}},
	},
	{
		"DATATYPE::SERVICEPERFDATA	TIMET::1441791002	HOSTNAME::xxx	SERVICEDESC::range	SERVICEPERFDATA::a used=4;2;10;1;4	SERVICECHECKCOMMAND::check_ranges!-w 3: -c 4: -g :46 -l :48 SERVICESTATE::0	SERVICESTATETYPE::1",
		[]PerformanceData{{
			Hostname:         "xxx",
			Service:          "range",
			Command:          "check_ranges",
			Time:             "1441791002000",
			PerformanceLabel: "a used",
			Unit:             "",
			Tags:             map[string]string{"warn-fill": "none", "crit-fill": "none"},
			Fields:           map[string]string{"value": "4.0", "warn": "2.0", "crit": "10.0", "min": "1.0", "max": "4.0"},
			Filterable:       collector.AllFilterable,
		}},
	},
	{
		"DATATYPE::SERVICEPERFDATA	TIMET::1441791003	HOSTNAME::xxx	SERVICEDESC::range	SERVICEPERFDATA::a used=4;2:4;8:10;1;4	SERVICECHECKCOMMAND::check_ranges!-w 3: -c 4: -g :46 -l :48 SERVICESTATE::0	SERVICESTATETYPE::1",
		[]PerformanceData{{
			Hostname:         "xxx",
			Service:          "range",
			Command:          "check_ranges",
			Time:             "1441791003000",
			PerformanceLabel: "a used",
			Unit:             "",
			Tags:             map[string]string{"warn-fill": "outer", "crit-fill": "outer"},
			Fields:           map[string]string{"value": "4.0", "warn-min": "2.0", "warn-max": "4.0", "crit-min": "8.0", "crit-max": "10.0", "min": "1.0", "max": "4.0"},
			Filterable:       collector.AllFilterable,
		}},
	},
	{
		"DATATYPE::SERVICEPERFDATA	TIMET::1441791004	HOSTNAME::xxx	SERVICEDESC::range	SERVICEPERFDATA::a used=4;@2:4;@8:10;1;4	SERVICECHECKCOMMAND::check_ranges!-w 3: -c 4: -g :46 -l :48 SERVICESTATE::0	SERVICESTATETYPE::1",
		[]PerformanceData{{
			Hostname:         "xxx",
			Service:          "range",
			Command:          "check_ranges",
			Time:             "1441791004000",
			PerformanceLabel: "a used",
			Unit:             "",
			Tags:             map[string]string{"warn-fill": "inner", "crit-fill": "inner"},
			Fields:           map[string]string{"value": "4.0", "warn-min": "2.0", "warn-max": "4.0", "crit-min": "8.0", "crit-max": "10.0", "min": "1.0", "max": "4.0"},
			Filterable:       collector.AllFilterable,
		}},
	},
	{
		"DATATYPE::SERVICEPERFDATA	TIMET::1441791005	HOSTNAME::xxx	SERVICEDESC::range	SERVICEPERFDATA::a used=4;2:;10:;1;4	SERVICECHECKCOMMAND::check_ranges!-w 3: -c 4: -g :46 -l :48 SERVICESTATE::0	SERVICESTATETYPE::1",
		[]PerformanceData{{
			Hostname:         "xxx",
			Service:          "range",
			Command:          "check_ranges",
			Time:             "1441791005000",
			PerformanceLabel: "a used",
			Unit:             "",
			Tags:             map[string]string{"warn-fill": "none", "crit-fill": "none"},
			Fields:           map[string]string{"value": "4.0", "warn": "2.0", "crit": "10.0", "min": "1.0", "max": "4.0"},
			Filterable:       collector.AllFilterable,
		}},
	},
	{
		"DATATYPE::SERVICEPERFDATA	TIMET::1441791006	HOSTNAME::xxx	SERVICEDESC::range	SERVICEPERFDATA::a used=4;:2;:10;1;4	SERVICECHECKCOMMAND::check_ranges!-w 3: -c 4: -g :46 -l :48 SERVICESTATE::0	SERVICESTATETYPE::1",
		[]PerformanceData{{
			Hostname:         "xxx",
			Service:          "range",
			Command:          "check_ranges",
			Time:             "1441791006000",
			PerformanceLabel: "a used",
			Unit:             "",
			Tags:             map[string]string{"warn-fill": "none", "crit-fill": "none"},
			Fields:           map[string]string{"value": "4.0", "warn": "2.0", "crit": "10.0", "min": "1.0", "max": "4.0"},
			Filterable:       collector.AllFilterable,
		}},
	},
	{
		"DATATYPE::SERVICEPERFDATA	TIMET::1441791007	HOSTNAME::xxx	SERVICEDESC::range	SERVICEPERFDATA::a used=4;~:2;10:~;1;4	SERVICECHECKCOMMAND::check_ranges!-w 3: -c 4: -g :46 -l :48 SERVICESTATE::0	SERVICESTATETYPE::1",
		[]PerformanceData{{
			Hostname:         "xxx",
			Service:          "range",
			Command:          "check_ranges",
			Time:             "1441791007000",
			PerformanceLabel: "a used",
			Unit:             "",
			Tags:             map[string]string{"warn-fill": "none", "crit-fill": "none"},
			Fields:           map[string]string{"value": "4.0", "warn": "2.0", "crit": "10.0", "min": "1.0", "max": "4.0"},
			Filterable:       collector.AllFilterable,
		}},
	},
	{
		// test dot separated data
		"DATATYPE::SERVICEPERFDATA	TIMET::1441791000	HOSTNAME::xxx	SERVICEDESC::range	SERVICEPERFDATA::a used=4.5	SERVICECHECKCOMMAND::check_ranges!-w 3: -c 4: -g :46 -l :48 SERVICESTATE::0	SERVICESTATETYPE::1",
		[]PerformanceData{{
			Hostname:         "xxx",
			Service:          "range",
			Command:          "check_ranges",
			Time:             "1441791000000",
			PerformanceLabel: "a used",
			Unit:             "",
			Tags:             map[string]string{},
			Fields:           map[string]string{"value": "4.5"},
			Filterable:       collector.AllFilterable,
		}},
	},
	{
		// test comma separated data
		"DATATYPE::SERVICEPERFDATA	TIMET::1441791000	HOSTNAME::xxx	SERVICEDESC::range	SERVICEPERFDATA::comma=4,5	SERVICECHECKCOMMAND::check_ranges!-w 3: -c 4: -g :46 -l :48 SERVICESTATE::0	SERVICESTATETYPE::1",
		[]PerformanceData{{
			Hostname:         "xxx",
			Service:          "range",
			Command:          "check_ranges",
			Time:             "1441791000000",
			PerformanceLabel: "comma",
			Unit:             "",
			Tags:             map[string]string{},
			Fields:           map[string]string{"value": "4.5"},
			Filterable:       collector.AllFilterable,
		}},
	},
	{
		// test comma separated data
		`DATATYPE::SERVICEPERFDATA	TIMET::1441791000	HOSTNAME::xxx	SERVICEDESC::range	SERVICEPERFDATA::a used=4,6 'C:\ used %'=44,1%;89,2;94,3;0,4;100,5	SERVICECHECKCOMMAND::check_ranges!-w 3: -c 4: -g :46 -l :48 SERVICESTATE::0	SERVICESTATETYPE::1`,
		[]PerformanceData{{
			Hostname:         "xxx",
			Service:          "range",
			Command:          "check_ranges",
			Time:             "1441791000000",
			PerformanceLabel: "a used",
			Unit:             "",
			Tags:             map[string]string{},
			Fields:           map[string]string{"value": "4.6"},
			Filterable:       collector.AllFilterable,
		}, {
			Hostname:         "xxx",
			Service:          "range",
			Command:          "check_ranges",
			Time:             "1441791000000",
			PerformanceLabel: `'C:\ used %'`,
			Unit:             "%",
			Tags:             map[string]string{"warn-fill": "none", "crit-fill": "none"},
			Fields:           map[string]string{"value": "44.1", "warn": "89.2", "crit": "94.3", "min": "0.4", "max": "100.5"},
			Filterable:       collector.AllFilterable,
		}},
	},
	{
		// test tag
		"DATATYPE::SERVICEPERFDATA	TIMET::1441791000	NAGFLUX:TAG::foo=bar	HOSTNAME::xxx	SERVICEDESC::range	SERVICEPERFDATA::tag=4.5	SERVICECHECKCOMMAND::check_ranges!-w 3: -c 4: -g :46 -l :48 SERVICESTATE::0	SERVICESTATETYPE::1",
		[]PerformanceData{{
			Hostname:         "xxx",
			Service:          "range",
			Command:          "check_ranges",
			Time:             "1441791000000",
			PerformanceLabel: "tag",
			Unit:             "",
			Tags:             map[string]string{"foo": "bar"},
			Fields:           map[string]string{"value": "4.5"},
			Filterable:       collector.AllFilterable,
		}},
	},
	{
		// test empty tag
		"DATATYPE::SERVICEPERFDATA	TIMET::1441791000	NAGFLUX:TAG::	HOSTNAME::xxx	SERVICEDESC::range	SERVICEPERFDATA::tag=4.5	SERVICECHECKCOMMAND::check_ranges!-w 3: -c 4: -g :46 -l :48 SERVICESTATE::0	SERVICESTATETYPE::1",
		[]PerformanceData{{
			Hostname:         "xxx",
			Service:          "range",
			Command:          "check_ranges",
			Time:             "1441791000000",
			PerformanceLabel: "tag",
			Unit:             "",
			Tags:             map[string]string{},
			Fields:           map[string]string{"value": "4.5"},
			Filterable:       collector.AllFilterable,
		}},
	},
	{
		// test malformed tag
		"DATATYPE::SERVICEPERFDATA	TIMET::1441791000	NAGFLUX:TAG::$_SERVICENAGFLUX_TAG$	HOSTNAME::xxx	SERVICEDESC::range	SERVICEPERFDATA::tag=4.5	SERVICECHECKCOMMAND::check_ranges!-w 3: -c 4: -g :46 -l :48 SERVICESTATE::0	SERVICESTATETYPE::1",
		[]PerformanceData{{
			Hostname:         "xxx",
			Service:          "range",
			Command:          "check_ranges",
			Time:             "1441791000000",
			PerformanceLabel: "tag",
			Unit:             "",
			Tags:             map[string]string{},
			Fields:           map[string]string{"value": "4.5"},
			Filterable:       collector.AllFilterable,
		}},
	},
	{
		// test filterable
		"DATATYPE::SERVICEPERFDATA	TIMET::1441791000	NAGFLUX:TARGET::foo	HOSTNAME::xxx	SERVICEDESC::range	SERVICEPERFDATA::tag=4.5	SERVICECHECKCOMMAND::check_ranges!-w 3: -c 4: -g :46 -l :48 SERVICESTATE::0	SERVICESTATETYPE::1",
		[]PerformanceData{{
			Hostname:         "xxx",
			Service:          "range",
			Command:          "check_ranges",
			Time:             "1441791000000",
			PerformanceLabel: "tag",
			Unit:             "",
			Tags:             map[string]string{},
			Fields:           map[string]string{"value": "4.5"},
			Filterable:       collector.Filterable{Filter: "foo"},
		}},
	},
	{
		// github https://github.com/Griesbacher/nagflux/issues/19#issuecomment-286799167
		"DATATYPE::SERVICEPERFDATA	TIMET::1489572014	HOSTNAME::HOST_SERVER	SERVICEDESC::web	SERVICEPERFDATA::time=0,004118s;;;0,000000 size=128766B;;;0	SERVICECHECKCOMMAND::check_http!HOST_SERVER!80!/!20	HOSTSTATE::UP	HOSTSTATETYPE::HARD SERVICESTATE::OK	SERVICESTATETYPE::HARD	SERVICEOUTPUT::HTTP OK: HTTP/1.1 200 OK - 128766 bytes in 0,004 second response time",
		[]PerformanceData{{
			Hostname:         "HOST_SERVER",
			Service:          "web",
			Command:          "check_http",
			Time:             "1489572014000",
			PerformanceLabel: "time",
			Unit:             "s",
			Tags:             map[string]string{},
			Fields:           map[string]string{"value": "0.004118", "min": "0.000000"},
			Filterable:       collector.AllFilterable,
		}, {
			Hostname:         "HOST_SERVER",
			Service:          "web",
			Command:          "check_http",
			Time:             "1489572014000",
			PerformanceLabel: "size",
			Unit:             "B",
			Tags:             map[string]string{},
			Fields:           map[string]string{"value": "128766.0", "min": "0.0"},
			Filterable:       collector.AllFilterable,
		}},
	},
	{
		// github https://github.com/Griesbacher/nagflux/issues/32
		"DATATYPE::SERVICEPERFDATA	TIMET::1490957788	HOSTNAME::müü	SERVICEDESC::möö	SERVICEPERFDATA::getItinerary_min=34385µs getItinerary_avg=130925µs getItinerary_max=267719µs	SERVICECHECKCOMMAND::check_perfs	SERVICESTATE::0	SERVICESTATETYPE::1",
		[]PerformanceData{{
			Hostname:         "müü",
			Service:          "möö",
			Command:          "check_perfs",
			Time:             "1490957788000",
			PerformanceLabel: "getItinerary_min",
			Unit:             "µs",
			Tags:             map[string]string{},
			Fields:           map[string]string{"value": "34385.0"},
			Filterable:       collector.AllFilterable,
		}, {
			Hostname:         "müü",
			Service:          "möö",
			Command:          "check_perfs",
			Time:             "1490957788000",
			PerformanceLabel: "getItinerary_avg",
			Unit:             "µs",
			Tags:             map[string]string{},
			Fields:           map[string]string{"value": "130925.0"},
			Filterable:       collector.AllFilterable,
		}, {
			Hostname:         "müü",
			Service:          "möö",
			Command:          "check_perfs",
			Time:             "1490957788000",
			PerformanceLabel: "getItinerary_max",
			Unit:             "µs",
			Tags:             map[string]string{},
			Fields:           map[string]string{"value": "267719.0"},
			Filterable:       collector.AllFilterable,
		}},
	},
	{
		"DATATYPE::SERVICEPERFDATA	TIMET::1490957788	HOSTNAME::test	SERVICEDESC::test space	SERVICEPERFDATA::'test rss'=35512320B;;;0;	SERVICECHECKCOMMAND::check_test	SERVICESTATE::0	SERVICESTATETYPE::1",
		[]PerformanceData{{
			Hostname:         "test",
			Service:          "test space",
			Command:          "check_test",
			Time:             "1490957788000",
			PerformanceLabel: "'test rss'",
			Unit:             "B",
			Tags:             map[string]string{},
			Fields:           map[string]string{"value": "35512320.0", "min": "0.0"},
			Filterable:       collector.AllFilterable,
		}},
	},
}

func TestNagiosSpoolfileWorker_PerformanceDataIterator(t *testing.T) {
	w := NewNagiosSpoolfileWorker(0, nil, nil, nil, 4096, collector.AllFilterable)
	for _, data := range TestPerformanceData {
		splittedPerformanceData := helper.StringToMap(data.input, "\t", "::")
		collectedPerfData := []PerformanceData{}
		for singlePerfdata := range w.PerformanceDataIterator(splittedPerformanceData) {
			collectedPerfData = append(collectedPerfData, singlePerfdata)
		}
		assert.Equalf(t, data.expected, collectedPerfData, "performance data matches")
	}
}
