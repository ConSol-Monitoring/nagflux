package livestatus

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/kdar/factorlog"
)

// Connector fetches data from livestatus.
type Connector struct {
	Log               *factorlog.FactorLog
	LivestatusAddress string
	ConnectionType    string
}

// Queries livestatus and returns an list of list outer list are lines inner elements within the line.
func (connector *Connector) connectToLivestatus(query string, result chan []string, outerFinish chan bool) {
	var conn net.Conn
	switch connector.ConnectionType {
	case "tcp":
		conn, _ = net.Dial("tcp", connector.LivestatusAddress)
	case "file":
		conn, _ = net.Dial("unix", connector.LivestatusAddress)
	default:
		connector.Log.Critical("Connection type is unknown, options are: tcp, file. Input:" + connector.ConnectionType)
		outerFinish <- false
		return
	}
	if conn == nil {
		outerFinish <- false
		return
	}

	connector.Log.Debugf("livestatus query: %s", query)

	defer conn.Close()
	fmt.Fprint(conn, query)
	reader := bufio.NewReader(conn)

	length := 1
	for length > 0 {
		message, _, err := reader.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			connector.Log.Warn(err)
		}
		length = len(message)
		if length > 0 {
			csvReader := csv.NewReader(strings.NewReader(string(message)))
			csvReader.Comma = ';'
			csvReader.LazyQuotes = true
			records, err := csvReader.Read()
			if err != nil {
				connector.Log.Warn("Query failed while csv parsing:" + query)
				connector.Log.Warn(string(message))
				connector.Log.Warn(err)
			}
			result <- records
		}
	}
	outerFinish <- true
}

func (connector *Connector) buildQuery(baseQuery string, filter []string) string {
	if len(filter) == 0 {
		return baseQuery
	}

	filterStr := strings.Builder{}
	for _, str := range filter {
		str = strings.TrimSpace(str)
		str = strings.ReplaceAll(str, `\\n`, "\n")

		filterStr.WriteString("\n")
		filterStr.WriteString(str)
	}

	return strings.TrimSpace(baseQuery) + "\n" + strings.TrimSpace(filterStr.String()) + "\n\n"
}
