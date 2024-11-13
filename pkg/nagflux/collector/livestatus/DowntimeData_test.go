package livestatus

import (
	"testing"

	"pkg/nagflux/config"
	"pkg/nagflux/logging"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeValuesDowntime(t *testing.T) {
	t.Parallel()
	down := &DowntimeData{Data: Data{hostName: "host 1", serviceDisplayName: "service 1", author: "philip"}, endTime: "123"}
	down.sanitizeValues()
	assert.Equalf(t, `host\ 1`, down.Data.hostName, "The notificationType should be escaped.")
}

func TestPrintInfluxdbDowntime(t *testing.T) {
	logging.InitTestLogger()
	down := &DowntimeData{Data: Data{hostName: "host 1", serviceDisplayName: "service 1", author: "philip"}, endTime: "123"}
	if !didThisPanic(down.PrintForInfluxDB, "0.8") {
		t.Errorf("This should panic, due to unsuported influxdb version")
	}

	result := down.PrintForInfluxDB("0.9")
	expected := `messages,host=host\ 1,service=service\ 1,type=downtime,author=philip message="Downtime start: <br>" 000
messages,host=host\ 1,service=service\ 1,type=downtime,author=philip message="Downtime end: <br>" 123000`
	assert.Equalf(t, expected, result, "The result did not match the expected")
}

func TestPrintElasticsearchDowntime(t *testing.T) {
	logging.InitTestLogger()
	config.InitConfigFromString(Config)
	down := &DowntimeData{Data: Data{hostName: "host 1", serviceDisplayName: "service 1", author: "philip", entryTime: "1458988932000"}, endTime: "123"}
	if !didThatPanic(down.PrintForElasticsearch, "1.0", "index") {
		t.Errorf("This should panic, due to unsuported elasticsearch version")
	}

	result := down.PrintForElasticsearch("2.0", "index")
	expected := `{"index":{"_index":"index-2016.03","_type":"messages"}}
{"timestamp":1458988932000000,"message":"Downtime start: <br>","author":"philip","host":"host 1","service":"service 1","type":"downtime"}

{"index":{"_index":"index-1970.01","_type":"messages"}}
{"timestamp":123000,"message":"Downtime end: <br>","author":"philip","host":"host 1","service":"service 1","type":"downtime"}
`
	assert.Equalf(t, expected, result, "The result did not match the expected")
}
