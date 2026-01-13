package nagflux

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"nagflux/target/file/jsontarget"
	"pkg/nagflux/collector"
	"pkg/nagflux/collector/livestatus"
	"pkg/nagflux/collector/modgearman"
	"pkg/nagflux/collector/nagflux"
	"pkg/nagflux/collector/spoolfile"
	"pkg/nagflux/config"
	"pkg/nagflux/data"
	"pkg/nagflux/logging"
	"pkg/nagflux/statistics"
	"pkg/nagflux/target/elasticsearch"
	"pkg/nagflux/target/influx"

	"github.com/kdar/factorlog"
)

// Stoppable represents every daemonlike struct which can be stopped
type Stoppable interface {
	Stop()
}

// nagfluxVersion contains the current Github-Release
const nagfluxVersion string = "v0.5.4"

var (
	log  *factorlog.FactorLog
	quit = make(chan bool)
)

func Nagflux(Build string) {
	// Parse Args
	var configPath string
	var printver bool
	flag.Usage = func() {
		fmt.Printf(`Nagflux version %s (Build: %s, %s)
Commandline Parameter:
-configPath Path to the config file. If no file path is given the default is ./config.gcfg.
-V Print version and exit

Original author: Philip Griesbacher
For further informations / bugs reports: https://github.com/ConSol-Monitoring/nagflux
`, nagfluxVersion, Build, runtime.Version())
	}
	flag.StringVar(&configPath, "configPath", "config.gcfg", "path to the config file")
	flag.BoolVar(&printver, "V", false, "print version and exit")
	flag.Parse()

	// Print version and exit
	if printver {
		fmt.Printf("%s (Build: %s, %s)\n", nagfluxVersion, Build, runtime.Version())
		os.Exit(0)
	}

	// Load config
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Printf("Can not find config file: '%s'.\n\nHelp:\n", configPath)
		flag.Usage()
		os.Exit(1)
	}
	config.InitConfig(configPath)
	cfg := config.GetConfig()

	// Create Logger
	logging.InitLogger(cfg.Log.LogFile, cfg.Log.MinSeverity)
	log = logging.GetLogger()
	log.Info(`Started Nagflux `, nagfluxVersion)
	log.Debugf("Using Config: %s", configPath)
	resultQueues := collector.ResultQueues{}
	stoppables := []Stoppable{}
	if len(cfg.Main.FieldSeparator) < 1 {
		panic("FieldSeparator is too short!")
	}
	pro := statistics.NewPrometheusServer(cfg.Monitoring.PrometheusAddress)
	pro.WatchResultQueueLength(resultQueues)
	fieldSeparator := []rune(cfg.Main.FieldSeparator)[0]

	for name, value := range cfg.InfluxDB {
		if value == nil || !(*value).Enabled {
			continue
		}
		influxConfig := (*value)
		target := data.Target{Name: name, Datatype: data.InfluxDB}
		config.StoreValue(target, false)
		resultQueues[target] = make(chan collector.Printable, cfg.Main.BufferSize)
		influx := influx.ConnectorFactory(
			resultQueues[target],
			influxConfig.Address, influxConfig.Arguments, cfg.Main.DumpFile, influxConfig.Version,
			cfg.Main.InfluxWorker, cfg.Main.MaxInfluxWorker, cfg.InfluxDBGlobal.CreateDatabaseIfNotExists,
			influxConfig.StopPullingDataIfDown, target, cfg.InfluxDBGlobal.ClientTimeout, influxConfig.HealthURL, influxConfig.AuthToken,
		)
		if influx.IsAlive() {
			stoppables = append(stoppables, influx)
		} else {
			log.Criticalf("Nagflux is disabled for InfluxDB(%s)", target.Name)
		}
		influxDumpFileCollector := nagflux.NewDumpfileCollector(resultQueues[target], cfg.Main.DumpFile, target, cfg.Main.FileBufferSize)
		waitForDumpfileCollector(influxDumpFileCollector)
		stoppables = append(stoppables, influxDumpFileCollector)
	}

	for name, value := range cfg.Elasticsearch {
		if value == nil || !(*value).Enabled {
			continue
		}
		elasticConfig := (*value)
		target := data.Target{Name: name, Datatype: data.Elasticsearch}
		resultQueues[target] = make(chan collector.Printable, cfg.Main.BufferSize)
		config.StoreValue(target, false)
		elasticsearch := elasticsearch.ConnectorFactory(
			resultQueues[target],
			elasticConfig.Address, elasticConfig.Index, cfg.Main.DumpFile, elasticConfig.Version,
			cfg.Main.InfluxWorker, cfg.Main.MaxInfluxWorker, true,
		)
		stoppables = append(stoppables, elasticsearch)
		elasticDumpFileCollector := nagflux.NewDumpfileCollector(resultQueues[target], cfg.Main.DumpFile, target, cfg.Main.FileBufferSize)
		waitForDumpfileCollector(elasticDumpFileCollector)
		stoppables = append(stoppables, elasticDumpFileCollector)
	}

	for name, value := range cfg.JSONFileExport {
		if value == nil || !(*value).Enabled {
			continue
		}
		jsonFileConfig := (*value)
		target := data.Target{Name: name, Datatype: data.JSONFile}
		resultQueues[target] = make(chan collector.Printable, cfg.Main.BufferSize)
		templateFile := jsontarget.NewJSONFileWorker(
			log, jsonFileConfig.AutomaticFileRotation,
			resultQueues[target], target, jsonFileConfig.Path,
		)
		stoppables = append(stoppables, templateFile)
	}

	// Some time for the dumpfile to fill the queue
	time.Sleep(time.Duration(100) * time.Millisecond)

	liveconnector := &livestatus.Connector{Log: log, LivestatusAddress: cfg.Livestatus.Address, ConnectionType: cfg.Livestatus.Type}
	livestatusCollector := livestatus.NewLivestatusCollector(resultQueues, liveconnector, cfg.Livestatus.Version)
	livestatusCache := livestatus.NewLivestatusCacheBuilder(liveconnector)

	for name, data := range cfg.ModGearman {
		if data == nil || !data.Enabled {
			continue
		}
		log.Infof("Mod_Gearman: %s - %s [%s]", name, data.Address, data.Queue)
		secret := modgearman.GetSecret(data.Secret, data.SecretFile)
		for range data.Worker {
			gearmanWorker := modgearman.NewGearmanWorker(data.Address,
				data.Queue,
				secret,
				resultQueues,
				livestatusCache,
			)
			stoppables = append(stoppables, gearmanWorker)
		}
	}

	log.Info("Nagios Spoolfile Folder: ", cfg.Main.NagiosSpoolfileFolder)
	nagiosCollector := spoolfile.NagiosSpoolfileCollectorFactory(
		cfg.Main.NagiosSpoolfileFolder,
		cfg.Main.NagiosSpoolfileWorker,
		resultQueues,
		livestatusCache,
		cfg.Main.FileBufferSize,
		collector.Filterable{Filter: cfg.Main.DefaultTarget},
	)

	log.Info("Nagflux Spoolfile Folder: ", cfg.Main.NagfluxSpoolfileFolder)
	nagfluxCollector := nagflux.NewNagfluxFileCollector(resultQueues, cfg.Main.NagfluxSpoolfileFolder, fieldSeparator)

	// Listen for Interrupts
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGINT)
	signal.Notify(signalChannel, syscall.SIGTERM)
	signal.Notify(signalChannel, syscall.SIGUSR1)
	go func() {
		for {
			switch <-signalChannel {
			case syscall.SIGINT, syscall.SIGTERM:
				log.Warn("Got Interrupted")
				stoppables = append(stoppables, []Stoppable{livestatusCollector, livestatusCache, nagiosCollector, nagfluxCollector}...)
				cleanUp(stoppables, resultQueues)
				quit <- true
				return
			case syscall.SIGUSR1:
				buf := make([]byte, 1<<16)
				n := runtime.Stack(buf, true)
				if n < len(buf) {
					buf = buf[:n]
				}
				log.Warnf("Got USR1 signal, logging thread dump:\n%s", buf)
			}
		}
	}()

	// wait for quit
	<-quit
}

func waitForDumpfileCollector(dump *nagflux.DumpfileCollector) {
	if dump != nil {
		for i := 0; i < 30 && dump.IsRunning; i++ {
			time.Sleep(time.Duration(2) * time.Second)
		}
	}
}

// Wait till the Performance Data is sent.
func cleanUp(itemsToStop []Stoppable, resultQueues collector.ResultQueues) {
	log.Info("Cleaning up...")
	for i := len(itemsToStop) - 1; i >= 0; i-- {
		itemsToStop[i].Stop()
	}
	for _, q := range resultQueues {
		log.Debugf("Remaining queries %d", len(q))
	}
}
