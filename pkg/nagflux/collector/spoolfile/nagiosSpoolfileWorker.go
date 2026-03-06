package spoolfile

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"pkg/nagflux/collector"
	"pkg/nagflux/collector/livestatus"
	"pkg/nagflux/config"
	"pkg/nagflux/filter"
	"pkg/nagflux/helper"
	"pkg/nagflux/logging"
	"pkg/nagflux/statistics"

	"github.com/kdar/factorlog"
)

const (
	nagfluxTags   string = "NAGFLUX:TAG"
	nagfluxField  string = "NAGFLUX:FIELD"
	nagfluxTarget string = "NAGFLUX:TARGET"

	hostPerfdata string = "HOSTPERFDATA"

	servicePerfdata string = "SERVICEPERFDATA"

	hostType    string = "HOST"
	serviceType string = "SERVICE"

	hostname     string = "HOSTNAME"
	timet        string = "TIMET"
	checkcommand string = "CHECKCOMMAND"
	servicedesc  string = "SERVICEDESC"
)

var (
	// Start at the line, match anything that has two consecutive columns.
	// First capture group is before the colon and the second is after the column.
	checkMulitRegex = regexp.MustCompile(`^(.*::)(.*)`)

	// Digit, point or dash in a group, repeat this once or more
	// This idea is to get how many numbers, possibly negative, are in a string
	rangeRegex = regexp.MustCompile(`[\d\.\-]+`)

	// https://regex101.com/r/wNeesp/1
	// Read this as well if you are new to monitoring plugins output.
	// https://www.monitoring-plugins.org/doc/guidelines.html#AEN197
	// https://www.monitoring-plugins.org/doc/guidelines.html#THRESHOLDFORMAT
	//	1st Capture Group: ([^=]+)
	//		Anything without an '=' sign , at least one repeating
	//		Idea: It captures the name of the perfdata
	//	A literal '='
	//		Required in a perfdata string, helps to capture anything before it as the name
	//	2nd Capture Group: (U|[\d\.\,\-]+)
	//		There are two options here
	//			1. Either a literal 'U'.
	//				Means "Unknown". This is used by plugins to indicate that it could not get performance data for some reason.
	// 				Writing 'U' is better than not including it, which would mean its unavailable.
	//			2. [\d\.,\-]+
	//				'\d' is for digits, '\.' is a literal point, '\,' is a literal comma, '\-' is a literal dash i.e negative sign. Repat any of these one or more times.
	//				Idea: This captures the current value of the perfdata, which should be some kind of number, possibly negative as well. Examples: '123' '24.5' '-54,2'
	//	3rd Capture Group: ([\pL\/\%]*)
	//		'\pL' matches any kind of letter in any language. '\/' matches a literal forward slash, '\%' matches a literal percentage sign. Repeat any of it zero or more times.
	//		Idea: This captures the Unit of Measurement for the current value.
	// 		Might be empty since the raw value might be enough. Forward slash is used for rate. Examples: '' 's' 'ms' 'B' 'KB' '%' 'B/s' '/s'
	//	A literal ';'. Zero or Once. Semicolons are used as seperators between different fields in a perfdata.
	// 	Idea: Perfdata might only have the current value, and does not report, warning, critical, min, max and other values.
	//	4rd Capture Group: ([\d\.\,\-\:\~\@]*)
	//		Literal digits, point, comma, dash, colon, tilde, at sign. Colon and at sign is used in range definitions. Tilde is used when specifying ranges for negative infinity
	//		Repeat this capture group zero or one time.
	//		Idea: These are the threshold definitions for warning threshold. It does not have to be set
	//	A literal ';'. Zero or Once. Semicolons are used as seperators between different fields in a perfdata.
	//	5th Capture Group: ([\d\.\,\-\:\~\@]*)
	//		Same as 4th capture group, but it is for critical threshold this time
	//	A literal ';'. Zero or Once. Semicolons are used as seperators between different fields in a perfdata.
	//	6th Capture Group: ([\d\.\,\-]*)
	//		Similar to the 2nd capture group, but it might be repeated zero or more times instead, as this field may be empty
	//		Idea: Used for the min value so far. It does not need an Unit of Measurement or might be specified as a range like a threshold. Therefore it is simpler
	//	A literal ';'. Zero or Once. Semicolons are used as seperators between different fields in a perfdata.
	//	7th Capture Group: ([\d\.\,\-]*)
	//		Same as 6th capture group, but it is for the maximum value this time.
	// '\s' matches any whitespace character, infinite times.
	//	Idea: It seperates different perf data values as this must be matched new capture group can be captured
	regexPerformancelable = regexp.MustCompile(`([^=]+)=(U|[\d\.\,\-]+)([\pL\/\%]*);?([\d\.\,\-\:\~\@]*)?;?([\d\.\,\-\:\~\@]*)?;?([\d\.\,\-]*)?;?([\d\.\,\-]*)?;?\s*`)

	// The perfdata part might have some alternative check at the end, recognize it by it being at the end and only containing letters, '_', '-'
	// The check name will be in the capture group.
	// This convention is not found in monitoring-plugins development guidelines
	regexAlternativeCommand = regexp.MustCompile(`.*\[([a-zA-Z\_\-\.\ ]+)\]\s?$`)

	// The perfdata part might report errors for different data
	// it has to put them in square brackets first, and use an equal sign for the error
	// This is to differenciate it from alternative command, which does not have an equal sign.
	// This convention is not found in monitoring-plugins development guidelines
	regexStripErrors = regexp.MustCompile(`\[[^\]]*=[^\]]*\]`)
)

var (
	log *factorlog.FactorLog = logging.GetLogger()
)

// NagiosSpoolfileWorker parses the given spoolfiles and adds the extraced perfdata to the queue.
type NagiosSpoolfileWorker struct {
	workerID                       int
	quit                           chan bool
	jobs                           chan string
	results                        collector.ResultQueues
	livestatusCacheBuilder         *livestatus.CacheBuilder
	fileBufferSize                 int
	defaultTarget                  collector.Filterable
	filterProcessor                filter.Processor
	perfdataLabelMaxSize           int
	perfdataUOMMaxLength           int
	perfdataNumericValuesMaxLength int
	perfdataThresholdsMaxLength    int
}

// NewNagiosSpoolfileWorker returns a new NagiosSpoolfileWorker.
func NewNagiosSpoolfileWorker(workerID int, jobs chan string, results collector.ResultQueues,
	livestatusCacheBuilder *livestatus.CacheBuilder, fileBufferSize int, defaultTarget collector.Filterable, perfdataLabelMaxLength int, perfdataUOMMaxLength int, perfdataNumericValuesMaxLength int, perfdataThresholdsMaxLength int,
) *NagiosSpoolfileWorker {
	cfg := config.GetConfig()
	return &NagiosSpoolfileWorker{
		workerID:                       workerID,
		quit:                           make(chan bool),
		jobs:                           jobs,
		results:                        results,
		livestatusCacheBuilder:         livestatusCacheBuilder,
		fileBufferSize:                 fileBufferSize,
		defaultTarget:                  defaultTarget,
		filterProcessor:                filter.NewFilter(cfg.Filter.SpoolFileLineTerms),
		perfdataLabelMaxSize:           perfdataLabelMaxLength,
		perfdataUOMMaxLength:           perfdataUOMMaxLength,
		perfdataNumericValuesMaxLength: perfdataNumericValuesMaxLength,
		perfdataThresholdsMaxLength:    perfdataThresholdsMaxLength,
	}
}

// NagiosSpoolfileWorkerGenerator generates a worker and starts it.
func NagiosSpoolfileWorkerGenerator(jobs chan string, results collector.ResultQueues,
	livestatusCacheBuilder *livestatus.CacheBuilder, fileBufferSize int, defaultTarget collector.Filterable, perfdataLabelMaxLength int, perfdataUOMMaxLength int, perfdataNumericValuesMaxLength int, perfdataThresholdsMaxLength int,
) func() *NagiosSpoolfileWorker {
	workerID := 0
	return func() *NagiosSpoolfileWorker {
		s := NewNagiosSpoolfileWorker(workerID, jobs, results, livestatusCacheBuilder, fileBufferSize, defaultTarget, perfdataLabelMaxLength, perfdataUOMMaxLength, perfdataNumericValuesMaxLength, perfdataThresholdsMaxLength)
		workerID++
		go s.run()
		return s
	}
}

// Stop stops the worker
func (w *NagiosSpoolfileWorker) Stop() {
	w.quit <- true
	<-w.quit
	log.Debug("SpoolfileWorker stopped")
}

// Waits for files to parse and sends the data to the main queue.
func (w *NagiosSpoolfileWorker) run() {
	promServer := statistics.GetPrometheusServer()
	var file string
	for {
		select {
		case <-w.quit:
			w.quit <- true
			return
		case file = <-w.jobs:
			promServer.SpoolFilesInQueue.Set(float64(len(w.jobs)))
			startTime := time.Now()
			log.Debug("Reading file: ", file)

			filehandle, err := os.OpenFile(file, os.O_RDONLY, os.ModePerm)
			if err != nil {
				log.Warn("NagiosSpoolfileWorker: Opening file error: ", err)
				break
			}
			reader := bufio.NewReaderSize(filehandle, w.fileBufferSize)
			queries := 0
			line, isPrefix, err := reader.ReadLine()
			for err == nil && !isPrefix {
				splittedPerformanceData := helper.StringToMap(string(line), "\t", "::")
				if skipLine := w.filterProcessor.TestLine(line); !skipLine {
					log.Debugf("skipping line %s", string(line))

					line, isPrefix, err = reader.ReadLine()
					continue
				}
				for singlePerfdata := range w.PerformanceDataIterator(splittedPerformanceData) {
					for _, r := range w.results {
						select {
						case <-w.quit:
							w.quit <- true
							return
						case r <- singlePerfdata:
							queries++
						case <-time.After(time.Duration(10) * time.Second):
							log.Warn("NagiosSpoolfileWorker: Could not write to buffer")
						}
					}
				}
				line, isPrefix, err = reader.ReadLine()
			}
			if err != nil && err != io.EOF {
				log.Warn(err)
			}
			if isPrefix {
				log.Warn("NagiosSpoolfileWorker: filebuffer is too small")
			}
			filehandle.Close()
			err = os.Remove(file)
			if err != nil {
				log.Warn(err)
			}
			timeDiff := float64(time.Since(startTime).Nanoseconds() / 1000000)
			if timeDiff >= 0 {
				promServer.SpoolFilesParsedDuration.Add(timeDiff)
			}
			if queries >= 0 {
				promServer.SpoolFilesLines.Add(float64(queries))
			}
		case <-time.After(time.Duration(5) * time.Minute):
			log.Debug("NagiosSpoolfileWorker: Got nothing to do")
		}
	}
}

// PerformanceDataIterator returns an iterator to loop over generated perf data.
func (w *NagiosSpoolfileWorker) PerformanceDataIterator(input map[string]string) <-chan *PerformanceData {
	ch := make(chan *PerformanceData)
	dataType := findDataType(input)
	if dataType == "" {
		if len(input) > 1 {
			log.Info("Line does not match the scheme: ", input)
		}
		close(ch)
		return ch
	}

	currentCommand := w.searchAlternativeCommand(input[dataType+"PERFDATA"], input[dataType+checkcommand])
	currentTime := helper.CastStringTimeFromSToMs(input[timet])
	currentService := ""

	if dataType != hostType {
		currentService = input[servicedesc]
	}

	// anonymous closure, starts immediately after definition in another goroutine without blocking
	go func() {
		perdataString := input[dataType+"PERFDATA"]
		cleaned := regexStripErrors.ReplaceAllString(perdataString, "")

		// Slices up the string into a form like this
		// Each match is put into an array with their capture groups
		// These arrays are put into another array
		// Here is an example:
		/*
			[][]string len: 4, cap: 10, [
				["rta=0.024ms;3000.000;5000.000;0; ","rta","0.024","ms","3000.000","5000.000","0",""],
				["rtmax=0.085ms;;;; ","rtmax","0.085","ms","","","",""],
				["rtmin=0.000ms;;;; ","rtmin","0.000","ms","","","",""],
				["pl=0%;80;100;0;100","pl","0","%","80","100","0","100"]
			]
		*/
		perfdataStringMatches := regexPerformancelable.FindAllStringSubmatch(cleaned, -1)
		currentCheckMultiLabel := ""

		// try to find a check_multi prefix
		if len(perfdataStringMatches) > 0 && len(perfdataStringMatches[0]) > 1 {
			currentCheckMultiLabel = getCheckMultiRegexMatch(perfdataStringMatches[0][1])
		}

	perfdataStringMatchLoop:
		for _, perfdataStringMatch := range perfdataStringMatches {
			// Allows to add tags and fields to spoolfileentries
			tags := map[string]string{}
			if tagString, ok := input[nagfluxTags]; ok {
				tags = helper.StringToMap(tagString, " ", "=")
			}
			field := map[string]string{}
			if fieldString, ok := input[nagfluxField]; ok {
				field = helper.StringToMap(fieldString, " ", "=")
			}
			var target collector.Filterable
			if targetString, ok := input[nagfluxTarget]; ok {
				target = collector.Filterable{Filter: targetString}
			} else {
				target = collector.AllFilterable
			}

			perf := &PerformanceData{
				Hostname:         input[hostname],
				Service:          currentService,
				Command:          currentCommand,
				Time:             currentTime,
				PerformanceLabel: perfdataStringMatch[PerformanceDataSliceFields(Label)],
				Unit:             perfdataStringMatch[PerformanceDataSliceFields(UOM)],
				Tags:             tags,
				Fields:           field,
				Filterable:       target,
			}

			if currentCheckMultiLabel != "" {
				// if an check_multi prefix was found last time
				// test if the current one has also one
				if potentialNextOne := getCheckMultiRegexMatch(perf.PerformanceLabel); potentialNextOne == "" {
					// if not put the last one in front the current
					perf.PerformanceLabel = currentCheckMultiLabel + perf.PerformanceLabel
				} else {
					// else remember the current prefix for the next one
					currentCheckMultiLabel = potentialNextOne
				}
			}

			// perfdataStringMatch might not have all fields like perfdataStringMatch[Crit] available,
			// iterate each field until the end
			for i, data := range perfdataStringMatch {
				fieldType, err := indexToPerformanceDataSliceField(i)
				if err != nil {
					log.Warnf("Error when converting the index to a known field in performance data : %s", err.Error())
					continue
				}

				switch fieldType {
				case RawMatch:
					continue
				case Label:
					if len(data) > w.perfdataLabelMaxSize {
						log.Warnf("Perfdata Label: '%s' is too long with length: %s and longer than the limit: %s. Probably an anomally. Skipping this perfdata item, Host: %v , Service: %v, Perfdata fields: %v", data, len(data), w.perfdataLabelMaxSize, perf.Hostname, perf.Service, perfdataStringMatch)
						goto perfdataStringMatchLoop
					}
					continue
				case UOM:
					if len(data) > w.perfdataUOMMaxLength {
						log.Warnf("Perfdata UOM: '%s' is too long with length: %s and longer than the limit: %s. Probably an anomally. Host: %v , Service: %v, Perfdata fields: %v", data, len(data), w.perfdataUOMMaxLength, perf.Hostname, perf.Service, perfdataStringMatch)
						goto perfdataStringMatchLoop
					}
					continue
				case Value, Min, Max:
					if len(data) > w.perfdataNumericValuesMaxLength {
						log.Warnf("Perfdata field %s: '%s' is too long with length: %s and longer than the limit: %s. Probably an anomally. Host: %v , Service: %v, Perfdata fields: %v", fieldType.String(), data, len(data), w.perfdataNumericValuesMaxLength, perf.Hostname, perf.Service, perfdataStringMatch)
						goto perfdataStringMatchLoop
					}
					continue
				case Warn, Crit:
					if len(data) > w.perfdataThresholdsMaxLength {
						log.Warnf("Perfdata field %s: '%s' is too long with length: %s and longer than the limit: %s. Probably an anomally. Host: %v , Service: %v, Perfdata fields: %v", fieldType.String(), data, len(data), w.perfdataThresholdsMaxLength, perf.Hostname, perf.Service, perfdataStringMatch)
						goto perfdataStringMatchLoop
					}
					continue
				}

				if data == "" {
					continue
				}

				// Anything after here is a number or a range
				// Convert all commas to points, to help in the integer/float parsing
				data = strings.ReplaceAll(data, ",", ".")

				// Add downtime tag if needed
				if fieldType == PerformanceDataSliceFields(Value) && w.livestatusCacheBuilder != nil && w.livestatusCacheBuilder.IsServiceInDowntime(perf.Hostname, perf.Service, input[timet]) {
					perf.Tags["downtime"] = "true"
				}

				switch fieldType {
				case Warn, Crit:
					// Range handling
					fillLabel := fieldType.String() + "-fill"
					// find how many numbers are there in the string, if there are two it is a range
					rangeHits := rangeRegex.FindAllStringSubmatch(data, -1)
					if len(rangeHits) == 1 {
						perf.Tags[fillLabel] = "none"
						perf.Fields[fieldType.String()] = helper.StringIntToStringFloat(rangeHits[0][0])
					} else if len(rangeHits) == 2 {
						// If there is a range with no infinity as border, create two points
						if strings.Contains(data, "@") {
							perf.Tags[fillLabel] = "inner"
						} else {
							perf.Tags[fillLabel] = "outer"
						}

						for i, tag := range []string{"min", "max"} {
							tagKey := fmt.Sprintf("%s-%s", rangeRegex.String, tag)
							perf.Fields[tagKey] = helper.StringIntToStringFloat(rangeHits[i][0])
						}
					} else {
						log.Warnf("String: '%s' in field '%s' could not be parsed. Host: %v, Service: %v, Perf Data Fields: %v", data, fieldType.String(), perf.Hostname, perf.Service, perfdataStringMatch)
					}
				case Value, Min, Max:
					if data == "U" {
						perf.Fields["unknown"] = "true"
						continue
					}
					if !helper.IsStringANumber(data) {
						log.Warnf("String: '%s' in field '%s' is not a number, should be one. Host: %v, Service: %v, Perf Data Fields: %v", data, fieldType.String(), perf.Hostname, perf.Service, perfdataStringMatch)
						continue perfdataStringMatchLoop
					}
					perf.Fields[fieldType.String()] = helper.StringIntToStringFloat(data)
				}
			}
			ch <- perf
		}
		close(ch)
	}()

	return ch
}

func getCheckMultiRegexMatch(perfDataValueName string) string {
	regexResult := checkMulitRegex.FindAllStringSubmatch(perfDataValueName, -1)
	if len(regexResult) == 1 && len(regexResult[0]) == 3 {
		return regexResult[0][1]
	}
	return ""
}

func findDataType(input map[string]string) string {
	var typ string
	if isHostPerformanceData(input) {
		typ = hostType
	} else if isServicePerformanceData(input) {
		typ = serviceType
	}
	return typ
}

// searchAlternativeCommand looks for alternative command name in perfdata
func (w *NagiosSpoolfileWorker) searchAlternativeCommand(perfData, command string) string {
	result := command
	search := regexAlternativeCommand.FindAllStringSubmatch(perfData, 1)
	if len(search) == 1 && len(search[0]) == 2 {
		result = search[0][1]
	}
	return splitCommandInput(result)
}

// Cuts the command at the first !.
func splitCommandInput(command string) string {
	return strings.Split(command, "!")[0]
}

// Tests if perfdata is of type hostperfdata.
func isHostPerformanceData(input map[string]string) bool {
	return input["DATATYPE"] == hostPerfdata
}

// Tests if perfdata is of type serviceperfdata.
func isServicePerformanceData(input map[string]string) bool {
	return input["DATATYPE"] == servicePerfdata
}

type PerformanceDataSliceFields int

const (
	RawMatch PerformanceDataSliceFields = iota
	Label
	Value
	UOM
	Warn
	Crit
	Min
	Max
)

// Converts the index of the sliced perftype to an string.
func indexToPerformanceDataSliceField(index int) (PerformanceDataSliceFields, error) {
	switch index {
	case 0:
		return RawMatch, nil
	case 1:
		return Label, nil
	case 2:
		return Value, nil
	case 4:
		return Warn, nil
	case 5:
		return Crit, nil
	case 6:
		return Min, nil
	case 7:
		return Max, nil
	default:
		return 0, errors.New("Illegal index: " + strconv.Itoa(index))
	}
}

// String returns the string representation of a PerformanceType
func (pt PerformanceDataSliceFields) String() string {
	switch pt {
	case RawMatch:
		return "rawMatch"
	case Label:
		return "name"
	case Value:
		return "value"
	case Warn:
		return "warn"
	case Crit:
		return "crit"
	case Min:
		return "min"
	case Max:
		return "max"
	default:
		return ""
	}
}
