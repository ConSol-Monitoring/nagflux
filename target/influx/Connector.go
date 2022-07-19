package influx

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"time"

	"github.com/ConSol/nagflux/collector"
	"github.com/ConSol/nagflux/config"
	"github.com/ConSol/nagflux/data"
	"github.com/ConSol/nagflux/helper"
	"github.com/ConSol/nagflux/logging"
	"github.com/kdar/factorlog"
)

//Connector makes the basic connection to an Influxdb.
type Connector struct {
	connectionHost            string
	connectionArgs            string
	dumpFile                  string
	workers                   []*Worker
	maxWorkers                int
	jobs                      chan collector.Printable
	quit                      chan bool
	log                       *factorlog.FactorLog
	version                   string
	isAlive                   bool
	databaseExists            bool
	databaseName              string
	httpClient                http.Client
	target                    data.Target
	stopReadingDataIfDown     bool
	clientTimeout             int
	createDatabaseIfNotExists bool
	healthUrl                 string
	authToken                 string
}

//ConnectorFactory Constructor which will create some workers if the connection is established.
func ConnectorFactory(jobs chan collector.Printable, connectionHost, connectionArgs, dumpFile, version string,
	workerAmount, maxWorkers int, createDatabaseIfNotExists, stopReadingDataIfDown bool, target data.Target, clientTimeout int, healthUrl string, authToken string) *Connector {
	parsedArgs := helper.StringToMap(connectionArgs, "&", "=")
	var databaseName string
	if db, found_db := parsedArgs["db"]; found_db {
		databaseName = db
	}

	timeout := time.Duration(time.Duration(clientTimeout) * time.Second)
	transport := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	client := http.Client{Timeout: timeout, Transport: transport}
	s := &Connector{
		connectionHost: connectionHost, connectionArgs: connectionArgs, dumpFile: dumpFile,
		workers: make([]*Worker, workerAmount), maxWorkers: maxWorkers, jobs: jobs, quit: make(chan bool),
		log: logging.GetLogger(), version: version, isAlive: false, databaseExists: false, databaseName: databaseName,
		httpClient: client, target: target, stopReadingDataIfDown: stopReadingDataIfDown, clientTimeout: clientTimeout, createDatabaseIfNotExists: createDatabaseIfNotExists, healthUrl: healthUrl,
		authToken: authToken,
	}

	// InfluxDB v2
	if version == "2.0" {
		// InfluxDB OSS requires either org or orgID
		var orgInfo string
		for _, v := range []string{"org", "orgID"} {
			if o, ok := parsedArgs[v]; ok {
				orgInfo = o
				break
			}
		}
		if orgInfo == "" {
			result := helper.GetHeaders(s.httpClient, s.connectionHost+"/ping", "GET")
			if len(result) != 0 && result["X-Influxdb-Build"] == "OSS" {
				s.log.Critical("InfluxDB OSS requires Orgranization Details. Please provide either orgID or org")
				s.isAlive = false
				return s
			}
		}
		// In InfluxDB 2.0 or later versions, databases no longer exist, they are replaced by buckets.
		s.createDatabaseIfNotExists = false
		createDatabaseIfNotExists = false
	}

	if createDatabaseIfNotExists && databaseName == "" {
		s.log.Info("InfluxDB(" + target.Name + ") Database not found, db creation not possible -> createDatabaseIfNotExists switched to false")
		s.createDatabaseIfNotExists = false
		createDatabaseIfNotExists = false
	}

	// set external health check url "healthUrl":
	matched, _ := regexp.MatchString("http.*://", healthUrl)
	if matched == false {
		if healthUrl != "" {
			// make local uri global:
			s.healthUrl = connectionHost + healthUrl
		} else {
			// default for influxDB:
			s.healthUrl = connectionHost + "/ping"
		}
	}

	// if  createDatabaseIfNotExists is false, set flag databaseExists flag to true!
	if !createDatabaseIfNotExists {
		s.log.Info("InfluxDB(" + target.Name + ") createDatabaseIfNotExists false > databaseExists set permanantly to true")
		s.databaseExists = true
	}

	loginData := ""
	if pw, found_pw := parsedArgs["p"]; found_pw {
		if login, found_login := parsedArgs["u"]; found_login {
			loginData = fmt.Sprintf("p=%s&u=%s", pw, login)
		}
	}

	gen := WorkerGenerator(jobs, connectionHost+"/write?"+connectionArgs, dumpFile, version, s, target, stopReadingDataIfDown)
	if s.version == "2.0" {
		gen = WorkerGenerator(jobs, connectionHost+"/api/v2/write?"+connectionArgs, dumpFile, version, s, target, stopReadingDataIfDown)
	}
	s.TestIfIsAlive(stopReadingDataIfDown)
	if !s.isAlive && !stopReadingDataIfDown {
		s.log.Warnf("InfluxDB server(%s) is down but starting anyway due to 'stopReadingDataIfDown' = %t", target.Name, stopReadingDataIfDown)
	} else {
		if !s.isAlive {
			s.log.Info("Waiting for InfluxDB server(" + target.Name + ")")
		}
		for !s.isAlive {
			s.TestIfIsAlive(stopReadingDataIfDown)
			time.Sleep(time.Duration(5) * time.Second)
			s.log.Debugln("Waiting for InfluxDB server (" + target.Name + ")")
		}
		if s.isAlive {
			s.log.Debug("Influxdb(" + target.Name + ") is running")
		}
		if createDatabaseIfNotExists {
			s.TestDatabaseExists()
			for i := 0; i < 5 && !s.databaseExists; i++ {
				time.Sleep(time.Duration(2) * time.Second)
				if createDatabaseIfNotExists {
					s.CreateDatabase(loginData)
				}
				s.TestDatabaseExists()
			}
			if !s.databaseExists {
				s.log.Critical("InfluxDB Database(" + databaseName + ") does not exists and Nagflux was not able to create it")
			}
		}
	}

	for w := 0; w < workerAmount; w++ {
		s.workers[w] = gen(w)
	}
	go s.run()
	return s
}

//AddWorker creates a new worker
func (connector *Connector) AddWorker() {
	oldLength := connector.AmountWorkers()
	if oldLength < connector.maxWorkers {
		gen := WorkerGenerator(
			connector.jobs, connector.connectionHost+"/write?"+connector.connectionArgs,
			connector.dumpFile, connector.version, connector, connector.target, connector.stopReadingDataIfDown,
		)
		if connector.version == "2.0" {
			gen = WorkerGenerator(
				connector.jobs, connector.connectionHost+"/api/v2/write?"+connector.connectionArgs,
				connector.dumpFile, connector.version, connector, connector.target, connector.stopReadingDataIfDown,
			)
		}
		connector.workers = append(connector.workers, gen(oldLength+2))
		connector.log.Infof("Starting Worker: %d -> %d", oldLength, connector.AmountWorkers())
	}
}

//RemoveWorker stops a worker
func (connector *Connector) RemoveWorker() {
	oldLength := connector.AmountWorkers()
	if oldLength > 1 {
		lastWorkerIndex := oldLength - 1
		connector.workers[lastWorkerIndex].Stop()
		connector.workers = connector.workers[:lastWorkerIndex]
		connector.log.Infof("Stopping Worker: %d -> %d", oldLength, connector.AmountWorkers())
	}
}

//AmountWorkers current amount of workers.
func (connector Connector) AmountWorkers() int {
	return len(connector.workers)
}

//IsAlive is the database system alive.
func (connector Connector) IsAlive() bool {
	return connector.isAlive
}

//DatabaseExists does the database exist.
func (connector Connector) DatabaseExists() bool {
	return connector.databaseExists
}

//Stop the connector and its workers.
func (connector *Connector) Stop() {
	connector.quit <- true
	<-connector.quit
	connector.log.Debug("InfluxConnectorFactory stopped")
}

//Waits just for the end.
func (connector *Connector) run() {
	for {
		select {
		case <-connector.quit:
			for _, worker := range connector.workers {
				go worker.Stop()
			}
			for len(connector.workers) > 0 {
				for connector.workers[0].IsRunning == true {
					time.Sleep(time.Duration(100) * time.Millisecond)
				}
				if len(connector.workers) > 1 {
					connector.workers = connector.workers[1:]
				} else {
					connector.workers = connector.workers[:0]
				}
			}
			connector.quit <- true
			return
		}
	}
}

//TestIfIsAlive test active if the database system is alive.
func (connector *Connector) TestIfIsAlive(stopReadingDataIfDown bool) bool {
	result := helper.RequestedReturnCodeIsOK(connector.httpClient, connector.healthUrl, "GET")
	connector.isAlive = result
	connector.log.Infof("Is InfluxDB(%s) running: %t", connector.target.Name, result)
	if stopReadingDataIfDown {
		config.StoreValue(connector.target, !result)
	}
	return result
}

//TestDatabaseExists test active if the database exists.
func (connector *Connector) TestDatabaseExists() bool {

	if !connector.createDatabaseIfNotExists {
		connector.log.Debug("Skipped TestDatabaseExists:" + connector.databaseName)
		return true
	}
	resp, err := connector.httpClient.Get(connector.connectionHost + "/query?q=show%20databases&" + connector.connectionArgs)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var jsonResult ShowSeriesResult
		err := json.Unmarshal(body, &jsonResult)
		if err == nil && len(jsonResult.Results) > 0 && len(jsonResult.Results[0].Series) > 0 {
			for _, tablename := range jsonResult.Results[0].Series[0].Values {
				if len(tablename) > 0 && connector.databaseName == tablename[0] {
					connector.databaseExists = true
					return true
				}
			}
		} else {
			connector.log.Warn(err)
		}
	}
	connector.databaseExists = false
	return false
}

//CreateDatabase creates the database.
func (connector *Connector) CreateDatabase(loginData string) bool {
	host := connector.connectionHost + "/query"
	if loginData != "" {
		host += "?" + loginData + "&"
	} else {
		host += "?"
	}
	host += "q=CREATE%20DATABASE%20" + connector.databaseName

	result := helper.RequestedReturnCodeIsOK(connector.httpClient, host, "GET")
	if !result {
		connector.log.Warn("Could not create database:" + connector.databaseName)
	}
	return result

}
