package spoolfile

import (
	"os"
	"path"
	"time"

	"pkg/nagflux/collector"
	"pkg/nagflux/collector/livestatus"
	"pkg/nagflux/config"
	"pkg/nagflux/logging"
	"pkg/nagflux/statistics"
)

const (
	// MinFileAge is the duration to wait, before the files are parsed
	MinFileAge = 3 * time.Second

	// IntervalToCheckDirectory the interval to check if there are new files
	IntervalToCheckDirectory = 1500 * time.Millisecond
)

// NagiosSpoolfileCollector scans the nagios spoolfile folder and delegates the files to its workers.
type NagiosSpoolfileCollector struct {
	quit           chan bool
	jobs           chan string
	spoolDirectory string
	workers        []*NagiosSpoolfileWorker
}

// NagiosSpoolfileCollectorFactory creates the give amount of Woker and starts them.
func NagiosSpoolfileCollectorFactory(spoolDirectory string, workerAmount int, results collector.ResultQueues,
	livestatusCacheBuilder *livestatus.CacheBuilder, fileBufferSize int, defaultTarget collector.Filterable,
) *NagiosSpoolfileCollector {
	s := &NagiosSpoolfileCollector{
		quit:           make(chan bool),
		jobs:           make(chan string, 100),
		spoolDirectory: spoolDirectory,
		workers:        make([]*NagiosSpoolfileWorker, workerAmount),
	}

	gen := NagiosSpoolfileWorkerGenerator(s.jobs, results, livestatusCacheBuilder, fileBufferSize, defaultTarget)

	for w := range workerAmount {
		s.workers[w] = gen()
	}

	go s.run()
	return s
}

// Stop stops his workers and itself.
func (s *NagiosSpoolfileCollector) Stop() {
	if s == nil {
		log := logging.GetLogger()
		log.Warnf("cannot stop Nagios File Collector, as it is nil")
		return
	}
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
