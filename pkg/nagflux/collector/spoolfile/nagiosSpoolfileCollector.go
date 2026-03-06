package spoolfile

import (
	"fmt"
	"os"
	"path"
	"time"

	"github.com/ConSol-Monitoring/nagflux/pkg/nagflux/collector"
	"github.com/ConSol-Monitoring/nagflux/pkg/nagflux/collector/livestatus"
	"github.com/ConSol-Monitoring/nagflux/pkg/nagflux/config"
	"github.com/ConSol-Monitoring/nagflux/pkg/nagflux/helper"
	"github.com/ConSol-Monitoring/nagflux/pkg/nagflux/logging"
	"github.com/ConSol-Monitoring/nagflux/pkg/nagflux/statistics"
)

const (
	// MinFileAge is the duration to wait, before the files are parsed
	MinFileAge = 3 * time.Second

	// IntervalToCheckDirectory the interval to check if there are new files
	IntervalToCheckDirectory = 1500 * time.Millisecond

	// If a perfdata label is longer than this, it will be logged as an anomaly and skipped
	PerfdataLabelMaxLengthDefault = int(32)

	// If a perfdata UOM is longer than this, it will be logged as an anomaly and skipped
	PerfdataUOMMaxLengthDefault = int(16)

	// If a perfdata numeric value, e.g current value, min, and max is longer than this, it will be logged as an anomaly and skipped
	// Reasoning: Largest uint64 is 20 characters wide, floats are not printed with too much precision either
	PerfdataNumericValuesMaxLengthDefault = int(32)

	// If a perfdata numeric value, e.g current value, min, and max is longer than this, it will be logged as an anomaly and skipped
	// Reasoning: Twice the numeric value max length, since there are seperators as well
	PerfdataThresholdsMaxLengthDefault = int(64)
)

// NagiosSpoolfileCollector scans the nagios spoolfile folder and delegates the files to its workers.
type NagiosSpoolfileCollector struct {
	quit           chan bool
	jobs           chan string
	spoolDirectory string
	workers        []*NagiosSpoolfileWorker
}

// NagiosSpoolfileCollectorFactory creates the give amount of Woker and starts them.
func NagiosSpoolfileCollectorFactory(cfg config.Config, results collector.ResultQueues,
	livestatusCacheBuilder *livestatus.CacheBuilder, fileBufferSize int, defaultTarget collector.Filterable,
) (*NagiosSpoolfileCollector, error) {
	spoolDirectory := ""
	search, found := helper.GetPreferredConfigValue(cfg, "NagiosSpoolfile.Folder", []string{"Main.NagiosSpoolfileFolder"})
	if !found {
		return nil, fmt.Errorf("Could not find a config value for Nagios Spoolfile Folder")
	}
	spoolDirectoryPtr, ok := search.(*string)
	if ok {
		spoolDirectory = *(spoolDirectoryPtr)
	} else {
		return nil, fmt.Errorf("Expected a *string value out of the config value for Nagios Spoolfile Folder")
	}

	workerAmount := 0
	search, found = helper.GetPreferredConfigValue(cfg, "NagiosSpoolfile.WorkerCount", []string{"Main.NagiosSpoolfileWorker"})
	if !found {
		return nil, fmt.Errorf("Could not find a config value for Nagios Spoolfile Worker Count")
	}
	workerAmountPtr, ok := search.(*int)
	if ok {
		workerAmount = *(workerAmountPtr)
	} else {
		return nil, fmt.Errorf("Expected a *int value out of the config value for Nagios Spoolfile Worker Count")
	}

	perfdataLabelMaxLength := PerfdataLabelMaxLengthDefault
	search, found = helper.GetPreferredConfigValue(cfg, "NagiosSpoolfile.PerfdataLabelMaxLength", []string{})
	if found {
		perfdataLabelMaxLengthPtr, ok := search.(*int)
		if ok {
			perfdataLabelMaxLength = *(perfdataLabelMaxLengthPtr)
		} else {
			return nil, fmt.Errorf("Expected a *int value out of the config value for Nagios Spoolfile Perfdata Label Max Length")
		}
	}

	perfdataUOMMaxLength := PerfdataUOMMaxLengthDefault
	search, found = helper.GetPreferredConfigValue(cfg, "NagiosSpoolfile.PerfdataUOMMaxLength", []string{})
	if found {
		perfdataUOMMaxLengthPtr, ok := search.(*int)
		if ok {
			perfdataUOMMaxLength = *(perfdataUOMMaxLengthPtr)
		} else {
			return nil, fmt.Errorf("Expected a *int value out of the config value for Nagios Spoolfile Perfdata UOM Max Length")
		}
	}

	perfdataNumericValuesMaxLength := PerfdataNumericValuesMaxLengthDefault
	search, found = helper.GetPreferredConfigValue(cfg, "NagiosSpoolfile.PerfdataNumericValuesMaxLength", []string{})
	if found {
		perfdataNumericValuesMaxLengthPtr, ok := search.(*int)
		if ok {
			perfdataNumericValuesMaxLength = *(perfdataNumericValuesMaxLengthPtr)
		} else {
			return nil, fmt.Errorf("Expected a *int value out of the config value for Nagios Spoolfile Perfdata UOM Max Length")
		}
	}

	perfdataThresholdsMaxLength := PerfdataThresholdsMaxLengthDefault
	search, found = helper.GetPreferredConfigValue(cfg, "NagiosSpoolfile.PerfdataThresholdsMaxLength", []string{})
	if found {
		perfdataThresholdsMaxLengthPtr, ok := search.(*int)
		if ok {
			perfdataThresholdsMaxLength = *(perfdataThresholdsMaxLengthPtr)
		} else {
			return nil, fmt.Errorf("Expected a *int value out of the config value for Nagios Spoolfile Perfdata Thresholds Max Length")
		}
	}

	s := &NagiosSpoolfileCollector{
		quit:           make(chan bool),
		jobs:           make(chan string, 100),
		spoolDirectory: spoolDirectory,
		workers:        make([]*NagiosSpoolfileWorker, workerAmount),
	}

	gen := NagiosSpoolfileWorkerGenerator(s.jobs, results, livestatusCacheBuilder, fileBufferSize, defaultTarget, perfdataLabelMaxLength, perfdataUOMMaxLength, perfdataNumericValuesMaxLength, perfdataThresholdsMaxLength)

	for w := range workerAmount {
		s.workers[w] = gen()
	}

	go s.run()
	return s, nil
}

// Stop stops his workers and itself.
func (s *NagiosSpoolfileCollector) Stop() {
	s.quit <- true
	<-s.quit
	for _, worker := range s.workers {
		worker.Stop()
	}
	logging.GetLogger().Debug("SpoolfileCollector stopped")
}

// Delegates the files to its workers.
func (s *NagiosSpoolfileCollector) run() {
	promServer := statistics.GetPrometheusServer()
	for {
		select {
		case <-s.quit:
			s.quit <- true
			return
		case <-time.After(IntervalToCheckDirectory):
			pause := config.IsAnyTargetOnPause()
			if pause {
				logging.GetLogger().Debugln("NagiosSpoolfileCollector in pause")
				continue
			}

			logging.GetLogger().Debug("Reading Directory: ", s.spoolDirectory)
			oldFiles, totalFiles := FilesInDirectoryOlderThanX(s.spoolDirectory, MinFileAge)
			promServer.SpoolFilesOnDisk.Set(float64(totalFiles))
			for _, currentFile := range oldFiles {
				logging.GetLogger().Debug("Reading file: ", currentFile)

				select {
				case <-s.quit:
					s.quit <- true
					return
				case s.jobs <- currentFile:
				case <-time.After(time.Duration(1) * time.Minute):
					logging.GetLogger().Warn("NagiosSpoolfileCollector: Could not write to buffer")
				}
			}
		}
	}
}

// FilesInDirectoryOlderThanX returns a list of file, of a folder, names which are older then a certain duration.
func FilesInDirectoryOlderThanX(folder string, age time.Duration) (oldFiles []string, totalFiles int) {
	files, _ := os.ReadDir(folder)
	for _, currentFile := range files {
		fsinfo, err := currentFile.Info()
		if err != nil {
			continue
		}
		if IsItTime(fsinfo.ModTime(), age) {
			oldFiles = append(oldFiles, path.Join(folder, currentFile.Name()))
		}
	}
	return oldFiles, len(files)
}

// IsItTime checks if the timestamp plus duration is in the past.
func IsItTime(timeStamp time.Time, duration time.Duration) bool {
	return time.Now().After(timeStamp.Add(duration))
}
