package config

import (
	"bufio"
	"os"
	"testing"
)

var configFileContent = `[main]
	# This option is deprecated, use NagiosSpoolfile.Folder when possible
	NagiosSpoolfileFolder = "/var/spool/nagios"
	# This option is deprecated, use NagiosSpoolfile.WorkerCount when possible
	NagiosSpoolfileWorker = 1
	InfluxWorker = 2
	MaxInfluxWorker = 5
	DumpFile = "nagflux.dump"
	# This option is deprecated, use NagfluxSpoolfile.Folder when possible
	NagfluxSpoolfileFolder = "/var/spool/nagflux"
	FieldSeparator = "&"
	BufferSize = 10000
	FileBufferSize = 65536

[Log]
	# leave empty for stdout
	LogFile = ""
	# List of Severities https://godoc.org/github.com/kdar/factorlog#Severity
	MinSeverity = "INFO"

[Monitoring]
	# leave empty to disable
	# PrometheusAddress = ":8080"
	PrometheusAddress = ":8080"

[Livestatus]
	Enabled = true
	# tcp or file
	Type = "tcp"
	# tcp: 127.0.0.1:6557 or file /var/run/live
	Address = "127.0.0.1:6557"
	# The amount to minutes to wait for livestatus to come up, if set to 0 the detection is disabled
	MinutesToWait = 2

[NagiosSpoolfile]
    Enabled = true
	# This option takes predence over main.NagiosSpoolfileFolder if set
    Folder = "/var/spool/nagios"
	# This option takes predence over main.NagiosSpoolfileWorker if set
    WorkerCount = 1

[NagfluxSpoolfile]
    Enabled = true
	# This option takes predence over main.NagfluxSpoolfileFolder if set
    Folder = "/var/spool/nagflux"

[ModGearman "example"] #copy this block and rename it to add a second ModGearman queue
	Enabled = false
	Address = "127.0.0.1:4730"
	Queue = "perfdata"
	# Leave Secret and SecretFile empty to disable encryption
	# If both are filled the Secret will be used
	# Secret to encrypt the gearman jobs
	Secret = ""
	# Path to a file which holds the secret to encrypt the gearman jobs
	SecretFile = "/etc/mod-gearman/secret.key"
	Worker = 1

[InfluxDBGlobal]
	CreateDatabaseIfNotExists = true
	NastyString = ""
	NastyStringToReplace = ""
	HostcheckAlias = "hostcheck"
	ClientTimeout = "30"

[InfluxDB "nagflux"]
	Enabled = true
	Version = 1.0
	Address = "http://127.0.0.1:8086"
	Arguments = "precision=ms&u=root&p=root&db=nagflux"
	StopPullingDataIfDown = true

[InfluxDB "fast"]
	Enabled = true
	Version = 1.0
	Address = "http://127.0.0.1:8086"
	Arguments = "precision=ms&u=root&p=root"
	StopPullingDataIfDown = false
	HealthUrl = "/ping"

[ElasticsearchGlobal]
	HostcheckAlias = "hostcheck"
	NumberOfShards = 1
	NumberOfReplicas = 1
	# Sorts the indices "monthly" or "yearly"
	IndexRotation = "monthly"

[Elasticsearch "example"]
	Enabled = false
	Address = "http://localhost:9200"
	Index = "nagflux"
	Version = 2.1

[Filter]
    LivestatusLineTerms = "host_\\d+"`

func TestInitConfig(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "prefix")
	if err != nil {
		panic(err)
	}

	w := bufio.NewWriter(file)
	_, err = w.WriteString(configFileContent)
	w.Flush()
	if err != nil {
		panic(err)
	}
	InitConfig(file.Name())
	cfg := GetConfig()
	os.Remove(file.Name())
	if cfg.Main.InfluxWorker != 2 {
		t.Errorf("Content did not match %d != %d", cfg.Main.InfluxWorker, 2)
	}
}

func TestInitConfigFromString(t *testing.T) {
	InitConfigFromString(configFileContent)
	cfg := GetConfig()
	if cfg.Main.MaxInfluxWorker != 5 {
		t.Errorf("Content did not match %d != %d", cfg.Main.MaxInfluxWorker, 5)
	}
}
