package filter

import (
	"fmt"
	"pkg/nagflux/config"
	"pkg/nagflux/helper"
	"pkg/nagflux/logging"
	"testing"

	"github.com/stretchr/testify/assert"
)

const configFileContent = `
[LineFilter]
    SpoolFileLineTerms = "check_date|check_service|test-check-host-alive-parent|test-check-host-alive"

[FieldFilter "HOSTNAME"]
	Term = "test*"

[FieldFilter "DATATYPE"]
	Term = "HOSTPERFDATA"

`

const (
	spoolFilesLine  = `DATATYPE::HOSTPERFDATA  TIMET::1749111379       HOSTNAME::dev_host_098 HOSTPERFDATA::rta=0.014ms;3000.000;5000.000;0; rtmax=0.045ms;;;; rtmin=0.000ms;;;; pl=0%;80;100;0;100 HOSTCHECKCOMMAND::check-host-alive      HOSTSTATE::UP   HOSTSTATETYPE::HARD`
	spoolFilesLine2 = "DATATYPE::HOSTPERFDATA	TIMET::1750856135	HOSTNAME::dev_host_035	HOSTPERFDATA::	HOSTCHECKCOMMAND::test-check-host-alive-parent!random!$HOSTSTATE:dev_router_02$	HOSTSTATE::UP	HOSTSTATETYPE::HARD"
)

func TestFilterLine(t *testing.T) {
	config.InitConfigFromString(configFileContent)
	logging.InitTestLogger()
	config := config.GetConfig()
	fmt.Printf("config.GetConfig(): %v\n", config)
	filter := NewFilter()

	ok := filter.TestLine([]byte(spoolFilesLine))
	assert.True(t, ok, "Line should be ok but wasn't")
}

func TestFilterLineRL(t *testing.T) {
	config.InitConfigFromString(configFileContent)
	logging.InitTestLogger()
	filter := NewFilter()
	ok := filter.TestLine([]byte(spoolFilesLine2))

	assert.True(t, ok, "Line should be ok but wasn't")
}

func TestFilterField(t *testing.T) {
	config.InitConfigFromString(configFileContent)
	logging.InitTestLogger()

	filter := NewFilter()
	splittedPerformanceData := helper.StringToMap(string(spoolFilesLine), "\t", "::")
	ok := filter.FilterPerformanceData(splittedPerformanceData)

	assert.True(t, ok, "Line should be ok but wasn't")
}
