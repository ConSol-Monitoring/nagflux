package filter

import (
	"regexp"
	"strings"
	"sync"

	"pkg/nagflux/logging"
)

type Processor struct {
	lineMut       *sync.RWMutex
	lineFilterReg []*regexp.Regexp
}

func NewFilter(filterTerms []string) Processor {
	lineFilterReg := make([]*regexp.Regexp, 0, 10)
	logging.GetLogger().Debug("Try to compile these terms: %v", filterTerms)
	fp := Processor{
		lineMut:       &sync.RWMutex{},
		lineFilterReg: lineFilterReg,
	}
	fp.lineMut.Lock()
	for _, term := range filterTerms {
		logging.GetLogger().Printf("Term is: %s", term)
		reg, err := regexp.Compile(term)
		if err != nil {
			logging.GetLogger().Warnf("could not compile your filter: %s", term)
			continue
		}
		fp.lineFilterReg = append(fp.lineFilterReg, reg)
	}
	fp.lineMut.Unlock()

	return fp
}

func (f Processor) FilterNagiosSpoolFileLine(line []byte) bool {
	return f.TestLine(line)
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
			return true
		}
	}
	f.lineMut.RUnlock()
	return false
}
