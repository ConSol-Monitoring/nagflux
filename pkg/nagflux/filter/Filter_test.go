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
	Term = "check-host-alive"

[FieldFilter "HOSTNAME"]
	Term = "test*"

[FieldFilter "DATATYPE"]
	Term = "HOSTPERFDATA"

`
const SpoolFilesLine = `DATATYPE::HOSTPERFDATA  TIMET::1749111379       HOSTNAME::testhost_1008 HOSTPERFDATA::rta=0.014ms;3000.000;5000.000;0; rtmax=0.045ms;;;; rtmin=0.000ms;;;; pl=0%;80;100;0;100 HOSTCHECKCOMMAND::check-host-alive      HOSTSTATE::UP   HOSTSTATETYPE::HARD`

func TestFilterLine(t *testing.T) {
	config.InitConfigFromString(configFileContent)
	logging.InitTestLogger()
	config := config.GetConfig()
	fmt.Printf("config.GetConfig(): %v\n", config)
	filter := NewFilter()

	ok := filter.testLine([]byte(SpoolFilesLine))
	assert.True(t, ok, "Line should be ok but wasn't")
}

func TestFilterField(t *testing.T) {
	config.InitConfigFromString(configFileContent)
	logging.InitTestLogger()

	filter := NewFilter()
	splittedPerformanceData := helper.StringToMap(string(SpoolFilesLine), "\t", "::")
	ok := filter.testFields(splittedPerformanceData)

	assert.True(t, ok, "Line should be ok but wasn't")

}
