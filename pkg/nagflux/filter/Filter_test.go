package filter

import (
	"testing"

	"pkg/nagflux/config"
	"pkg/nagflux/logging"

	"github.com/stretchr/testify/assert"
)

const configFileContent = `
[Filter]
    SpoolFileLineTerms = "check_date|check_service|test-check-host-alive-parent|test-check-host-alive"
    SpoolFileLineTerms = "HOSTPERFDATA::"
`

const (
	spoolFilesLine  = `DATATYPE::HOSTPERFDATA  TIMET::1749111379       HOSTNAME::dev_host_098 HOSTPERFDATA::rta=0.014ms;3000.000;5000.000;0; rtmax=0.045ms;;;; rtmin=0.000ms;;;; pl=0%;80;100;0;100 HOSTCHECKCOMMAND::check-host-alive      HOSTSTATE::UP   HOSTSTATETYPE::HARD`
	spoolFilesLine2 = "DATATYPE::HOSTPERFDATA	TIMET::1750856135	HOSTNAME::dev_host_035	HOSTPERFDATA::	HOSTCHECKCOMMAND::test-check-host-alive-parent!random!$HOSTSTATE:dev_router_02$	HOSTSTATE::UP	HOSTSTATETYPE::HARD"
)

func TestFilterLine(t *testing.T) {
	config.InitConfigFromString(configFileContent)
	logging.InitTestLogger()
	config := config.GetConfig()
	filter := NewFilter(config.Filter.SpoolFileLineTerms)

	ok := filter.TestLine([]byte(spoolFilesLine))
	assert.True(t, ok, "Line should be ok but wasn't")

	ok = filter.TestLine([]byte(spoolFilesLine2))
	assert.True(t, ok, "Line should be ok but wasn't")
}
