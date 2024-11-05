package spoolfile

import (
	"os"
	"path"
	"time"

	"github.com/ConSol-Monitoring/nagflux/collector"
	"github.com/ConSol-Monitoring/nagflux/collector/livestatus"
	"github.com/ConSol-Monitoring/nagflux/config"
	"github.com/ConSol-Monitoring/nagflux/logging"
	"github.com/ConSol-Monitoring/nagflux/statistics"
)

const (
	// MinFileAge is the duration to wait, before the files are parsed
	MinFileAge = time.Duration(10) * time.Second
	// IntervalToCheckDirectory the interval to check if there are new files
	IntervalToCheckDirectory = time.Duration(5) * time.Second
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
			files, _ := os.ReadDir(s.spoolDirectory)
			promServer.SpoolFilesOnDisk.Set(float64(len(files)))
			for _, currentFile := range files {
				select {
				case <-s.quit:
					s.quit <- true
					return
				case s.jobs <- path.Join(s.spoolDirectory, currentFile.Name()):
				case <-time.After(time.Duration(1) * time.Minute):
					logging.GetLogger().Warn("NagiosSpoolfileCollector: Could not write to buffer")
				}
			}
		}
	}
}

// FilesInDirectoryOlderThanX returns a list of file, of a folder, names which are older then a certain duration.
func FilesInDirectoryOlderThanX(folder string, age time.Duration) []string {
	files, _ := os.ReadDir(folder)
	var oldFiles []string
	for _, currentFile := range files {
		fsinfo, err := currentFile.Info()
		if err != nil {
			continue
		}
		if IsItTime(fsinfo.ModTime(), age) {
			oldFiles = append(oldFiles, path.Join(folder, currentFile.Name()))
		}
	}
	return oldFiles
}

// IsItTime checks if the timestamp plus duration is in the past.
func IsItTime(timeStamp time.Time, duration time.Duration) bool {
	return time.Now().After(timeStamp.Add(duration))
}
