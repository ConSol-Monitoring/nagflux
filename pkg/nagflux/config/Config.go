package config

// Config Represents the config file.
// Optional arguments use pointers, if they are unspecified, they will be set to nil
type Config struct {
	Main struct {
		// This option is deprecated, use NagiosSpoolfile.Folder when possible
		NagiosSpoolfileFolder *string
		// This option is deprecated, use NagiosSpoolfile.WorkerCount when possible
		NagiosSpoolfileWorker *int
		InfluxWorker          int
		MaxInfluxWorker       int
		DumpFile              string
		// This option is deprecated, use NagfluxSpoolfile.Folder when possible
		NagfluxSpoolfileFolder *string
		FieldSeparator         string
		BufferSize             int
		FileBufferSize         int
		DefaultTarget          string
	}
	ModGearman map[string]*struct {
		Enabled    bool
		Address    string
		Queue      string
		Secret     string
		SecretFile string
		Worker     int
	}
	Log struct {
		LogFile     string
		MinSeverity string
	}
	Filter struct {
		SpoolFileLineTerms            []string
		LivestatusLineTerms           []string
		LivestatusCommentsFilter      []string
		LivestatusDowntimesFilter     []string
		LivestatusNotificationsFilter []string // filter used while querying notifications from log table
		LivestatusHostsFilter         []string // filter used while querying active host downtimes
		LivestatusServicesFilter      []string // filter used while querying active service downtimes
	}
	Monitoring struct {
		PrometheusAddress string
	}
	InfluxDBGlobal struct {
		CreateDatabaseIfNotExists bool
		NastyString               string
		NastyStringToReplace      string
		HostcheckAlias            string
		ClientTimeout             int
	}
	InfluxDB map[string]*struct {
		Enabled               bool
		Address               string
		Arguments             string
		Version               string
		StopPullingDataIfDown bool
		HealthURL             string
		AuthToken             string
	}
	Livestatus struct {
		Enabled       *bool
		Type          string
		Address       string
		MinutesToWait int
		Version       string
	}
	NagiosSpoolfile struct {
		Enabled *bool
		// This option takes predence over Main.NagiosSpoolfileFolder if set
		Folder *string
		// This option takes predence over Main.NagiosSpoolfileWorker if set
		WorkerCount *int
	}
	NagfluxSpoolfile struct {
		Enabled *bool
		// This option takes predence over Main.NagfluxSpoolfileFolder if set
		Folder *string
	}
	ElasticsearchGlobal struct {
		HostcheckAlias   string
		NumberOfShards   int
		NumberOfReplicas int
		IndexRotation    string
	}
	Elasticsearch map[string]*struct {
		Enabled bool
		Address string
		Index   string
		Version string
	}
	JSONFileExport map[string]*struct {
		Enabled               bool
		Path                  string
		AutomaticFileRotation int
	}
}
