package filter

import (
	"pkg/nagflux/config"
	"pkg/nagflux/logging"
	"regexp"
)

type FilterProcessor struct {
	lineFilterReg  []*regexp.Regexp
	fieldFilterReg map[string]*regexp.Regexp
}

func NewFilter() FilterProcessor {
	logg := logging.GetLogger()
	cfg := config.GetConfig()
	lineFilterReg := make([]*regexp.Regexp, 0, 10)
	for _, term := range cfg.LineFilter.Term {
		logg.Printf("Term is: %s", term)
		reg, err := regexp.Compile(*term)
		if err != nil {
			logging.GetLogger().Warnf("could not compile your filter: %s", term)
			continue
		}
		lineFilterReg = append(lineFilterReg, reg)
	}

	fieldFilterReg := map[string]*regexp.Regexp{}
	for fieldName, term := range cfg.FieldFilter {
		compiledTerm, err := regexp.Compile(term.Term)
		if err != nil {
			logging.GetLogger().Warnf("Could not compile your field filter: %s", term.Term)
			continue
		}
		fieldFilterReg[fieldName] = compiledTerm
	}
	return FilterProcessor{
		lineFilterReg:  lineFilterReg,
		fieldFilterReg: fieldFilterReg,
	}
}

func (f FilterProcessor) FilterNagiosSpoolFileLineAndFields(line []byte, splittedPerformanceData map[string]string) bool {
	return f.testLine(line) || f.testFields(splittedPerformanceData)
}

func (f FilterProcessor) testLine(line []byte) bool {
	var anyMatch bool
	for _, term := range f.lineFilterReg {
		match := term.Match(line)
		anyMatch = anyMatch || match
	}
	return anyMatch
}

func (f FilterProcessor) testFields(splittedPerformanceData map[string]string) bool {
	var anyMatch bool
	for fieldName, term := range f.fieldFilterReg {
		field, ok := splittedPerformanceData[fieldName]
		if !ok {
			logging.GetLogger().Warnf("no field in performance data for filtername %s", fieldName)
			continue
		}
		match := term.MatchString(field)
		anyMatch = anyMatch || match
	}
	return anyMatch
}
