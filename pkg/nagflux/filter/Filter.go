package filter

import (
	"pkg/nagflux/config"
	"pkg/nagflux/logging"
	"regexp"
	"strings"
	"sync"
)

type Processor struct {
	lineMut        *sync.RWMutex
	lineFilterReg  []*regexp.Regexp
	fieldFilterReg map[string]*regexp.Regexp
}

func NewFilter() Processor {
	// TODO: should not compile regex for each call to NewFilter
	cfg := config.GetConfig()
	lineFilterReg := make([]*regexp.Regexp, 0, 10)
	fieldFilterReg := map[string]*regexp.Regexp{}
	logging.GetLogger().Info("Try to compile these terms: %v", cfg.LineFilter.SpoolFileLineTerms)
	fp := Processor{
		lineMut:        &sync.RWMutex{},
		lineFilterReg:  lineFilterReg,
		fieldFilterReg: fieldFilterReg,
	}
	fp.lineMut.Lock()
	for _, term := range cfg.LineFilter.SpoolFileLineTerms {
		logging.GetLogger().Printf("Term is: %s", term)
		reg, err := regexp.Compile(term)
		if err != nil {
			logging.GetLogger().Warnf("could not compile your filter: %s", term)
			continue
		}
		fp.lineFilterReg = append(fp.lineFilterReg, reg)
	}
	fp.lineMut.Unlock()

	for fieldName, terms := range cfg.FieldFilter {
		for _, term := range terms.Term {
			compiledTerm, err := regexp.Compile(term)
			if err != nil {
				logging.GetLogger().Warnf("Could not compile your field filter: %s", term)
				continue
			}
			fp.fieldFilterReg[fieldName] = compiledTerm
		}
	}
	return fp
}

func (f Processor) FilterNagiosSpoolFileLineAndFields(line []byte, splittedPerformanceData map[string]string) bool {
	return f.TestLine(line) || f.FilterPerformanceData(splittedPerformanceData)
}

func (f Processor) FilterLiveStatusLine(line []string) bool {
	return f.TestLine([]byte(strings.Join(line, " ")))
}

func (f Processor) TestLine(line []byte) bool {
	f.lineMut.RLock()
	if len(f.lineFilterReg) < 1 {
		return true
	}
	for _, term := range f.lineFilterReg {
		match := term.Match(line)
		if match {
			//			logging.GetLogger().Infof("The line %s, matched the term %s", string(line), term.String())
			return true
		}
	}
	f.lineMut.RUnlock()
	return false
}

func (f Processor) FilterPerformanceData(splittedPerformanceData map[string]string) bool {
	var anyMatch bool
	if len(f.fieldFilterReg) < 1 {
		return true
	}
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
