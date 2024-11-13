# Nagflux

[![GoDoc](https://godoc.org/github.com/ConSol-Monitoring/nagflux?status.svg)](https://godoc.org/github.com/ConSol-Monitoring/nagflux)
[![Build Status](https://github.com/ConSol-Monitoring/nagflux/actions/workflows/citest.yml/badge.svg)](https://github.com/ConSol-Monitoring/nagflux/actions/workflows/citest.yml)
[![License: GPL v3](https://img.shields.io/badge/License-GPL%20v2-blue.svg)](http://www.gnu.org/licenses/gpl-2.0)

(forked from github.com/griesbacher/nagflux)

A connector which transforms performancedata from Nagios/Naemon/Icinga(2) to InfluxDB/Elasticsearch

Nagflux collects data from the NagiosSpoolfileFolder and adds informations from Livestatus. This data is sent to an InfluxDB, to get displayed by Grafana. Therefor is the tool [Histou](https://github.com/ConSol-Monitoring/histou) gives you the possibility to add Templates to Grafana.

Nagflux can be seen as the process_perfdata.pl script from PNP4Nagios.

The data storage is no restricted to InfluxDB, but can be any target which support the InfluxDB
http Line Protocol. TCP and UDP servers are not supported. Ex. Victoriametrics and Telegraf work.

As they have no db concept, the check if the database exists is omitted if "db=x" is not found in the arguments.
Additionally a custom health check url can be set. It not set the default is "/ping" from InfluxDB.

## Limitations

Nagflux only provides the timestamp in milliseconds.

## Dependencies

    Golang 1.22+

## Install

    go install github.com/ConSol-Monitoring/nagflux/cmd/nagflux@latest

This typically installs a nagflux binary into `~/go/bin/nagflux`

A x86-64 Linux binary will be added to the releases.
Here the link to the latest [Release](https://github.com/ConSol-Monitoring/nagflux/releases/latest).

## Configuration

Here are some of the important config-options:

| Section       | Config-Key    | Meaning       |
| ------------- | ------------- | ------------- |
|main|NagiosSpoolfileFolder|This is the folder where nagios/icinga writes its spoolfiles. Icinga2: `/var/spool/icinga2/perfdata`|
|main|NagfluxSpoolfileFolder|In this folder you can dump files with InfluxDBs linequery syntax, the will be shipped to the InfluxDB, the timestamp has to be in ms|
|main|FieldSeperator|This char is used to separate the logical parts of the tablenames. This char has to be an char which is not allowed in one of those: host-, servicename, command, perfdata|
|main|FileBufferSize|This is the size of the buffer which is used to read files from disk, if you have huge checks or a lot of them you maybe recive error messages that your buffer is too small and that's the point to change it|
|Log|MinSeverity|INFO is default an enough for the most. DEBUG give you a lot more data but it's mostly just spamming|
|Influx "name"|Version|**1.0** - for InfluxDB 0.9+ and 2.0 earlier versions<br>**2.0** - for InfluxDB 2.0 or later versions|
|Influx "name"|Address|The URL of the InfluxDB-API|
|Influx "name"|Arguments|Here you can set your user name and password as well as the database. **The precision has to be ms!**<br> Organization & Bucket details required for InfluxDB 2.0 or later versions|
|Influx "name"|AuthToken|InfluxDB API Token with required permissions|
|Influx "name"|NastyString/NastyStringToReplace|These keys are to avoid a bug in InfluxDB and should disappear when the bug is fixed|
|Influx "name"|StopPullingDataIfDown|This is used to tell Nagflux, if this Influxdb is down to stop reading new data. That's useful if you're using spoolfiles. But if you're using gearman set this always to false because by default gearman will not buffer the data endlessly|

## Start

If the configfile is in the same folder as the executable:

    ./nagflux

else:

    ./nagflux -configPath=/path/to/config.gcfg

## Debugging

- If the InfluxDB is not available Nagflux will stop and an log entry will be written.
- If the Livestatus is not available Nagflux will just write an log entry, but additional informations can't be gathered.
- If any part of the Tablename is not valid for the InfluxDB an log entry will written and the data is writen to a file which has the same name as the logfile just with the ending '.dump-errors'. You could fix the errors by hand and copy the lines in the NagfluxSpoolfileFolder
- If the Data can't be send to the InfluxDB, Nagflux will also write them in the '.dump-errors' file, you can handle them the same way.
- If the logs are showing files are being read (in DEBUG mode) but nothing is going into InfluxDB, check the perfdata template to ensure it matches OMD format. See [Perfdata Template](https://github.com/ConSol-Monitoring/nagflux#perfdata-template) for more details.

## Dataflow

There are basically two ways for Nagflux to receive data:

- Spoolfiles: They are for useful if Nagflux is running at the same machine as Nagios
- Gearman: If you have a distributed setup, that's the way to go
  With both ways you could enrich your performance data with additional informations from livestatus. Like downtimes, notifications and so.

Targets can be:

- **InfluxDB**, that's the main target and the reason for this project.
- Elasticsearch, more a prove of concept but it worked some time ago ;)
- JSON, to parse the data by an third tool.

![Dataflow Image](https://raw.githubusercontent.com/ConSol-Monitoring/nagflux/master/doc/NagfluxDataflow.png "Nagflux Dataflow")

## OMD

Nagflux is fully integrated in [OMD-Labs](https://github.com/ConSol-Monitoring/omd), as well as Histou is. Therefor if you wanna try it out, it's maybe easier to install OMD-Labs.

## Perfdata Template

Nagflux supports a couple of Perfdata templates (see `main_test.go` for some supported formats). By default it assumes you have the [OMD formattemplate](https://github.com/ConSol-Monitoring/omd/blob/labs/packages/nagflux/skel/etc/nagflux/nagios_nagflux.cfg). If you are setting this up manually (not using OMD) please ensure your perfdata template is as follows:

### Host

    DATATYPE::HOSTPERFDATA\tTIMET::$TIMET$\tHOSTNAME::$HOSTNAME$\tHOSTPERFDATA::$HOSTPERFDATA$\tHOSTCHECKCOMMAND::$HOSTCHECKCOMMAND$

### Service

    DATATYPE::SERVICEPERFDATA\tTIMET::$TIMET$\tHOSTNAME::$HOSTNAME$\tSERVICEDESC::$SERVICEDESC$\tSERVICEPERFDATA::$SERVICEPERFDATA$\tSERVICECHECKCOMMAND::$SERVICECHECKCOMMAND$

If you are using Nagios the default templates will not work. Use the above templates
with config `host_perfdata_file_template` and `service_perfdata_file_template`, respectively.

## Demo

This Dockercontainer contains OMD and everything is preconfigured to use Nagflux/Histou/Grafana/InfluxDB: https://github.com/Griesbacher/docker-omd-grafana

## Presentations

- Here is a presentation I held about Nagflux and Histou in 2016, only in German, sorry: [Slides](http://www.slideshare.net/PhilipGriesbacher/monitoring-workshop-kiel-2016-performancedaten-visualisierung-mit-grafana-influxdb)
- That's the first one from 2015, also only in German. [Slides](https://www.netways.de/fileadmin/images/Events_Trainings/Events/OSMC/2015/Slides_2015/Grafana_meets_Monitoring_Vorstellung_einer_Komplettloesung-Philip_Griesbacher.pdf) - [Video](https://www.youtube.com/watch?v=rY6N2H0UCFQ)
